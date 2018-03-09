package wp

import (
	"fmt"
	"strconv"
	. "github.com/crewjam/go-cloudformation"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/constants"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/cf_funcs"
)

/**
 * ECS services can only specify a single load balancer or target group. Because of this, multiple ECS services are
 * required.
 * https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-load-balancing.html
 */
type wpSubdomainResource struct {
	template               *Template
	config                 *TemplateConfig
	elbLogicalName         string
	elbListenerLogicalName string
	vpcIdRefFunc           RefFunc
	ec2SecGrpRef           *StringExpr
	elbSecGrpRef           *StringExpr
	ecsClusterRef          *StringExpr
	logGroupRef            *StringExpr
	subdomain              string
	port                   int64
	cpuUnits               int64
	memoryMb               int64
}

func newWordPressResource(
	template *Template, config *TemplateConfig, elbLogicalName string, elbLstnrLogName string, vpcIdRefFunc RefFunc,
	ec2SecGrpLogName *StringExpr, elbSecGrpLogName *StringExpr, ecsClusterRef *StringExpr, logGroupRef *StringExpr,
	subdomain string, port int64, cpuUnits int64, memoryMb int64,
) wpSubdomainResource {
	return wpSubdomainResource{
		template, config, elbLogicalName, elbLstnrLogName,
		vpcIdRefFunc, ec2SecGrpLogName, elbSecGrpLogName, ecsClusterRef, logGroupRef,
		subdomain, port, cpuUnits, memoryMb,
	}
}

func (wpr *wpSubdomainResource) AddToTemplate() {
	wpr.addLoadBalancerTargetGroup()
	wpr.addElbListenerRules()
	wpr.addEc2SecurityGroupIngresses()
	wpr.addWpEcsService()
	wpr.addWpEcsServiceRole()
	wpr.addWpTaskDef()
}

func (wpr *wpSubdomainResource) elbListenerRuleLogicalName() string {
	return wpr.config.CfName(fmt.Sprintf("HttpsListenerRule%s", wpr.subdomain))
}

func (wpr *wpSubdomainResource) elbTargetGroupLogicalName() string {
	return wpr.subdomainLogicalName("LBTargetGroup")
}

func (wpr *wpSubdomainResource) elbTargetGroupRef() *StringExpr {
	return Ref(wpr.subdomainLogicalName("LBTargetGroup")).String()
}

func (wpr *wpSubdomainResource) wpServiceContainerName() string {
	return wpr.subdomainLogicalName("WpServiceContainer")
}

func (wpr *wpSubdomainResource) wpTaskDefLogicalName() string {
	return wpr.subdomainLogicalName("WpTaskDef")
}

func (wpr *wpSubdomainResource) wpServiceRoleLogicalName() string {
	return wpr.subdomainLogicalName("WpServiceRole")
}

func (wpr *wpSubdomainResource) subdomainLogicalName(basename string) string {
	return wpr.config.CfName(fmt.Sprintf("%s%s", basename, wpr.subdomain))
}

func (wpr *wpSubdomainResource) dbDockerVolumeName() string {
	return wpr.config.CfName(fmt.Sprintf("MySqlVolume%s", wpr.subdomain))
}

func (wpr *wpSubdomainResource) wpContentDockerVolumeName() string {
	return wpr.config.CfName(fmt.Sprintf("WpContentVolume%s", wpr.subdomain))
}

func (wpr *wpSubdomainResource) dbContainerName() string {
	return wpr.config.CfName(fmt.Sprintf("MariaDbContainer%s", wpr.subdomain))
}

func (wpr *wpSubdomainResource) addLoadBalancerTargetGroup() {
	targetGroup := Resource{
		DependsOn: []string{wpr.elbLogicalName},
		Properties: &ElasticLoadBalancingV2TargetGroup{
			HealthCheckIntervalSeconds: Integer(10),
			HealthCheckPath:            String("/"),
			HealthCheckPort:            String(strconv.FormatInt(wpr.port, 10)),
			HealthCheckProtocol:        String(HttpProtocol),
			HealthCheckTimeoutSeconds:  Integer(5),
			HealthyThresholdCount:      Integer(2),
			Matcher: &ElasticLoadBalancingTargetGroupMatcher{
				HttpCode: String("200,301,302"),
			},
			Port:                       Integer(wpr.port),
			Protocol:                   String(HttpProtocol),
			UnhealthyThresholdCount:    Integer(2),
			VpcId:                      wpr.vpcIdRefFunc.String(),
		},
	}

	wpr.template.Resources[wpr.elbTargetGroupLogicalName()] = &targetGroup
}

func (wpr *wpSubdomainResource) addElbListenerRules() {
	wpr.template.AddResource(
		wpr.elbListenerRuleLogicalName(),
		ElasticLoadBalancingV2ListenerRule{
			Actions: &ElasticLoadBalancingListenerRuleActionsList{
				ElasticLoadBalancingListenerRuleActions{
					TargetGroupArn: wpr.elbTargetGroupRef(),
					Type:           String("forward"),
				},
			},
			Conditions: &ElasticLoadBalancingListenerRuleConditionsList{
				ElasticLoadBalancingListenerRuleConditions{
					Field: String("host-header"),
					Values: StringList(
						Sub(String(fmt.Sprintf("%s.${%s}", wpr.subdomain, DomainNameParamName))),
					),
				},
			},
			ListenerArn: Ref(wpr.elbListenerLogicalName).String(),
			Priority:    Integer(int64(wpr.port)),
		},
	)
}

