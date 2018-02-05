package cli

import (
	"github.com/urfave/cli"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/services"
)

var defaultProfile = "default"

type CliModels struct {
	Context *cli.Context
}

func (cm *CliModels) CloudFormationClient() *CloudFormationClient {
	return &CloudFormationClient{
		CloudFormationService: cm.Aws().CloudFormationService(),
	}
}

func (cm *CliModels) AlertSysConfig() *AlertSysConfig {
	config := AlertSysConfig{
		Region: cm.awsRegion(),
		Stage:  StageFromString(StageCliOpt.Value(cm.Context)),
	}

	return &config
}

func (cm *CliModels) Aws() *Aws {
	return &Aws{
		Profile: cm.awsProfile(),
		Region:  cm.awsRegion(),
	}
}

func (cm *CliModels) awsProfile() string {
	return ProfileCliOpt.ValueOrDefault(cm.Context, defaultProfile)
}

func (cm *CliModels) awsRegion() *Region {
	if RegionCliOpt.IsAbsent(cm.Context) {
		return &DefaultRegion
	} else {
		return RegionFromString(RegionCliOpt.Value(cm.Context))
	}
}
