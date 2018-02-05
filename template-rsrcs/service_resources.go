package template_rsrcs

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/crewjam/go-cloudformation"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"strconv"
)

var allIps = "0.0.0.0/0"
var httpsPort int64 = 443
var sshPort int64 = 22
var nfsPort int64 = 2049
var scalaPlayPort int64 = 9000
var allProtocols = "-1"
var tcpProtocol = "tcp"
var httpProtocol = "HTTP"
var httpsProtocol = "HTTPS"
var oneCpu int64 = 1024             // in ECS, there are 1024 units per VCPU
var memoryPerInstanceMb int64 = 992 // based on EC2 t2.micro
var databaseContainerName = "AlertSysDbContainer"
var databaseEcsVolumeName = String("DatabaseVolume")
var numSubnets = 3

var MysqlPasswordParamName = "MysqlPassword"
var DomainNameParamName = "DomainName"
var CertificateArnParamName = "CertificateArn"
var TwilioUserParamName = "TwilioUser"
var TwilioPasswordParamName = "TwilioPassword"
var TwilioPhoneParamName = "TwilioPhone"
var PlayFwkSecretKeyParamName = "PlaySecretKey"

type ServiceParameters struct {
	Config *AlertSysConfig
}

func (s *ServiceParameters) AddToTemplate(template *Template) {
	template.Parameters[MysqlPasswordParamName] = &Parameter{
		AllowedPattern:        "[a-zA-Z][a-zA-Z0-9]*",
		ConstraintDescription: "must begin with a letter and contain only alphanumeric characters",
		Description:           "Password for the mysql database",
		MinLength:             Integer(8),
		MaxLength:             Integer(64),
		Type:                  "String",
	}
	template.Parameters[DomainNameParamName] = &Parameter{
		AllowedPattern:        "[a-zA-Z][a-zA-Z0-9-]*.[a-zA-Z]+",
		ConstraintDescription: "must be a URL",
		Description:           "Domain name for the alert system",
		Type:                  "String",
	}
	template.Parameters[CertificateArnParamName] = &Parameter{
		AllowedPattern:        "arn:aws:acm:.*certificate.*",
		ConstraintDescription: "must be a certificate ARN",
		Description:           "AWS ACM Certificate ARN",
		Type:                  "String",
	}
	template.Parameters[TwilioUserParamName] = &Parameter{
		AllowedPattern:        ".+",
		ConstraintDescription: "must be at least one character",
		Description:           "Twilio account user name",
		Type:                  "String",
	}
	template.Parameters[TwilioPasswordParamName] = &Parameter{
		AllowedPattern:        ".+",
		ConstraintDescription: "must be at least one character. it really should be more. like srsly.",
		Description:           "Twilio account password",
		Type:                  "String",
	}
	template.Parameters[TwilioPhoneParamName] = &Parameter{
		AllowedPattern:        "\\+1[0-9]{10}",
		ConstraintDescription: "must be a phone number with '+1' followed by 10 digits",
		Description:           "Twilio phone number",
		Type:                  "String",
	}
	template.Parameters[PlayFwkSecretKeyParamName] = &Parameter{
		AllowedPattern:        ".+",
		ConstraintDescription: "must be at least one character. it really should be more. like srsly.",
		Description:           "Play framework secret key",
		Type:                  "String",
	}
}

func (s *ServiceParameters) CloudFormationParameters(
	dbPassword string, domainName string, certArn string, twilioUser string, twilioPassword string, twilioPhone string,
	playFwkSecretKey string,
) []*cloudformation.Parameter {

	return []*cloudformation.Parameter{
		{
			ParameterKey:   &MysqlPasswordParamName,
			ParameterValue: &dbPassword,
		},
		{
			ParameterKey:   &DomainNameParamName,
			ParameterValue: &domainName,
		},
		{
			ParameterKey:   &CertificateArnParamName,
			ParameterValue: &certArn,
		},
		{
			ParameterKey:   &TwilioUserParamName,
			ParameterValue: &twilioUser,
		},
		{
			ParameterKey:   &TwilioPasswordParamName,
			ParameterValue: &twilioPassword,
		},
		{
			ParameterKey:   &TwilioPhoneParamName,
			ParameterValue: &twilioPhone,
		},
		{
			ParameterKey:   &PlayFwkSecretKeyParamName,
			ParameterValue: &playFwkSecretKey,
		},
	}
}

