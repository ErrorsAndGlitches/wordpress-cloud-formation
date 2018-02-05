package services

import (
	"github.com/aws/aws-sdk-go/service/route53domains"
	"fmt"
	"github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

type DomainRegistrationInProgressError struct {
	DomainName string
}

func (err *DomainRegistrationInProgressError) Error() string {
	return fmt.Sprintf("Domain registration still in progress for '%s'", err.DomainName)
}

type DomainNames struct {
	Route53Domains *route53domains.Route53Domains
	DomainName     string
	FirstName      string
	LastName       string
	Email          string
	ContactType    string
	Organization   string
	StreetAddress  string
	City           string
	State          string
	ZipCode        string
	PhoneNumber    string
}

func (dn *DomainNames) Execute() {
	dn.panicIfUnavailable()
	operationId := dn.registerDomain()
	models.SugaredLogger().Infof("Registration operation id: '%s'", *operationId)
}

func (dn *DomainNames) panicIfUnavailable() {
	availability := (&AwsCall{
		Action: fmt.Sprintf("Checking for availability of domain name '%s'", dn.DomainName),
		Callable: func() (interface{}, error) {
			return dn.Route53Domains.CheckDomainAvailability(&route53domains.CheckDomainAvailabilityInput{
				DomainName: &dn.DomainName,
			})
		},
	}).Output().(*route53domains.CheckDomainAvailabilityOutput).Availability

	if route53domains.DomainAvailabilityAvailable != *availability {
		panic(fmt.Sprintf("Domain name is not available. Status is: '%s'", *availability))
	}
}

func (dn *DomainNames) registerDomain() *string {
	autoRenew := true
	var registrationDurationYears int64 = 1

	return (&AwsCall{
		Action: fmt.Sprintf("Registering domain name '%s'", dn.DomainName),
		Callable: func() (interface{}, error) {
			usCountryCode := route53domains.CountryCodeUs

			contactDetail := &route53domains.ContactDetail{
				CountryCode:      &usCountryCode,
				FirstName:        &dn.FirstName,
				LastName:         &dn.LastName,
				Email:            &dn.Email,
				ContactType:      &dn.ContactType,
				OrganizationName: &dn.Organization,
				AddressLine1:     &dn.StreetAddress,
				City:             &dn.City,
				State:            &dn.State,
				ZipCode:          &dn.ZipCode,
				PhoneNumber:      &dn.PhoneNumber,
			}

			return dn.Route53Domains.RegisterDomain(&route53domains.RegisterDomainInput{
				AdminContact:      contactDetail,
				AutoRenew:         &autoRenew,
				DomainName:        &dn.DomainName,
				DurationInYears:   &registrationDurationYears,
				RegistrantContact: contactDetail,
				TechContact:       contactDetail,
			})
		},
	}).Output().(*route53domains.RegisterDomainOutput).OperationId
}
