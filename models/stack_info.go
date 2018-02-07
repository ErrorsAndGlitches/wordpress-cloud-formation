package models

import "fmt"

const serviceStackTemplateFileName = "./wp-service-cf-%s.json"
const serviceStackName = "wp-system-service"

func ServiceStackInfo(config *TemplateConfig) *StackInfo {
	return &StackInfo{
		config:        config,
		baseStackName: serviceStackName,
		baseFileName:  serviceStackTemplateFileName,
	}
}

type StackInfo struct {
	config        *TemplateConfig
	baseStackName string
	baseFileName  string
}

func (stackName *StackInfo) StackName() *string {
	name := fmt.Sprintf("%s-%s", stackName.baseStackName, stackName.config.Stage)
	return &name
}

func (stackName *StackInfo) TemplateFileName() string {
	return fmt.Sprintf(stackName.baseFileName, stackName.config.Stage)
}
