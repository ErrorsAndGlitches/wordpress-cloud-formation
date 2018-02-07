package actions

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53domains"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

var regionFilterKey = "region-name"

// Aws handles creating AWS services. It ensures that all of the services are generated using the same profile and
// region.
type Aws struct {
	Profile string
	Region  *Region
}

func (a *Aws) Ec2Service() *ec2.EC2 {
	return ec2.New(a.session())
}

func (a *Aws) CloudFormationService() *cloudformation.CloudFormation {
	return cloudformation.New(a.session())
}

func (a *Aws) Route53() *route53.Route53 {
	return route53.New(a.session())
}

func (a *Aws) CertificateManager() *acm.ACM {
	return acm.New(a.session())
}

func (a *Aws) Route53Domains() *route53domains.Route53Domains {
	return route53domains.New(a.session())
}

func (a *Aws) Azs() []*ec2.AvailabilityZone {
	describeOutput, err := a.Ec2Service().DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			{
				Name: &regionFilterKey,
				Values: []*string{
					&a.Region.StringExpr().Literal,
				},
			},
		},
	})

	checkError(err)
	return describeOutput.AvailabilityZones
}

func (a *Aws) session() *session.Session {
	SugaredLogger().Infof("Using profile '%s' to talk to an AWS service", a.Profile)
	region := a.Region.String()

	sess, err := session.NewSession(&aws.Config{
		Region: &region,
		Credentials: credentials.NewCredentials(
			&credentials.SharedCredentialsProvider{
				Profile: a.Profile,
			},
		),
	})
	checkError(err)

	return sess
}

func checkError(e error) {
	if e != nil {
		panic(e)
	}
}

// AwsCall is a utility to make AWS service calls. It panics with a solid message on failure, otherwise just returning
// the output of making the call.
type AwsCall struct {
	Action   string
	Callable func() (interface{}, error)
}

func (awsCall *AwsCall) Output() interface{} {
	SugaredLogger().Debugf("Performing action: '%s'", awsCall.Action)
	output, err := awsCall.Callable()
	awsCall.check(output, err)
	return output
}

func (awsCall *AwsCall) check(output interface{}, err error) {
	if err != nil {
		panic(fmt.Sprintf(
			"Error occurred performing action '%s'\n  Output: '%s'\n  Error: '%s'\n",
			awsCall.Action,
			output,
			err,
		))
	} else {
		SugaredLogger().Debugf("AWS Call SUCCESS. Action: '%s'. Output: '%s'", awsCall.Action, output)
	}
}
