package actions

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"fmt"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

var upsertAction = route53.ChangeActionUpsert
var aliasType = route53.RRTypeA
var noHealthEvaluation = false

type AliasRecord struct {
	route53             *route53.Route53
	domainName          string
	hostedZoneId        string
	elbDomainName       string
	elbHostedZone       string
	wordPressSubDomains []string
}

func NewAliasRecord(
	route53 *route53.Route53, domainName string, hostedZoneId string, elbDomainName string, elbHostedZone string,
	wordPressSubDomains []string,
) *AliasRecord {
	return &AliasRecord{
		route53:             route53,
		domainName:          domainName,
		hostedZoneId:        hostedZoneId,
		elbDomainName:       elbDomainName,
		elbHostedZone:       elbHostedZone,
		wordPressSubDomains: wordPressSubDomains,
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

			var changes[]*route53.Change
			for _, subdomain := range ar.wordPressSubDomains {
				changes = append(changes, ar.route53ChangeForSubdomain(subdomain))
			}

			return ar.route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53.ChangeBatch{
					Changes: changes,
					Comment: &comment,
				},
				HostedZoneId: &ar.hostedZoneId,
			})
		},
	}).Output().(*route53.ChangeResourceRecordSetsOutput).ChangeInfo

	SugaredLogger().Infof("Status of request: '%s'", *changeInfo.Status)
	SugaredLogger().Infof("Operation id: '%s'", *changeInfo.Id)
}

func (ar *AliasRecord) route53ChangeForSubdomain(subdomain string) *route53.Change {
	aliasValue := fmt.Sprintf("dualstack.%s", ar.elbDomainName)
	recordName := fmt.Sprintf("%s.%s", subdomain, ar.domainName)

	return &route53.Change{
		Action: &upsertAction,
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: &recordName,
			Type: &aliasType,
			AliasTarget: &route53.AliasTarget{
				DNSName:              &aliasValue,
				EvaluateTargetHealth: &noHealthEvaluation,
				HostedZoneId:         &ar.elbHostedZone,
			},
		},
	}
}
