package actions

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/route53domains"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

type ChangeResourceRecordStatus struct {
	Route53Domains *route53domains.Route53Domains
	OperationId    string
}

func (dn *ChangeResourceRecordStatus) PrintStatus() {
	opStatus := (&AwsCall{
		Action: fmt.Sprintf(
			"Querying status of resource record operation with op id: '%s'",
			dn.OperationId,
		),
		Callable: func() (interface{}, error) {
			return dn.Route53Domains.GetOperationDetail(&route53domains.GetOperationDetailInput{
				OperationId: &dn.OperationId,
			})
		},
	}).Output().(*route53domains.GetOperationDetailOutput).Status

	SugaredLogger().Infof("Current status: '%s'", *opStatus)
}