type ServiceResources struct {
	Template *Template
	Config   *AlertSysConfig
	AZs      []*ec2.AvailabilityZone
}

func (s *ServiceResources) AddToTemplate() {
	s.addParameters()

	s.addVPC()
	s.addSubnets()
	s.addLaunchConfiguration()
	s.addEc2IamInstanceProfile()
	s.addEc2Role()
	s.addEc2SecurityGroup()
	s.addInternetGateway()
	s.addInternetGatewayAttachment()
	s.addRouteTable()
	s.addPublicRoute()
	s.addSubnetRouteTableAssociations()

	s.addLoadBalancer()
	s.addElbListener()
	s.addElbListenerRule()
	s.addLoadBalancerSecurityGroup()
	s.addLoadBalancerTargetGroup()

	s.addEfsVolume()
	s.addEfsMountTargets()

	s.addEcsAsg()
	s.addEcsService()
	s.addEcsCluster()
	s.addEcsTaskDef()
	s.addEcsServiceRole()
	s.addLogGroup()

	s.addOutputs()
}

func (s *ServiceResources) addParameters() {
	(&ServiceParameters{Config: s.Config}).AddToTemplate(s.Template)
}

func (s *ServiceResources) addOutputs() {
	elbOutputName := s.Config.CfName("OutputElb")
	s.Template.Outputs[elbOutputName] = &Output{
		Description: "Elastic Load Balancer Public DNS",
		Value:       Ref(s.elbLogicalName()),
	}
}

func (s *ServiceResources) stackName() RefFunc {
	return Ref("AWS::StackName")
}

func (s *ServiceResources) region() RefFunc {
	return Ref("AWS::Region")
}

func (s *ServiceResources) elbLogicalName() string {
	return s.Config.CfName("AppLoadBalancer")
}

func (s *ServiceResources) elbSecurityGroupLogicalName() string {
	return s.Config.CfName("LBSecurityGroup")
}

func (s *ServiceResources) elbTargetGroupLogicalName() string {
	return s.Config.CfName("LBTargetGroup")
}

func (s *ServiceResources) elbListenerLogicalName() string {
	return s.Config.CfName("ElbHttpsListener")
}

func (s *ServiceResources) ec2SecurityGroupLogicalName() string {
	return s.Config.CfName("Ec2SecurityGroup")
}

func (s *ServiceResources) ec2InstanceProfileLogicalName() string {
	return s.Config.CfName("Ec2InstanceIamProfile")
}

func (s *ServiceResources) ec2IamRoleLogicalName() string {
	return s.Config.CfName("Ec2IamRole")
}

func (s *ServiceResources) vpcLogicalName() string {
	return s.Config.CfName("VPC")
}

func (s *ServiceResources) subnetLogicalName(index int) string {
	return s.Config.CfName(fmt.Sprintf("Subnet%d", index))
}

func (s *ServiceResources) internetGatewayLogicalName() string {
	return s.Config.CfName("InternetGateway")
}

func (s *ServiceResources) internetGatewayAttachmentLogicalName() string {
	return s.Config.CfName("VpcInternetGatewayAttachment")
}

func (s *ServiceResources) routeTableLogicalName() string {
	return s.Config.CfName("VpcRouteTable")
}

func (s *ServiceResources) launchConfigLogicalName() string {
	return s.Config.CfName("EcsLaunchConfig")
}

func (s *ServiceResources) ecsClusterLogicalName() string {
	return s.Config.CfName("EcsCluster")
}

func (s *ServiceResources) efsLogicalName() string {
	return s.Config.CfName("Efs")
}

func (s *ServiceResources) ecsTaskDefLogicalName() string {
	return s.Config.CfName("AlertSysTaskDef")
}

func (s *ServiceResources) ecsServiceContainerName() string {
	return s.Config.CfName("AlertServiceEcsContainer")
}

func (s *ServiceResources) ecsServiceRoleLogicalName() string {
	return s.Config.CfName("AlertSystemEcsServiceRole")
}

