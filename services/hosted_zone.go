package services

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"fmt"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

var oneItem = "1"

type HostedZone struct {
	Route53    *route53.Route53
	DomainName string
}

func (hz *HostedZone) PrintHostedZone() {
	// in the registration of the domain name, the domain name receives a period at the end.
	domainName := fmt.Sprintf("%s.", hz.DomainName)

	hostedZones := (&AwsCall{
		Action: fmt.Sprintf("Querying Route 53 for the hosted zone associated with domain name '%s'", hz.DomainName),
		Callable: func() (interface{}, error) {
			return hz.Route53.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{
				DNSName:  &domainName,
				MaxItems: &oneItem,
			})
		},
	}).Output().(*route53.ListHostedZonesByNameOutput).HostedZones

	if len(hostedZones) == 0 {
		panic(fmt.Sprintf("Could not find any hosted zones associated with domain name '%s'", hz.DomainName))
	} else if len(hostedZones) > 1 {
		panic(fmt.Sprintf("Found more than one hosted zone associated domain name '%s'", hz.DomainName))
	}

	SugaredLogger().Infof("Domain name '%s' hosted zone: '%s'", hz.DomainName, *hostedZones[0].Id)
}
