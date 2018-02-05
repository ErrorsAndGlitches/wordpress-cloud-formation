package models

import (
	"fmt"
	. "github.com/crewjam/go-cloudformation"
)

// A configuration needed to build the cloud formation template.
type AlertSysConfig struct {
	*Stage
	Region *Region
}

// A Region is the AWS region. Instead of creating a region, consider using one of the pre-defined regions in this
// package.
type Region struct {
	name string
}

func (region *Region) String() string {
	return region.name
}

func (region *Region) StringExpr() *StringExpr {
	return String(region.name)
}

func RegionFromString(regionName string) *Region {
	for _, region := range []Region{UsEast1, UsWest2} {
		if region.name == regionName {
			return &region
		}
	}

	panic(fmt.Sprintf("Region '%s' is not valid. Choose from: '%s'", regionName, []Region{UsEast1, UsWest2}))
}

// A Stage is the deployment stage in the pipeline e.g. Alpha, Beta, Gamma, Prod. Instead of creating a region, consider
// using one of the pre-defined regions in this package.
type Stage struct {
	name string
}

var gammaStageName = "Gamma"
var prodStageName = "Prod"

func (stage *Stage) String() string {
	return stage.name
}

func (stage *Stage) StringExpr() *StringExpr {
	return String(stage.name)
}

func (stage *Stage) CfName(basename string) string {
	return fmt.Sprintf("%s%s", basename, stage.name)
}

func StageFromString(stageName string) *Stage {
	for _, stage := range []Stage{GammaStage, ProdStage} {
		if stage.name == stageName {
			return &stage
		}
	}

	panic(fmt.Sprintf("Stage '%s' is not valid. Choose from: '%s'", stageName, []Stage{GammaStage, ProdStage}))
}

var GammaStage = Stage{name: gammaStageName}
var ProdStage = Stage{name: prodStageName}

var DefaultRegion = UsWest2
var UsWest2 = Region{"us-west-2"}
var UsEast1 = Region{"us-east-1"}