func (s *ServiceResources) logGroupLogicalName() string {
	return s.Config.CfName("EcsCloudWatchLogGroup")
}

func (s *ServiceResources) subnetRefs() *StringListExpr {
	var subnets []Stringable
	for i := 0; i < numSubnets; i++ {
		subnets = append(subnets, Ref(s.subnetLogicalName(i)))
	}

	return StringList(subnets...)
}

func (s *ServiceResources) addLoadBalancerRecordSet() {
	s.Template.AddResource(
		s.Config.CfName("RecordSetForDomainName"),
		&Route53RecordSet{
			AliasTarget: &Route53AliasTargetProperty{
				DNSName:      GetAtt(s.elbLogicalName(), "DNSName"),
				HostedZoneId: GetAtt(s.elbLogicalName(), "CanonicalHostedZoneID"),
			},
			Name: Ref(DomainNameParamName).String(),
		},
	)
}

func (s *ServiceResources) addLoadBalancer() {
	s.Template.AddResource(
		s.elbLogicalName(),
		&ElasticLoadBalancingV2LoadBalancer{
			LoadBalancerAttributes: &ElasticLoadBalancingLoadBalancerLoadBalancerAttributesList{
				ElasticLoadBalancingLoadBalancerLoadBalancerAttributes{
					Key:   String("idle_timeout.timeout_seconds"),
					Value: String("30"),
				},
			},
			Name:           String(s.Config.CfName("AlertSysLoadBalancer")),
			SecurityGroups: StringList(Ref(s.elbSecurityGroupLogicalName()).String()),
			Subnets:        s.subnetRefs(),
		},
	)
}

func (s *ServiceResources) addElbListener() {
	listener := Resource{
		Properties: &ElasticLoadBalancingV2Listener{
			Certificates: &ElasticLoadBalancingListenerCertificatesList{
				ElasticLoadBalancingListenerCertificates{
					CertificateArn: Ref(CertificateArnParamName).String(),
				},
			},
			DefaultActions: &ElasticLoadBalancingListenerDefaultActionsList{
				ElasticLoadBalancingListenerDefaultActions{
					TargetGroupArn: Ref(s.elbTargetGroupLogicalName()).String(),
					Type:           String("forward"),
				},
			},
			LoadBalancerArn: Ref(s.elbLogicalName()).String(),
			Port:            Integer(httpsPort),
			Protocol:        String(httpsProtocol),
		},
	}

	s.Template.Resources[s.elbListenerLogicalName()] = &listener
}

func (s *ServiceResources) addElbListenerRule() {
	listenerRule := Resource{
		Properties: &ElasticLoadBalancingV2ListenerRule{
			Actions: &ElasticLoadBalancingListenerRuleActionsList{
				ElasticLoadBalancingListenerRuleActions{
					TargetGroupArn: Ref(s.elbTargetGroupLogicalName()).String(),
					Type:           String("forward"),
				},
			},
			Conditions: &ElasticLoadBalancingListenerRuleConditionsList{
				ElasticLoadBalancingListenerRuleConditions{
					Field:  String("path-pattern"),
					Values: StringList(String("/")),
				},
			},
			ListenerArn: Ref(s.elbListenerLogicalName()).String(),
			Priority:    Integer(1),
		},
	}

	s.Template.Resources[s.Config.CfName("HttpsListenerRule")] = &listenerRule
}

func (s *ServiceResources) addLoadBalancerSecurityGroup() {
	s.Template.AddResource(
		s.elbSecurityGroupLogicalName(),
		&EC2SecurityGroup{
			GroupDescription: String("Security group for the Application level load balancer"),
			SecurityGroupEgress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(allIps),
					IpProtocol: String(allProtocols),
				},
			},
			SecurityGroupIngress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(allIps),
					IpProtocol: String(tcpProtocol),
					FromPort:   Integer(sshPort),
					ToPort:     Integer(sshPort),
				},
				EC2SecurityGroupRule{
					CidrIp:     String(allIps),
					IpProtocol: String(tcpProtocol),
					FromPort:   Integer(httpsPort),
					ToPort:     Integer(httpsPort),
				},
			},
			VpcId: Ref(s.vpcLogicalName()).String(),
		},
	)
}

