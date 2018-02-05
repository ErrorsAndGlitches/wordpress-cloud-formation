package services

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"fmt"
	. "github.com/ColectivaLegal/sms-alert-system-cloud-formation/models"
)

type AliasRecord struct {
	route53       *route53.Route53
	domainName    string
	hostedZoneId  string
	elbDomainName string
	elbHostedZone string
}

func NewAliasRecord(
	route53 *route53.Route53, domainName string, hostedZoneId string, elbDomainName string, elbHostedZone string,
) *AliasRecord {
	return &AliasRecord{
		route53:       route53,
		domainName:    domainName,
		hostedZoneId:  hostedZoneId,
		elbDomainName: elbDomainName,
		elbHostedZone: elbHostedZone,
	}
}

func (ar *AliasRecord) Create() {
	changeInfo := (&AwsCall{
		Action: fmt.Sprintf(
			"Create alias record in Hosted Zone for domain name '%s' to '%s'",
			ar.domainName, ar.elbDomainName,
		),
		Callable: func() (interface{}, error) {
			comment := "Adding alias from domain name to ELB domain name"
			upsertAction := route53.ChangeActionUpsert
			aliasType := route53.RRTypeA
			noHealthEvaluation := false
			aliasValue := fmt.Sprintf("dualstack.%s", ar.elbDomainName)

			return ar.route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action: &upsertAction,
							ResourceRecordSet: &route53.ResourceRecordSet{
								Name: &ar.domainName,
								Type: &aliasType,
								AliasTarget: &route53.AliasTarget{
									DNSName:              &aliasValue,
									EvaluateTargetHealth: &noHealthEvaluation,
									HostedZoneId:         &ar.elbHostedZone,
								},
							},
						},
					},
					Comment: &comment,
				},
				HostedZoneId: &ar.hostedZoneId,
			})
		},
	}).Output().(*route53.ChangeResourceRecordSetsOutput).ChangeInfo

	SugaredLogger().Infof("Status of request: '%s'", *changeInfo.Status)
	SugaredLogger().Infof("Operation id: '%s'", *changeInfo.Id)
}
