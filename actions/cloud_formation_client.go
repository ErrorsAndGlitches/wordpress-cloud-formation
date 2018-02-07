package actions

import (
	. "github.com/crewjam/go-cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"os"
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
)

var prefix = ""
var indent = "    "
var replaceAll = -1
var iamCapability = "CAPABILITY_IAM"

type CloudFormationClient struct {
	CloudFormationService *cloudformation.CloudFormation
}

func (client *CloudFormationClient) WriteCloudFormationJsonTemplate(filename string, template *Template) {
	file, err := os.Create(filename)
	checkError(err)
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	writer.WriteString(*templateString(template))
	models.SugaredLogger().Infof("Wrote cloud formation template to: '%s'", filename)
}

func (client *CloudFormationClient) CreateCloudFormationStack(
	stackInfo *models.StackInfo, template *Template, parameters []*cloudformation.Parameter,
) {
	input := &cloudformation.CreateStackInput{
		StackName:    stackInfo.StackName(),
		Parameters:   parameters,
		TemplateBody: templateString(template),
		Capabilities: []*string{&iamCapability},
	}

	(&AwsCall{
		Action: "Create CloudFormation stack",
		Callable: func() (interface{}, error) {
			return client.CloudFormationService.CreateStack(input)
		},
	}).Output()

	models.SugaredLogger().Infof("Stack creation in progress: %s", *stackInfo.StackName())
}

func (client *CloudFormationClient) UpdateCloudFormationStack(
	stackInfo *models.StackInfo, template *Template, parameters []*cloudformation.Parameter,
) {
	input := &cloudformation.UpdateStackInput{
		StackName:    stackInfo.StackName(),
		Parameters:   parameters,
		TemplateBody: templateString(template),
		Capabilities: []*string{&iamCapability},
	}

	(&AwsCall{
		Action: "Update CloudFormation stack",
		Callable: func() (interface{}, error) {
			return client.CloudFormationService.UpdateStack(input)
		},
	}).Output()

	models.SugaredLogger().Infof("Stack update in progress: %s", *stackInfo.StackName())
}

func (client *CloudFormationClient) DescribeCloudFormationStack(stackInfo *models.StackInfo) {
	output := (&AwsCall{
		Action: "Describe CloudFormation stack",
		Callable: func() (interface{}, error) {
			return client.CloudFormationService.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: stackInfo.StackName(),
			})
		},
	}).Output().(*cloudformation.DescribeStacksOutput)

	models.SugaredLogger().Infof("%s", output)
}

func (client *CloudFormationClient) DeleteCloudFormationStack(stackInfo *models.StackInfo) {
	(&AwsCall{
		Action: "Delete CloudFormation stack",
		Callable: func() (interface{}, error) {
			return client.CloudFormationService.DeleteStack(&cloudformation.DeleteStackInput{
				StackName: stackInfo.StackName(),
			})
		},
	}).Output()

	models.SugaredLogger().Infof("Deleted stack: %s", *stackInfo.StackName())
}

func templateString(t *Template) *string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent(prefix, indent)
	encoder.Encode(t)

	// this is to get around a bug in the cloud formation go library: https://github.com/crewjam/go-cloudformation/issues/26
	fixedTemplate := strings.Replace(buffer.String(), " (SecurityGroupIngress only)", "", replaceAll)
	return &fixedTemplate
}