func (s *ServiceResources) addLoadBalancerTargetGroup() {
	targetGroup := Resource{
		DependsOn: []string{s.elbLogicalName()},
		Properties: &ElasticLoadBalancingV2TargetGroup{
			HealthCheckIntervalSeconds: Integer(10),
			HealthCheckPath:            String("/"),
			HealthCheckPort:            String(strconv.FormatInt(scalaPlayPort, 10)),
			HealthCheckProtocol:        String(httpProtocol),
			HealthCheckTimeoutSeconds:  Integer(5),
			HealthyThresholdCount:      Integer(2),
			Port:                       Integer(scalaPlayPort),
			Protocol:                   String(httpProtocol),
			UnhealthyThresholdCount:    Integer(2),
			VpcId:                      Ref(s.vpcLogicalName()).String(),
		},
	}

	s.Template.Resources[s.elbTargetGroupLogicalName()] = &targetGroup
}

func (s *ServiceResources) addEcsAsg() {
	s.Template.AddResource(
		s.Config.CfName("EcsAutoScalingGroup"),
		&AutoScalingAutoScalingGroup{
			AvailabilityZones:       GetAZs(s.Config.Region.StringExpr()),
			DesiredCapacity:         String("1"),
			LaunchConfigurationName: Ref(s.launchConfigLogicalName()).String(),
			MinSize:                 String("1"),
			MaxSize:                 String("1"),
			VPCZoneIdentifier:       s.subnetRefs(),
		},
	)
}

func (s *ServiceResources) addLaunchConfiguration() {
	s.Template.AddResource(
		s.launchConfigLogicalName(),
		&AutoScalingLaunchConfiguration{
			IamInstanceProfile: Ref(s.ec2InstanceProfileLogicalName()).String(),
			// see ECS optimized AMIs: http://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-optimized_AMI.html
			ImageId:            String("ami-7114c909"),
			InstanceMonitoring: Bool(false),
			InstanceType:       String("t2.micro"),
			KeyName:            String("alert-sys-cf-key"),
			SecurityGroups:     []interface{}{Ref(s.ec2SecurityGroupLogicalName()).String()},
			UserData: Base64(Sub(String(fmt.Sprintf(
				"#!/bin/bash -xe\n"+
					"echo ECS_CLUSTER=${%s} >> /etc/ecs/ecs.config\n"+
					"yum install -y aws-cfn-bootstrap nfs-utils\n"+
					"mkdir -p /mnt/efs/\n"+
					"chown ec2-user:ec2-user /mnt/efs/\n"+
					"mount -t nfs -o nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2 ${%s}.efs.${AWS::Region}.amazonaws.com:/ /mnt/efs/\n"+
					"/opt/aws/bin/cfn-signal -e $? --stack ${AWS::StackName} --region ${AWS::Region} --resource ECSAutoScalingGroup\n",
				s.ecsClusterLogicalName(), s.efsLogicalName(),
			)))),
		},
	)
}

func (s *ServiceResources) addEc2IamInstanceProfile() {
	s.Template.AddResource(
		s.ec2InstanceProfileLogicalName(),
		&IAMInstanceProfile{
			Roles: StringList(Ref(s.ec2IamRoleLogicalName()).String()),
		},
	)
}

func (s *ServiceResources) addEc2Role() {
	s.Template.AddResource(
		s.ec2IamRoleLogicalName(),
		&IAMRole{
			AssumeRolePolicyDocument: `{
                "Statement":[
                {
                  "Effect":"Allow",
                  "Principal":{
                    "Service":[
                      "ec2.amazonaws.com"
                    ]
                  },
                  "Action":[
                    "sts:AssumeRole"
                  ]
                }
              ]
            }`,
			Path: String("/"),
			Policies: &IAMPoliciesList{
				IAMPolicies{
					PolicyName: String("ec2-ecs-service-access"),
					PolicyDocument: `{
                        "Statement":[
                            {
                              "Effect":"Allow",
                              "Action":[
                                "ecs:CreateCluster",
                                "ecs:DeregisterContainerInstance",
                                "ecs:DiscoverPollEndpoint",
                                "ecs:Poll",
                                "ecs:RegisterContainerInstance",
                                "ecs:StartTelemetrySession",
                                "ecs:Submit*",
                                "logs:CreateLogStream",
                                "logs:PutLogEvents"
                              ],
                              "Resource":"*"
                            }
                        ]
                    }`,
				},
			},
		},
	)
}

