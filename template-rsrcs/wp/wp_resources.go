package wp

import (
	. "github.com/crewjam/go-cloudformation"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/models"
	. "github.com/ErrorsAndGlitches/wordpress-cloud-formation/template-rsrcs/constants"
)

var oneCpu int64 = 1024             // in ECS, there are 1024 units per VCPU
var memoryPerInstanceMb int64 = 992 // based on EC2 t2.micro
var baseWordPressPort int64 = 9000

type WordPressResources struct {
	template            *Template
	config              *TemplateConfig
	elbLogicalName      string
	wordPressSubDomains []string
	vpcIdRefFunc        RefFunc
	ec2SecGrpLogName    *StringExpr
	elbSecGrpLogName    *StringExpr
}

func NewWordPressResources(
	template *Template, config *TemplateConfig, elbLogicalName string, wordPressSubDomains []string,
	vpcIdRefFunc RefFunc, ec2SecGrpLogName *StringExpr, elbSecGrpLogName *StringExpr,
) WordPressResources {
	return WordPressResources{
		template, config, elbLogicalName, wordPressSubDomains, vpcIdRefFunc, ec2SecGrpLogName, elbSecGrpLogName,
	}
}

func (wprs *WordPressResources) AddToTemplate() {
	var wpSubdomainRsrcs []wpSubdomainResource
	for index, subdomain := range wprs.wordPressSubDomains {
		wpRsrc := newWordPressResource(
			wprs.template, wprs.config, wprs.elbLogicalName, wprs.elbListenerLogicalName(),
			wprs.vpcIdRefFunc, wprs.ec2SecGrpLogName, wprs.elbSecGrpLogName, wprs.ecsClusterRef(),
			Ref(wprs.logGroupLogicalName()).String(),
			subdomain, baseWordPressPort+int64(index), wprs.cpuUnitsPerTask(), wprs.memoryPerTask(),
		)
		wpRsrc.AddToTemplate()
		wpSubdomainRsrcs = append(wpSubdomainRsrcs, wpRsrc)
	}

	wprs.addEcsCluster()
	wprs.addLogGroup()

	wprs.addElbListener(wpSubdomainRsrcs[0].elbTargetGroupRef())
}

func (wprs *WordPressResources) EcsClusterLogicalName() string {
	return wprs.config.CfName("EcsCluster")
}

func (wprs *WordPressResources) ecsClusterRef() *StringExpr {
	return Ref(wprs.EcsClusterLogicalName()).String()
}

func (wprs *WordPressResources) logGroupLogicalName() string {
	return wprs.config.CfName("EcsCloudWatchLogGroup")
}

func (wprs *WordPressResources) elbListenerLogicalName() string {
	return wprs.config.CfName("ElbHttpsListener")
}

func (wprs *WordPressResources) cpuUnitsPerTask() int64 {
	return oneCpu / int64(2*len(wprs.wordPressSubDomains))
}

func (wprs *WordPressResources) memoryPerTask() int64 {
	return memoryPerInstanceMb / int64(2*len(wprs.wordPressSubDomains))
}

func (wprs *WordPressResources) addEcsCluster() {
	wprs.template.AddResource(
		wprs.EcsClusterLogicalName(),
		&ECSCluster{}, // purposefully empty, nothing should be specified
	)
}

func (wprs *WordPressResources) addLogGroup() {
	wprs.template.AddResource(
		wprs.logGroupLogicalName(),
		&LogsLogGroup{
			LogGroupName:    Join("-", Ref("AWS::StackName"), String("WordPress"), wprs.config.Stage.StringExpr()),
			RetentionInDays: Integer(7),
		},
	)
}

func (wprs *WordPressResources) addElbListener(defaultTargetGrpArn *StringExpr) {
	wprs.template.AddResource(
		wprs.elbListenerLogicalName(),
		ElasticLoadBalancingV2Listener{
			Certificates: &ElasticLoadBalancingListenerCertificatesList{
				ElasticLoadBalancingListenerCertificates{
					CertificateArn: Ref(CertificateArnParamName).String(),
				},
			},
			DefaultActions: &ElasticLoadBalancingListenerDefaultActionsList{
				ElasticLoadBalancingListenerDefaultActions{
					TargetGroupArn: defaultTargetGrpArn,
					Type:           String("forward"),
				},
			},
			LoadBalancerArn: Ref(wprs.elbLogicalName).String(),
			Port:            Integer(HttpsPort),
			Protocol:        String(HttpsProtocol),
		},
	)
}
