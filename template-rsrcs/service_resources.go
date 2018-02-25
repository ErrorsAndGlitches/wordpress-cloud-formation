package template_rsrcs

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/crewjam/go-cloudformation"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/constants"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/cf_funcs"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/wp"
)

var numSubnets = 3
var volumeSizeGiB = int64(8)
var ssdVolumeType = "gp2"

type ServiceParameters struct {
	Config *TemplateConfig
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
		Description:           "Domain name for the system",
		Type:                  "String",
	}
	template.Parameters[CertificateArnParamName] = &Parameter{
		AllowedPattern:        "arn:aws:acm:.*certificate.*",
		ConstraintDescription: "must be a certificate ARN",
		Description:           "AWS ACM Certificate ARN",
		Type:                  "String",
	}
	template.Parameters[Ec2KeyNameParamName] = &Parameter{
		AllowedPattern:        "[a-zA-Z][a-zA-Z0-9-]*",
		ConstraintDescription: "must begin with a letter and contain only alphanumeric characters",
		Description:           "AWS EC2 key name for SSH'ing into hosts",
		Type:                  "String",
	}
}

func (s *ServiceParameters) CloudFormationParameters(
	dbPassword string, domainName string, certArn string, ec2KeyName string,
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
			ParameterKey:   &Ec2KeyNameParamName,
			ParameterValue: &ec2KeyName,
		},
	}
}

type ServiceResources struct {
	Template            *Template
	Config              *TemplateConfig
	AZs                 []*ec2.AvailabilityZone
	WordPressSubDomains []string
}

func (s *ServiceResources) AddToTemplate() {
	s.addParameters()

	s.addVPC()
	s.addSubnets()
	s.addEc2IamInstanceProfile()
	s.addEc2Role()
	s.addEc2SecurityGroup()
	s.addInternetGateway()
	s.addInternetGatewayAttachment()
	s.addRouteTable()
	s.addPublicRoute()
	s.addSubnetRouteTableAssociations()

	s.addLoadBalancer()
	s.addLoadBalancerSecurityGroup()
	s.addEfsVolume()
	s.addEfsMountTargets()

	// add wp stuff here
	wpResources := wp.NewWordPressResources(
		s.Template, s.Config, s.elbLogicalName(), s.WordPressSubDomains,
		Ref(s.vpcLogicalName()), s.ec2SecurityGroupRefStringExpr(), Ref(s.elbSecurityGroupLogicalName()).String(),
	)
	wpResources.AddToTemplate()

	s.addLaunchConfiguration(wpResources.EcsClusterLogicalName())
	s.addAsg()

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

func (s *ServiceResources) elbLogicalName() string {
	return s.Config.CfName("AppLoadBalancer")
}

func (s *ServiceResources) elbSecurityGroupLogicalName() string {
	return s.Config.CfName("LBSecurityGroup")
}

func (s *ServiceResources) ec2SecurityGroupLogicalName() string {
	return s.Config.CfName("Ec2SecurityGroup")
}

func (s *ServiceResources) ec2SecurityGroupRefStringExpr() *StringExpr {
	return Ref(s.Config.CfName("Ec2SecurityGroup")).String()
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

func (s *ServiceResources) efsLogicalName() string {
	return s.Config.CfName("Efs")
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
			Name:           String(s.Config.CfName("WordPressLoadBalancer")),
			SecurityGroups: StringList(Ref(s.elbSecurityGroupLogicalName()).String()),
			Subnets:        s.subnetRefs(),
		},
	)
}

