package models

import "fmt"

const serviceStackTemplateFileName = "./alert-sys-service-cf-%s.json"
const serviceStackName = "colectiva-alert-system-service"

func ServiceStackInfo(config *AlertSysConfig) *StackInfo {
	return &StackInfo{
		config:        config,
		baseStackName: serviceStackName,
		baseFileName:  serviceStackTemplateFileName,
	}
}

type StackInfo struct {
	config        *AlertSysConfig
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
