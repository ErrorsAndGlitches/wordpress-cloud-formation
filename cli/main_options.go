package cli

import (
	"fmt"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	"github.com/aws/aws-sdk-go/service/route53domains"
)

var DefaultContactType = route53domains.ContactTypeAssociation

// Global Options - do not re-use the short options
var ProfileCliOpt = GlobalStringCliOption{&StringCliOptionImpl{
	LongOpt:  "profile",
	ShortOpt: "p",
	Usage:    fmt.Sprintf("AWS profile to use. Default: '%s'", defaultProfile),
}}

var StageCliOpt = GlobalStringCliOption{&StringCliOptionImpl{
	LongOpt:  "stage",
	ShortOpt: "s",
	Usage:    fmt.Sprintf("Stage to use: %s", []Stage{GammaStage, ProdStage}),
}}

var RegionCliOpt = GlobalStringCliOption{&StringCliOptionImpl{
	LongOpt:  "region",
	ShortOpt: "r",
	Usage:    fmt.Sprintf("Region to use. Default: %s. Choices: %s", DefaultRegion, []Region{UsEast1, UsWest2}),
}}

// Command options - the short options can be reused for different commands
var DbPasswordCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "db-password",
	ShortOpt: "b",
	Usage:    "Password to use for the mysql database",
}}

var TwilioUserCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "twilio-user",
	ShortOpt: "t",
	Usage:    "Twilio account user name",
}}

var TwilioPasswordCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "twilio-password",
	ShortOpt: "w",
	Usage:    "Twilio account password",
}}

var TwilioPhoneCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "twilio-phone-number",
	ShortOpt: "n",
	Usage:    "Twilio account phone number",
}}

var PlayFrameworkSecretCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "play-framework-secret",
	ShortOpt: "f",
	Usage:    "A secret for the Play Framework to use. It can be anything.",
}}

var DomainCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "domain-name",
	ShortOpt: "d",
	Usage:    "Domain name to request ownership of e.g. your-domain-name-gamma.com",
}}

var HostedZoneIdCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "hosted-zone-id",
	ShortOpt: "z",
	Usage:    "Id of the hosted zone for the domain name",
}}

var SslArnCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "ssl-arn",
	ShortOpt: "a",
	Usage:    "The AWS ARN of the SSL certificate created by AWS Certificate Manager",
}}

var ElbDomainNameCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "elb-domain-name",
	ShortOpt: "e",
	Usage:    "Domain name of the Elastic Load Balancer",
}}

var ElbHostedZoneCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "elb-hosted-zone",
	ShortOpt: "n",
	Usage:    "Hosted zone of the Elastic Load Balancer",
}}

// For registering domain name
var FirstNameCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "first-name",
	ShortOpt: "f",
	Usage:    "First name of domain name admin",
}}

var LastNameCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "last-name",
	ShortOpt: "l",
	Usage:    "Last name of domain name admin",
}}

var ContactTypeCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "contact-type",
	ShortOpt: "c",
	Usage:    fmt.Sprintf("Contact type of domain name admin. Default: %s", DefaultContactType),
}}

var EmailCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "email",
	ShortOpt: "e",
	Usage:    "Email of domain name admin",
}}

var OrgNameCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "organization-name",
	ShortOpt: "o",
	Usage:    "Organization name",
}}

var StreetAddressCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "street-address",
	ShortOpt: "t",
	Usage:    "Street address of the organization",
}}

var CityCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "city",
	ShortOpt: "y",
	Usage:    "City of the organization's address",
}}

var StateCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "state",
	ShortOpt: "x",
	Usage:    "State of the organization's address",
}}

var ZipCodeCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "zip-code",
	ShortOpt: "z",
	Usage:    "Zip code of the organization's address",
}}

var PhoneNumberCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "phone-number",
	ShortOpt: "n",
	Usage:    "Phone number of the organization",
}}

var OperationIdCliOpt = CommandStringCliOption{&StringCliOptionImpl{
	LongOpt:  "operation-id",
	ShortOpt: "i",
	Usage:    "Operation ID obtained from registering a domain name",
}}
