package constants

import (
	. "github.com/crewjam/go-cloudformation"
)

var EcsServiceRoleResourceProps ResourceProperties = &IAMRole{
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
}