func (s *ServiceResources) addEc2SecurityGroup() {
	s.Template.AddResource(
		s.ec2SecurityGroupLogicalName(),
		&EC2SecurityGroup{
			GroupDescription: String("Security group for the EC2 instances running in the ECS cluster"),
			SecurityGroupEgress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(allIps),
					IpProtocol: String(allProtocols),
				},
			},
			SecurityGroupIngress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(allIps),
					IpProtocol: String(tcpProtocol),
					FromPort:   Integer(sshPort),
					ToPort:     Integer(sshPort),
				},
			},
			VpcId: Ref(s.vpcLogicalName()).String(),
		},
	)

	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressFromElb"),
		&EC2SecurityGroupIngress{
			GroupId:               Ref(s.ec2SecurityGroupLogicalName()).String(),
			SourceSecurityGroupId: Ref(s.elbSecurityGroupLogicalName()).String(),
			IpProtocol:            String(tcpProtocol),
			FromPort:              Integer(scalaPlayPort),
			ToPort:                Integer(scalaPlayPort),
		},
	)

	// TODO: determine whether can be removed
	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressFromElbDynamicPorts"),
		&EC2SecurityGroupIngress{
			GroupId:               Ref(s.ec2SecurityGroupLogicalName()).String(),
			SourceSecurityGroupId: Ref(s.elbSecurityGroupLogicalName()).String(),
			IpProtocol:            String(tcpProtocol),
			FromPort:              Integer(32768),
			ToPort:                Integer(65535),
		},
	)

	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressForScalaPlay"),
		&EC2SecurityGroupIngress{
			GroupId:               Ref(s.ec2SecurityGroupLogicalName()).String(),
			SourceSecurityGroupId: Ref(s.ec2SecurityGroupLogicalName()).String(),
			IpProtocol:            String(tcpProtocol),
			FromPort:              Integer(scalaPlayPort),
			ToPort:                Integer(scalaPlayPort),
		},
	)

	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressEFS"),
		&EC2SecurityGroupIngress{
			GroupId:               Ref(s.ec2SecurityGroupLogicalName()).String(),
			SourceSecurityGroupId: Ref(s.ec2SecurityGroupLogicalName()).String(),
			IpProtocol:            String(tcpProtocol),
			FromPort:              Integer(nfsPort),
			ToPort:                Integer(nfsPort),
		},
	)
}

func (s *ServiceResources) addVPC() {
	s.Template.AddResource(
		s.vpcLogicalName(),
		&EC2VPC{
			CidrBlock:          String("10.0.0.0/16"),
			EnableDnsHostnames: Bool(true),
			InstanceTenancy:    String("default"),
		},
	)
}

func (s *ServiceResources) addSubnets() {
	for i := 0; i < numSubnets; i++ {
		s.Template.AddResource(
			s.subnetLogicalName(i),
			&EC2Subnet{
				AvailabilityZone:    String(*s.AZs[i%len(s.AZs)].ZoneName),
				CidrBlock:           String(fmt.Sprintf("10.0.%d.0/24", i)),
				MapPublicIpOnLaunch: Bool(true),
				VpcId:               Ref(s.vpcLogicalName()).String(),
			},
		)
	}
}

func (s *ServiceResources) addInternetGateway() {
	s.Template.AddResource(
		s.internetGatewayLogicalName(),
		&EC2InternetGateway{
			Tags: []ResourceTag{{Key: String("StackName"), Value: s.stackName().String()}},
		},
	)
}

func (s *ServiceResources) addInternetGatewayAttachment() {
	s.Template.AddResource(
		s.internetGatewayAttachmentLogicalName(),
		&EC2VPCGatewayAttachment{
			InternetGatewayId: Ref(s.internetGatewayLogicalName()).String(),
			VpcId:             Ref(s.vpcLogicalName()).String(),
		},
	)
}

