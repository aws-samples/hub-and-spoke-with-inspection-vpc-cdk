// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

// Permission is hereby granted, free of charge, to any person obtaining a copy of this
// software and associated documentation files (the "Software"), to deal in the Software
// without restriction, including without limitation the rights to use, copy, modify,
// merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
// INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cdkPipelines

import (
	"fmt"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	logs "github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	nf "github.com/aws/aws-cdk-go/awscdk/v2/awsnetworkfirewall"
	cr "github.com/aws/aws-cdk-go/awscdk/v2/customresources"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type NetworkFirewallStackProps struct {
	awscdk.StackProps
}

func NetworkFirewallStack(scope constructs.Construct, id string, cidr string, orgCidr string, transitGWId *string, props *NetworkFirewallStackProps) {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := ec2.NewVpc(stack, jsii.String("InspectionVPC"), &ec2.VpcProps{
		IpAddresses: ec2.IpAddresses_Cidr(&cidr),
		SubnetConfiguration: &[]*ec2.SubnetConfiguration{
			{
				Name:       jsii.String("Tgw_Subnet"),
				SubnetType: ec2.SubnetType_PRIVATE_ISOLATED,
				CidrMask:   jsii.Number(26),
			},
			{
				Name:       jsii.String("Firewall_Subnet"),
				SubnetType: ec2.SubnetType_PRIVATE_WITH_EGRESS,
				CidrMask:   jsii.Number(27),
			},
			{
				Name:       jsii.String("Public"),
				SubnetType: ec2.SubnetType_PUBLIC,
				CidrMask:   jsii.Number(27),
			},
		},
	})

	tGWSubnetIDs := vpc.SelectSubnets(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Tgw_Subnet"),
	}).SubnetIds

	tGWAttachment := ec2.NewCfnTransitGatewayAttachment(stack, jsii.String("TGW_Attachment"), &ec2.CfnTransitGatewayAttachmentProps{
		TransitGatewayId: transitGWId,
		SubnetIds:        tGWSubnetIDs,
		VpcId:            vpc.VpcId(),
		Options:          ec2.CfnTransitGatewayAttachment_OptionsProperty{ApplianceModeSupport: jsii.String("enable")},
		Tags: &[]*awscdk.CfnTag{
			{
				Key:   jsii.String("routeTable"),
				Value: jsii.String("inspection"),
			},
		},
	})

	ec2.NewCfnTransitGatewayRoute(stack, jsii.String("TGW_Route"), &ec2.CfnTransitGatewayRouteProps{
		DestinationCidrBlock:       jsii.String("0.0.0.0/0"),
		TransitGatewayAttachmentId: tGWAttachment.AttrId(),
		TransitGatewayRouteTableId: awscdk.Fn_ImportValue(jsii.String("WorkloadRouteTableId")),
	})

	firewallRules := NetworkFirewallRules(stack, "NetworkFirewallRules", nil)

	fwSubnets := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Firewall_Subnet"),
	})

	var fwSubnetList []*nf.CfnFirewall_SubnetMappingProperty
	for _, privateSub := range *fwSubnets {
		subnetMProps := &nf.CfnFirewall_SubnetMappingProperty{
			SubnetId: privateSub.SubnetId(),
		}
		fwSubnetList = append(fwSubnetList, subnetMProps)
	}

	networkFw := nf.NewCfnFirewall(stack, jsii.String("Network_Firewall"), &nf.CfnFirewallProps{
		FirewallName:      jsii.String("EgressInspectionFirewall"),
		FirewallPolicyArn: firewallRules.FwPolicyArn(),
		SubnetMappings:    fwSubnetList,
		VpcId:             vpc.VpcId(),
	})

	fwFlowLogsGroup := logs.NewLogGroup(stack, jsii.String("FWFlowLogsGroup"), &logs.LogGroupProps{
		LogGroupName:  jsii.String("NetworkFirewallFlowLogs"),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	fwAlertLogsGroup := logs.NewLogGroup(stack, jsii.String("FWAlertLogsGroup"), &logs.LogGroupProps{
		LogGroupName:  jsii.String("NetworkFirewallAlertLogs"),
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	nf.NewCfnLoggingConfiguration(stack, jsii.String("FirewallLoggingConfig"), &nf.CfnLoggingConfigurationProps{
		FirewallArn: networkFw.Ref(),
		LoggingConfiguration: nf.CfnLoggingConfiguration_LoggingConfigurationProperty{
			LogDestinationConfigs: []interface{}{
				&nf.CfnLoggingConfiguration_LogDestinationConfigProperty{
					LogDestination: map[string]*string{
						"logGroup": fwFlowLogsGroup.LogGroupName(),
					},
					LogDestinationType: jsii.String("CloudWatchLogs"),
					LogType:            jsii.String("FLOW"),
				},
				&nf.CfnLoggingConfiguration_LogDestinationConfigProperty{
					LogDestination: map[string]*string{
						"logGroup": fwAlertLogsGroup.LogGroupName(),
					},
					LogDestinationType: jsii.String("CloudWatchLogs"),
					LogType:            jsii.String("ALERT"),
				},
			},
		},
	})

	RouteLambdaRole := iam.NewRole(stack, jsii.String("routeLambdaRole"), &iam.RoleProps{
		AssumedBy: iam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		Path:      jsii.String("/"),
		ManagedPolicies: &[]iam.IManagedPolicy{
			iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("service-role/AWSLambdaBasicExecutionRole")),
		},
	})

	RouteLambdaRole.AddToPolicy(
		iam.NewPolicyStatement(&iam.PolicyStatementProps{
			Actions: jsii.Strings("network-firewall:DescribeFirewall"),
			Effect:  iam.Effect_ALLOW,
			Resources: &[]*string{
				networkFw.AttrFirewallArn(),
			},
		}),
	)

	tGWSubnets := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Tgw_Subnet"),
	})

	var tgwSubsIds []*string

	for _, subnet := range *tGWSubnets {
		subId := subnet.SubnetId()
		tgwSubsIds = append(tgwSubsIds, subId)
	}

	var cloudWanSubnetArns []*string

	for _, tgwSub := range *tGWSubnets {
		rtArns := awscdk.Arn_Format(&awscdk.ArnComponents{
			Service:  jsii.String("ec2"),
			Resource: jsii.String(fmt.Sprintf("route-table/%s", *tgwSub.RouteTable().RouteTableId())),
		},
			stack,
		)
		cloudWanSubnetArns = append(cloudWanSubnetArns, rtArns)
	}

	pubSubnets := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Public"),
	})

	var pubSubsIds []*string

	for _, subnet := range *pubSubnets {
		subId := subnet.SubnetId()
		pubSubsIds = append(pubSubsIds, subId)
	}

	var pubSubnetArns []*string
	for _, pubSub := range *pubSubnets {
		rtArns := awscdk.Arn_Format(&awscdk.ArnComponents{
			Service:  jsii.String("ec2"),
			Resource: jsii.String(fmt.Sprintf("route-table/%s", *pubSub.RouteTable().RouteTableId())),
		},
			stack,
		)
		pubSubnetArns = append(pubSubnetArns, rtArns)
	}

	listArns := append(cloudWanSubnetArns, pubSubnetArns...)

	RouteLambdaRole.AddToPolicy(
		iam.NewPolicyStatement(&iam.PolicyStatementProps{
			Actions:   jsii.Strings("ec2:CreateRoute", "ec2:DeleteRoute"),
			Effect:    iam.Effect_ALLOW,
			Resources: &listArns,
		}),
	)

	customRouteLambda := lambda.NewFunction(stack, jsii.String("RoutesFunction"), &lambda.FunctionProps{
		Runtime: lambda.Runtime_PYTHON_3_9(),
		Handler: jsii.String("index.lambda_handler"),
		Role:    RouteLambdaRole,
		Timeout: awscdk.Duration_Seconds(jsii.Number(60)),
		Code:    lambda.Code_FromAsset(jsii.String("lambda/routes"), nil),
	})

	customResource := cr.NewProvider(stack, jsii.String("provider"), &cr.ProviderProps{
		OnEventHandler: customRouteLambda,
		LogRetention:   logs.RetentionDays_ONE_DAY,
	})

	// TODO: not copied over the dependency on the Lambda as it should be
	// implicit; verify.

	// Create a default route towards AWS Network firewall endpoints. Select
	// all subnets in group CloudWANAttacment and use custom lambda function
	// to find AWS Network Firewall endpoint that is in same availability
	// zone as subnet.

	subs := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Tgw_Subnet"),
	})

	// Create a custom resource for each TGw subnet
	for _, subnet := range *subs {
		// Create CloudFormation custom resource to update firewall routing
		awscdk.NewCustomResource(stack, jsii.String(fmt.Sprintf("FirewallRoute-%s", strings.SplitN(*subnet.Node().Path(), "/", 1))), &awscdk.CustomResourceProps{
			ServiceToken: customResource.ServiceToken(),
			Properties: &map[string]interface{}{
				"FirewallArn":     networkFw.AttrFirewallArn(),
				"SubnetAz":        subnet.AvailabilityZone(),
				"RouteTableId":    subnet.RouteTable().RouteTableId(),
				"DestinationCidr": "0.0.0.0/0",
			},
		})
	}

	pubSubs := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Public"),
	})

	// Create a custom resource for each Public subnet
	for _, subnet := range *pubSubs {
		// Create CloudFormation custom resource to update firewall routing
		awscdk.NewCustomResource(stack, jsii.String(fmt.Sprintf("ReturnRoute-%s", strings.SplitN(*subnet.Node().Path(), "/", 1))), &awscdk.CustomResourceProps{
			ServiceToken: customResource.ServiceToken(),
			Properties: &map[string]interface{}{
				"FirewallArn":     networkFw.AttrFirewallArn(),
				"SubnetAz":        subnet.AvailabilityZone(),
				"RouteTableId":    subnet.RouteTable().RouteTableId(),
				"DestinationCidr": orgCidr,
			},
		})
	}

	fwSubs := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Firewall_Subnet"),
	})

	// Create a Route for each FW subnet
	for _, subnet := range *fwSubs {
		// Create CloudFormation custom resource to update firewall routing
		ec2.NewCfnRoute(stack, jsii.String(fmt.Sprintf("OrganisationRoute-%s", strings.SplitN(*subnet.Node().Path(), "/", 1))), &ec2.CfnRouteProps{
			RouteTableId:         subnet.RouteTable().RouteTableId(),
			DestinationCidrBlock: &orgCidr,
			TransitGatewayId:     transitGWId,
		}).AddDependency(tGWAttachment)
	}
}