func (wpr *wpSubdomainResource) addEc2SecurityGroupIngresses() {
	type BaseIdTuple struct {
		basename       string
		sourceSecGrpId *StringExpr
	}

	for _, basenameIdTuple := range []BaseIdTuple{
		{"EC2SecurityGroupIngressFromElb", wpr.elbSecGrpRef},
		{"EC2SecurityGroupIngressForWordPress", wpr.ec2SecGrpRef},
	} {
		wpr.template.AddResource(
			wpr.subdomainLogicalName(basenameIdTuple.basename),
			&EC2SecurityGroupIngress{
				GroupId:               wpr.ec2SecGrpRef,
				SourceSecurityGroupId: basenameIdTuple.sourceSecGrpId,
				IpProtocol:            String(TcpProtocol),
				FromPort:              Integer(wpr.port),
				ToPort:                Integer(wpr.port),
			},
		)
	}
}

func (wpr *wpSubdomainResource) addWpEcsService() {
	wpr.template.Resources[wpr.subdomainLogicalName("WpEcsService")] = &Resource{
		DependsOn: []string{
			wpr.elbListenerLogicalName, wpr.elbLogicalName, wpr.elbTargetGroupLogicalName(), wpr.elbListenerRuleLogicalName(),
		},
		Properties: &ECSService{
			Cluster:      wpr.ecsClusterRef,
			DesiredCount: Integer(1),
			LoadBalancers: &EC2ContainerServiceServiceLoadBalancersList{
				EC2ContainerServiceServiceLoadBalancers{
					ContainerName:  String(wpr.wpServiceContainerName()),
					ContainerPort:  Integer(HttpPort),
					TargetGroupArn: wpr.elbTargetGroupRef(),
				},
			},
			Role:           Ref(wpr.wpServiceRoleLogicalName()).String(),
			TaskDefinition: Ref(wpr.wpTaskDefLogicalName()).String(),
		},
	}
}

func (wpr *wpSubdomainResource) addWpEcsServiceRole() {
	wpr.template.AddResource(
		wpr.wpServiceRoleLogicalName(),
		EcsServiceRoleResourceProps,
	)
}

func (wpr *wpSubdomainResource) addWpTaskDef() {
	wpr.template.AddResource(
		wpr.wpTaskDefLogicalName(),
		&ECSTaskDefinition{
			ContainerDefinitions: &EC2ContainerServiceTaskDefinitionContainerDefinitionsList{
				wpr.wordpressContainerDef(),
				wpr.databaseContainerDef(),
			},
			Volumes: &EC2ContainerServiceTaskDefinitionVolumesList{
				EC2ContainerServiceTaskDefinitionVolumes{
					Name: String(wpr.dbDockerVolumeName()),
					Host: &EC2ContainerServiceTaskDefinitionVolumesHost{
						SourcePath: String(fmt.Sprintf("/mnt/efs/%s/mysql/", wpr.subdomain)),
					},
				},
				EC2ContainerServiceTaskDefinitionVolumes{
					Name: String(wpr.wpContentDockerVolumeName()),
					Host: &EC2ContainerServiceTaskDefinitionVolumesHost{
						SourcePath: String(fmt.Sprintf("/mnt/efs/%s/wp-content/", wpr.subdomain)),
					},
				},
			},
		},
	)
}

func (wpr *wpSubdomainResource) wordpressContainerDef() EC2ContainerServiceTaskDefinitionContainerDefinitions {
	return EC2ContainerServiceTaskDefinitionContainerDefinitions{
		Cpu: Integer(wpr.cpuUnits),
		Environment: &EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironmentList{
			{
				Name:  String("WORDPRESS_TABLE_PREFIX"),
				Value: String(wpr.subdomain),
			},
		},
		Essential:        Bool(true),
		Image:            String("wordpress"),
		Links:            StringList(String(fmt.Sprintf("%s:mysql", wpr.dbContainerName()))),
		LogConfiguration: wpr.ecsLogConfig(),
		Memory:           Integer(wpr.memoryMb),
		MountPoints: &EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPointsList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPoints{
				ContainerPath: String("/var/www/html/wp-content"),
				SourceVolume:  String(wpr.wpContentDockerVolumeName()),
			},
		},
		Name:             String(wpr.wpServiceContainerName()),
		PortMappings: &EC2ContainerServiceTaskDefinitionContainerDefinitionsPortMappingsList{
			{
				ContainerPort: Integer(HttpPort),
				HostPort:      Integer(wpr.port),
				Protocol:      String(TcpProtocol),
			},
		},
	}
}

func (wpr *wpSubdomainResource) databaseContainerDef() EC2ContainerServiceTaskDefinitionContainerDefinitions {
	return EC2ContainerServiceTaskDefinitionContainerDefinitions{
		Cpu:       Integer(wpr.cpuUnits),
		Essential: Bool(true),
		Environment: &EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironmentList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironment{
				Name:  String("MYSQL_ROOT_PASSWORD"),
				Value: Ref(MysqlPasswordParamName).String(),
			},
		},
		Name:             String(wpr.dbContainerName()),
		Image:            String("mariadb:10.3.2"),
		LogConfiguration: wpr.ecsLogConfig(),
		Memory:           Integer(wpr.memoryMb),
		MountPoints: &EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPointsList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPoints{
				ContainerPath: String("/var/lib/mysql"),
				SourceVolume:  String(wpr.dbDockerVolumeName()),
			},
		},
	}
}

func (wpr *wpSubdomainResource) ecsLogConfig() *EC2ContainerServiceTaskDefinitionContainerDefinitionsLogConfiguration {
	return &EC2ContainerServiceTaskDefinitionContainerDefinitionsLogConfiguration{
		LogDriver: String("awslogs"),
		Options: map[string]interface{}{
			"awslogs-group":         wpr.logGroupRef,
			"awslogs-region":        Ref("AWS::Region"),
			"awslogs-stream-prefix": "wordpress",
		},
	}
}
