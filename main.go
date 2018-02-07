package main

import (
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/cli"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/actions"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs"
	. "github.com/crewjam/go-cloudformation"
	"github.com/urfave/cli"
	"os"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/acm"
	"strings"
)

var actionSuccess error = nil
var wordPressSeparator = ":"

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Usage = "create and update the Colectiva Alert System Cloud Formation template and more"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{ProfileCliOpt.Flag(), StageCliOpt.Flag(), RegionCliOpt.Flag()}

	app.Commands = []cli.Command{
		{
			Name:  "register-domain-name",
			Usage: "Register a domain name with Route 53 Domains - be sure to differentiate gamma/prod",
			Flags: []cli.Flag{
				DomainCliOpt.Flag(), FirstNameCliOpt.Flag(), LastNameCliOpt.Flag(), PhoneNumberCliOpt.Flag(),
				EmailCliOpt.Flag(), ContactTypeCliOpt.Flag(), OrgNameCliOpt.Flag(), StreetAddressCliOpt.Flag(),
				CityCliOpt.Flag(), StateCliOpt.Flag(), ZipCodeCliOpt.Flag(),
			},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{
						&DomainCliOpt, &FirstNameCliOpt, &LastNameCliOpt, &PhoneNumberCliOpt, &EmailCliOpt,
						&OrgNameCliOpt, &StreetAddressCliOpt, &CityCliOpt, &StateCliOpt, &ZipCodeCliOpt,
					},
					func() {
						(&DomainNames{
							Route53Domains: (&CliModels{Context: c}).Aws().Route53Domains(),
							DomainName:     DomainCliOpt.Value(c),
							FirstName:      FirstNameCliOpt.Value(c),
							LastName:       LastNameCliOpt.Value(c),
							Email:          EmailCliOpt.Value(c),
							ContactType:    ContactTypeCliOpt.ValueOrDefault(c, DefaultContactType),
							Organization:   OrgNameCliOpt.Value(c),
							StreetAddress:  StreetAddressCliOpt.Value(c),
							City:           CityCliOpt.Value(c),
							State:          StateCliOpt.Value(c),
							ZipCode:        ZipCodeCliOpt.Value(c),
							PhoneNumber:    PhoneNumberCliOpt.Value(c),
						}).Execute()
					},
				)
			},
		},
		{
			Name:  "print-record-status",
			Usage: "Print the status of a Route 53 resource record set operation",
			Flags: []cli.Flag{OperationIdCliOpt.Flag()},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&OperationIdCliOpt},
					func() {
						(&ChangeResourceRecordStatus{
							Route53Domains: (&CliModels{Context: c}).Aws().Route53Domains(),
							OperationId:    OperationIdCliOpt.Value(c),
						}).PrintStatus()
					},
				)
			},
		},
		{
			Name:  "describe-hosted-zone",
			Usage: "Print the hosted zone given the domain name",
			Flags: []cli.Flag{DomainCliOpt.Flag()},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&DomainCliOpt},
					func() {
						(&HostedZone{
							Route53:    (&CliModels{Context: c}).Aws().Route53(),
							DomainName: DomainCliOpt.Value(c),
						}).PrintHostedZone()
					},
				)
			},
		},
		{
			Name:  "setup-ssl",
			Usage: "Request ownership for the given domain and set up an SSL certificate",
			Flags: []cli.Flag{
				DomainCliOpt.Flag(),
				HostedZoneIdCliOpt.Flag(),
			},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&StageCliOpt, &DomainCliOpt, &HostedZoneIdCliOpt},
					func() {
						cliModels := CliModels{Context: c}
						(&SslCertificateRequest{
							CertManager:  cliModels.Aws().CertificateManager(),
							Route53:      cliModels.Aws().Route53(),
							DomainName:   DomainCliOpt.Value(c),
							HostedZoneId: HostedZoneIdCliOpt.Value(c),
						}).Execute()
					},
				)
			},
		},
		{
			Name:  "describe-ssl",
			Usage: "Describe the SSL certificate",
			Flags: []cli.Flag{
				SslArnCliOpt.Flag(),
			},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&SslArnCliOpt},
					func() {
						sslCertArn := SslArnCliOpt.Value(c)
						certDetail := (&AwsCall{
							Action: "Describe Aws Certificate Manager certificate",
							Callable: func() (interface{}, error) {
								return (&CliModels{
									Context: c,
								}).Aws().CertificateManager().DescribeCertificate(&acm.DescribeCertificateInput{
									CertificateArn: &sslCertArn,
								})
							},
						}).Output().(*acm.DescribeCertificateOutput).Certificate

						SugaredLogger().Infow(
							"Certification status",
							"Cert ARN", *certDetail.CertificateArn,
							"Status", *certDetail.Status,
						)
					},
				)
			},
		},
		{
			Name:  "cf-service",
			Usage: "CloudFormation operations on the service stack",
			Subcommands: (&CloudFormationSubCommand{
				writeFlags:        []cli.Flag{StageCliOpt.Flag(), WordPressSubDomainsOpt.Flag()},
				writeRequiredOpts: []StringCliOption{&StageCliOpt, &WordPressSubDomainsOpt},
				createFlags: []cli.Flag{
					DomainCliOpt.Flag(), DbPasswordCliOpt.Flag(), SslArnCliOpt.Flag(), Ec2KeyNameCliOpt.Flag(),
					WordPressSubDomainsOpt.Flag(),
				},
				createRequiredOpts: []StringCliOption{
					&StageCliOpt, &DbPasswordCliOpt, &DomainCliOpt, &SslArnCliOpt, &WordPressSubDomainsOpt,
					&Ec2KeyNameCliOpt,
				},
				stackInfo: func(context *cli.Context) *StackInfo {
					return ServiceStackInfo((&CliModels{Context: context}).AlertSysConfig())
				},
				templateCreator: func(context *cli.Context) *Template {
					t := NewTemplate()
					cliModels := CliModels{Context: context}

					(&ServiceResources{
						Template:            t,
						Config:              cliModels.AlertSysConfig(),
						AZs:                 cliModels.Aws().Azs(),
						WordPressSubDomains: strings.Split(WordPressSubDomainsOpt.Value(context), wordPressSeparator),
					}).AddToTemplate()

					return t
				},
				parameters: func(context *cli.Context) []*cloudformation.Parameter {
					return (&ServiceParameters{
						Config: (&CliModels{Context: context}).AlertSysConfig(),
					}).CloudFormationParameters(
						DbPasswordCliOpt.Value(context),
						DomainCliOpt.Value(context),
						SslArnCliOpt.Value(context),
						Ec2KeyNameCliOpt.Value(context),
					)
				},
			}).SubCommands(),
		},
		{
			Name:  "create-elb-alias",
			Usage: "Create an alias from the domain name to the ELB public domain name",
			Flags: []cli.Flag{
				DomainCliOpt.Flag(), HostedZoneIdCliOpt.Flag(), ElbDomainNameCliOpt.Flag(), ElbHostedZoneCliOpt.Flag(),
				WordPressSubDomainsOpt.Flag(),
			},
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{
						&DomainCliOpt, &HostedZoneIdCliOpt, &ElbDomainNameCliOpt, &ElbHostedZoneCliOpt,
						&WordPressSubDomainsOpt,
					},
					func() {
						NewAliasRecord(
							(&CliModels{Context: c}).Aws().Route53(),
							DomainCliOpt.Value(c),
							HostedZoneIdCliOpt.Value(c),
							ElbDomainNameCliOpt.Value(c),
							ElbHostedZoneCliOpt.Value(c),
							strings.Split(WordPressSubDomainsOpt.Value(c), wordPressSeparator),
						).Create()
					},
				)
			},
		},
	}

	app.Run(os.Args)
}