func (s *ServiceResources) addRouteTable() {
	s.Template.AddResource(
		s.routeTableLogicalName(),
		&EC2RouteTable{
			VpcId: Ref(s.vpcLogicalName()).String(),
		},
	)
}

func (s *ServiceResources) addPublicRoute() {
	route := Resource{
		DependsOn: []string{s.internetGatewayAttachmentLogicalName()},
		Properties: &EC2Route{
			RouteTableId:         Ref(s.routeTableLogicalName()).String(),
			DestinationCidrBlock: String(allIps),
			GatewayId:            Ref(s.internetGatewayLogicalName()).String(),
		},
	}
	s.Template.Resources[s.Config.CfName("PublicRoute")] = &route
}

func (s *ServiceResources) addSubnetRouteTableAssociations() {
	for i, subnetRef := range s.subnetRefs().Literal {
		s.Template.AddResource(
			s.Config.CfName(fmt.Sprintf("Subnet%dRouteTableAssoc", i)),
			&EC2SubnetRouteTableAssociation{
				RouteTableId: Ref(s.routeTableLogicalName()).String(),
				SubnetId:     subnetRef,
			},
		)
	}
}

func (s *ServiceResources) addEcsService() {
	serviceResource := Resource{
		DependsOn: []string{s.elbListenerLogicalName()},
		Properties: &ECSService{
			Cluster:      Ref(s.ecsClusterLogicalName()).String(),
			DesiredCount: Integer(1),
			LoadBalancers: &EC2ContainerServiceServiceLoadBalancersList{
				EC2ContainerServiceServiceLoadBalancers{
					ContainerName:  String(s.ecsServiceContainerName()),
					ContainerPort:  Integer(scalaPlayPort),
					TargetGroupArn: Ref(s.elbTargetGroupLogicalName()).String(),
				},
			},
			Role:           Ref(s.ecsServiceRoleLogicalName()).String(),
			TaskDefinition: Ref(s.ecsTaskDefLogicalName()).String(),
		},
	}

	s.Template.Resources[s.Config.CfName("EcsService")] = &serviceResource
}

func (s *ServiceResources) addEcsCluster() {
	s.Template.AddResource(
		s.ecsClusterLogicalName(),
		&ECSCluster{}, // purposefully empty, nothing should be specified
	)
}

func (s *ServiceResources) addEfsVolume() {
	s.Template.AddResource(
		s.efsLogicalName(),
		&EFSFileSystem{
			PerformanceMode: String("generalPurpose"),
		},
	)
}

func (s *ServiceResources) addEfsMountTargets() {
	fileSystemId := Ref(s.efsLogicalName()).String()
	for i, subnetRef := range s.subnetRefs().Literal {
		s.Template.AddResource(
			s.Config.CfName(fmt.Sprintf("%s%d", "EC2MountTarget", i)),
			&EFSMountTarget{
				FileSystemId:   fileSystemId,
				SecurityGroups: StringList(Ref(s.ec2SecurityGroupLogicalName())),
				SubnetId:       subnetRef,
			},
		)
	}
}

func (s *ServiceResources) addEcsTaskDef() {
	s.Template.AddResource(
		s.ecsTaskDefLogicalName(),
		&ECSTaskDefinition{
			ContainerDefinitions: &EC2ContainerServiceTaskDefinitionContainerDefinitionsList{
				*s.alertServiceContainerDef(),
				*s.databaseContainerDef(),
			},
			Volumes: &EC2ContainerServiceTaskDefinitionVolumesList{
				EC2ContainerServiceTaskDefinitionVolumes{
					Name: databaseEcsVolumeName,
					Host: &EC2ContainerServiceTaskDefinitionVolumesHost{
						SourcePath: String("/mnt/efs/mysql/"),
					},
				},
			},
		},
	)
}