func (s *ServiceResources) addLoadBalancerSecurityGroup() {
	s.Template.AddResource(
		s.elbSecurityGroupLogicalName(),
		&EC2SecurityGroup{
			GroupDescription: String("Security group for the Application level load balancer"),
			SecurityGroupEgress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(AllIps),
					IpProtocol: String(AllProtocols),
				},
			},
			SecurityGroupIngress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(AllIps),
					IpProtocol: String(TcpProtocol),
					FromPort:   Integer(SshPort),
					ToPort:     Integer(SshPort),
				},
				EC2SecurityGroupRule{
					CidrIp:     String(AllIps),
					IpProtocol: String(TcpProtocol),
					FromPort:   Integer(HttpsPort),
					ToPort:     Integer(HttpsPort),
				},
			},
			VpcId: Ref(s.vpcLogicalName()).String(),
		},
	)
}

func (s *ServiceResources) addAsg() {
	s.Template.AddResource(
		s.Config.CfName("AutoScalingGroup"),
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

func (s *ServiceResources) addLaunchConfiguration(ecsClusterLogicalName string) {
	s.Template.AddResource(
		s.launchConfigLogicalName(),
		&AutoScalingLaunchConfiguration{
			BlockDeviceMappings: &AutoScalingBlockDeviceMappingList{
				{
					DeviceName: String("/dev/xvda"),
					Ebs: &AutoScalingEBSBlockDevice{
						VolumeSize: Integer(volumeSizeGiB),
						VolumeType: String(ssdVolumeType),
					},
				},
			},
			IamInstanceProfile: Ref(s.ec2InstanceProfileLogicalName()).String(),
			// see ECS optimized AMIs: http://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-optimized_AMI.html
			ImageId:            String("ami-7114c909"),
			InstanceMonitoring: Bool(false),
			InstanceType:       String("t2.micro"),
			KeyName:            Ref(Ec2KeyNameParamName).String(),
			SecurityGroups:     []interface{}{s.ec2SecurityGroupRefStringExpr()},
			UserData: Base64(Sub(String(fmt.Sprintf(
				"#!/bin/bash -xe\n"+
					"echo ECS_CLUSTER=${%s} >> /etc/ecs/ecs.config\n"+
					"yum install -y aws-cfn-bootstrap nfs-utils\n"+
					"mkdir -p /mnt/efs/\n"+
					"chown ec2-user:ec2-user /mnt/efs/\n"+
					"mount -t nfs -o nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2 ${%s}.efs.${AWS::Region}.amazonaws.com:/ /mnt/efs/\n"+
					"/opt/aws/bin/cfn-signal -e $? --stack ${AWS::StackName} --region ${AWS::Region} --resource ECSAutoScalingGroup\n",
				ecsClusterLogicalName, s.efsLogicalName(),
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
					CidrIp:     String(AllIps),
					IpProtocol: String(AllProtocols),
				},
			},
			SecurityGroupIngress: &EC2SecurityGroupRuleList{
				EC2SecurityGroupRule{
					CidrIp:     String(AllIps),
					IpProtocol: String(TcpProtocol),
					FromPort:   Integer(SshPort),
					ToPort:     Integer(SshPort),
				},
			},
			VpcId: Ref(s.vpcLogicalName()).String(),
		},
	)

	// TODO: determine whether can be removed
	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressFromElbDynamicPorts"),
		&EC2SecurityGroupIngress{
			GroupId:               s.ec2SecurityGroupRefStringExpr(),
			SourceSecurityGroupId: s.ec2SecurityGroupRefStringExpr(),
			IpProtocol:            String(TcpProtocol),
			FromPort:              Integer(32768),
			ToPort:                Integer(65535),
		},
	)

	s.Template.AddResource(
		s.Config.CfName("EC2SecurityGroupIngressEFS"),
		&EC2SecurityGroupIngress{
			GroupId:               Ref(s.ec2SecurityGroupLogicalName()).String(),
			SourceSecurityGroupId: Ref(s.ec2SecurityGroupLogicalName()).String(),
			IpProtocol:            String(TcpProtocol),
			FromPort:              Integer(NfsPort),
			ToPort:                Integer(NfsPort),
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
			Tags: []ResourceTag{{Key: String("StackName"), Value: Ref("AWS::StackName").String()}},
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
			DestinationCidrBlock: String(AllIps),
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