type CloudFormationSubCommand struct {
	writeFlags         []cli.Flag
	writeRequiredOpts  []StringCliOption
	createFlags        []cli.Flag
	createRequiredOpts []StringCliOption
	stackInfo          func(context *cli.Context) *StackInfo
	templateCreator    func(context *cli.Context) *Template
	parameters         func(context *cli.Context) []*cloudformation.Parameter
}

func (cfSubCmd *CloudFormationSubCommand) SubCommands() []cli.Command {
	return []cli.Command{
		{
			Name:  "write",
			Usage: "Write the cloud formation stack to a local file - useful for debugging",
			Flags: cfSubCmd.writeFlags,
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					cfSubCmd.writeRequiredOpts,
					func() {
						(&CliModels{Context: c}).CloudFormationClient().WriteCloudFormationJsonTemplate(
							cfSubCmd.stackInfo(c).TemplateFileName(),
							cfSubCmd.templateCreator(c),
						)
					},
				)
			},
		},
		{
			Name:  "create",
			Usage: "Create the cloud formation stack",
			Flags: cfSubCmd.createFlags,
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					cfSubCmd.createRequiredOpts,
					func() {
						(&CliModels{Context: c}).CloudFormationClient().CreateCloudFormationStack(
							cfSubCmd.stackInfo(c),
							cfSubCmd.templateCreator(c),
							cfSubCmd.parameters(c),
						)
					},
				)
			},
		},
		{
			Name:  "update",
			Usage: "Update the cloud formation stack",
			Flags: cfSubCmd.createFlags,
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					cfSubCmd.createRequiredOpts,
					func() {
						(&CliModels{Context: c}).CloudFormationClient().UpdateCloudFormationStack(
							cfSubCmd.stackInfo(c),
							cfSubCmd.templateCreator(c),
							cfSubCmd.parameters(c),
						)
					},
				)
			},
		},
		{
			Name:  "describe",
			Usage: "Describe the cloud formation stack after creation",
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&StageCliOpt},
					func() { (&CliModels{Context: c}).CloudFormationClient().DescribeCloudFormationStack(cfSubCmd.stackInfo(c)) },
				)
			},
		},
		{
			Name:  "delete",
			Usage: "Delete the cloud formation stack",
			Action: func(c *cli.Context) error {
				return runIfRequiredOptions(
					c,
					[]StringCliOption{&StageCliOpt},
					func() { (&CliModels{Context: c}).CloudFormationClient().DeleteCloudFormationStack(cfSubCmd.stackInfo(c)) },
				)
			},
		},
	}
}

func runIfRequiredOptions(c *cli.Context, requiredOpts []StringCliOption, action func()) error {
	for _, opt := range requiredOpts {
		if opt.IsAbsent(c) {
			return opt.ExitError()
		}
	}

	action()
	return actionSuccess
}