func (s *ServiceResources) alertServiceContainerDef() *EC2ContainerServiceTaskDefinitionContainerDefinitions {
	return &EC2ContainerServiceTaskDefinitionContainerDefinitions{
		Cpu: Integer(oneCpu / 4 * 3),
		Environment: &EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironmentList{
			{
				Name:  String("TWILIO_USERNAME"),
				Value: Ref(TwilioUserParamName).String(),
			},
			{
				Name:  String("TWILIO_PASSWORD"),
				Value: Ref(TwilioPasswordParamName).String(),
			},
			{
				Name:  String("TWILIO_PHONE"),
				Value: Ref(TwilioPhoneParamName).String(),
			},
			{
				Name:  String("PLAY_SECRET_KEY"),
				Value: Ref(PlayFwkSecretKeyParamName).String(),
			},
		},
		Essential:        Bool(true),
		Name:             String(s.ecsServiceContainerName()),
		Image:            String("errorsandglitches/dockerscalasmsalertsystem"),
		Links:            StringList(String(fmt.Sprintf("%s:mysql", databaseContainerName))),
		LogConfiguration: s.ecsLogConfig(),
		Memory:           Integer(memoryPerInstanceMb / 4 * 3),
		PortMappings: &EC2ContainerServiceTaskDefinitionContainerDefinitionsPortMappingsList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsPortMappings{
				ContainerPort: Integer(scalaPlayPort),
				HostPort:      Integer(scalaPlayPort),
				Protocol:      String(tcpProtocol),
			},
		},
	}
}

func (s *ServiceResources) databaseContainerDef() *EC2ContainerServiceTaskDefinitionContainerDefinitions {
	return &EC2ContainerServiceTaskDefinitionContainerDefinitions{
		Cpu:       Integer(oneCpu / 4 * 1),
		Essential: Bool(true),
		Environment: &EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironmentList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsEnvironment{
				Name:  String("MYSQL_ROOT_PASSWORD"),
				Value: Ref(MysqlPasswordParamName).String(),
			},
		},
		Name:             String(databaseContainerName),
		Image:            String("mariadb:10.3.2"),
		LogConfiguration: s.ecsLogConfig(),
		Memory:           Integer(memoryPerInstanceMb / 4 * 1),
		MountPoints: &EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPointsList{
			EC2ContainerServiceTaskDefinitionContainerDefinitionsMountPoints{
				ContainerPath: String("/var/lib/mysql"),
				SourceVolume:  databaseEcsVolumeName,
			},
		},
	}
}

func (s *ServiceResources) ecsLogConfig() *EC2ContainerServiceTaskDefinitionContainerDefinitionsLogConfiguration {
	return &EC2ContainerServiceTaskDefinitionContainerDefinitionsLogConfiguration{
		LogDriver: String("awslogs"),
		Options: map[string]interface{}{
			"awslogs-group":         Ref(s.logGroupLogicalName()),
			"awslogs-region":        s.region(),
			"awslogs-stream-prefix": "alert-system",
		},
	}
}

func (s *ServiceResources) addLogGroup() {
	s.Template.AddResource(
		s.logGroupLogicalName(),
		&LogsLogGroup{
			LogGroupName:    Join("-", s.stackName(), String("AlertSystem"), s.Config.Stage.StringExpr()),
			RetentionInDays: Integer(14),
		},
	)
}

func (s *ServiceResources) addEcsServiceRole() {
	s.Template.AddResource(
		s.ecsServiceRoleLogicalName(),
		&IAMRole{
			AssumeRolePolicyDocument: `{
                "Statement":[
                {
                  "Effect":"Allow",
                  "Principal":{
                    "Service":[
                      "ecs.amazonaws.com"
                    ]
                  },
                  "Action":[
                    "sts:AssumeRole"
                  ]
                }
              ]
            }`,
			Path: String("/"),
			Policies: &IAMPoliciesList{
				IAMPolicies{
					PolicyName: String("ecs-service-policy"),
					PolicyDocument: `{
                        "Statement":[
                            {
                              "Effect":"Allow",
                              "Action":[
                                "elasticloadbalancing:DeregisterInstancesFromLoadBalancer",
                                "elasticloadbalancing:DeregisterTargets",
                                "elasticloadbalancing:Describe*",
                                "elasticloadbalancing:RegisterInstancesWithLoadBalancer",
                                "elasticloadbalancing:RegisterTargets",
                                "ec2:Describe*",
                                "ec2:AuthorizeSecurityGroupIngress"
                               ],
                               "Resource":"*"
                            }
                        ]
                    }`,
				},
			},
		},
	)
}
