package actions

import (
	"github.com/aws/aws-sdk-go/service/acm"
	"fmt"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/avast/retry-go"
	"time"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

const sslMaxAttempts = 5

var defaultTtlSeconds int64 = 300

type MissingResourceRecordError struct {
	DomainName string
}

func (err *MissingResourceRecordError) Error() string {
	return fmt.Sprintf("Could not find ResourceRecord for '%s'", err.DomainName)
}

type SslCertificateRequest struct {
	CertManager  *acm.ACM
	Route53      *route53.Route53
	DomainName   string
	HostedZoneId string
}

func (sslCerts *SslCertificateRequest) Execute() {
	certArn := sslCerts.createCert()
	record := sslCerts.resourceRecord(certArn)
	sslCerts.updateHostedZone(record)
	SugaredLogger().Infof("Certificate ARN: '%s'", *certArn)
}

func (sslCerts *SslCertificateRequest) createCert() *string {
	dnsValidationMethod := acm.ValidationMethodDns
	allSubDomains := fmt.Sprintf("*.%s", sslCerts.DomainName)
	certArn := (&AwsCall{
		Action: fmt.Sprintf("Requesting SSL Certificate for: '%s'", sslCerts.DomainName),
		Callable: func() (interface{}, error) {
			return sslCerts.CertManager.RequestCertificate(&acm.RequestCertificateInput{
				DomainName:              &allSubDomains,
				ValidationMethod:        &dnsValidationMethod,
				SubjectAlternativeNames: []*string{&sslCerts.DomainName},
			})
		},
	}).Output().(*acm.RequestCertificateOutput).CertificateArn

	SugaredLogger().Infof("Created certificate with ARN: '%s'", *certArn)
	return certArn
}

func (sslCerts *SslCertificateRequest) resourceRecord(certArn *string) *acm.ResourceRecord {
	var resourceRecord *acm.ResourceRecord

	resultError := retry.Do(
		func() error {
			domainValidationOpt := (&AwsCall{
				Action: fmt.Sprintf(
					"Retrieving resource record for DNS certification for certificate '%s'",
					*certArn,
				),
				Callable: func() (interface{}, error) {
					return sslCerts.CertManager.DescribeCertificate(&acm.DescribeCertificateInput{
						CertificateArn: certArn,
					})
				},
			}).Output().(*acm.DescribeCertificateOutput).Certificate.DomainValidationOptions[0]

			if domainValidationOpt.ResourceRecord == nil {
				SugaredLogger().Infof("Unable to get the ResourceRecord for certificate '%s'. Retrying.", *certArn)
				return &MissingResourceRecordError{DomainName: sslCerts.DomainName}
			}

			resourceRecord = domainValidationOpt.ResourceRecord
			return nil
		},
		retry.Delay(4),
		retry.Units(time.Second),
		retry.Attempts(sslMaxAttempts),
	)

	if resultError != nil {
		panic("Unable to retrieve the ResourceRecord! This means the hosted zone was not updated!")
	}

	return resourceRecord
}

func (sslCerts *SslCertificateRequest) updateHostedZone(record *acm.ResourceRecord) {
	comment := "Adding DNS validation CNAME"
	upsertAction := route53.ChangeActionUpsert

	changeOutput := (&AwsCall{
		Action: comment,
		Callable: func() (interface{}, error) {
			return sslCerts.Route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action: &upsertAction,
							ResourceRecordSet: &route53.ResourceRecordSet{
								Name:            record.Name,
								Type:            record.Type,
								TTL:             &defaultTtlSeconds,
								ResourceRecords: []*route53.ResourceRecord{{Value: record.Value}},
							},
						},
					},
					Comment: &comment,
				},
				HostedZoneId: &sslCerts.HostedZoneId,
			})
		},
	}).Output().(*route53.ChangeResourceRecordSetsOutput)

	SugaredLogger().Infof("Current status of DNS update is: '%s'", *changeOutput.ChangeInfo.Status)
}
