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
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type InspectionWorkloadStackProps struct {
	awscdk.StackProps
	cidr        string
	transitGWId *string
}

func InspectionWorkloadStack(scope constructs.Construct, id string, props *InspectionWorkloadStackProps) {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := ec2.NewVpc(stack, jsii.String("vpc"), &ec2.VpcProps{
		MaxAzs:             jsii.Number(2),
		IpAddresses:        ec2.IpAddresses_Cidr(&props.cidr),
		EnableDnsSupport:   jsii.Bool(true),
		EnableDnsHostnames: jsii.Bool(true),
		SubnetConfiguration: &[]*ec2.SubnetConfiguration{
			{
				Name:       jsii.String("Private"),
				SubnetType: ec2.SubnetType_PRIVATE_ISOLATED,
				CidrMask:   jsii.Number(24),
			},
		},
	})

	awscdk.NewCfnOutput(stack, jsii.String("vpc_id"), &awscdk.CfnOutputProps{Value: vpc.VpcId()})

	privateSubs := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Private"),
	})
	var privateSubsIds []*string

	for _, privateSub := range *privateSubs {
		privateSubId := privateSub.SubnetId()
		privateSubsIds = append(privateSubsIds, privateSubId)
	}

	tGWAttachment := ec2.NewCfnTransitGatewayAttachment(stack, jsii.String("TGW_Attachment"), &ec2.CfnTransitGatewayAttachmentProps{
		TransitGatewayId: props.transitGWId
		SubnetIds:        &privateSubsIds,
		VpcId:            vpc.VpcId(),
		Tags: &[]*awscdk.CfnTag{
			{
				Key:   jsii.String("routeTable"),
				Value: jsii.String("workload"),
			},
		},
	})

	subs := vpc.SelectSubnetObjects(&ec2.SubnetSelection{
		SubnetGroupName: jsii.String("Private"),
	})

	// Create a custom resource for each TGw subnet
	for _, subnet := range *subs {
		ec2.NewCfnRoute(stack, jsii.String(fmt.Sprintf("Default-route-%s", strings.SplitN(*subnet.Node().Path(), "/", 1))), &ec2.CfnRouteProps{
			RouteTableId:         subnet.RouteTable().RouteTableId(),
			DestinationCidrBlock: jsii.String("0.0.0.0/0"),
			TransitGatewayId:     transitGWId,
		}).AddDependency(tGWAttachment)
	}

	vpc.AddInterfaceEndpoint(jsii.String("SSMEndpoint"), &ec2.InterfaceVpcEndpointOptions{Service: ec2.InterfaceVpcEndpointAwsService_SSM()})
	vpc.AddInterfaceEndpoint(jsii.String("SSMMessagesEndpoint"), &ec2.InterfaceVpcEndpointOptions{Service: ec2.InterfaceVpcEndpointAwsService_SSM_MESSAGES()})
	vpc.AddInterfaceEndpoint(jsii.String("Ec2MessagesEndpoint"), &ec2.InterfaceVpcEndpointOptions{Service: ec2.InterfaceVpcEndpointAwsService_EC2_MESSAGES()})

	SSMRole := iam.NewRole(stack, jsii.String("SSMRole"), &iam.RoleProps{
		AssumedBy:       iam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
		ManagedPolicies: &[]iam.IManagedPolicy{iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore"))},
	})

	securityGroup := ec2.NewSecurityGroup(stack, jsii.String("WorkloadEC2SG"), &ec2.SecurityGroupProps{
		SecurityGroupName: jsii.String("workload-sg"),
		Vpc:               vpc,
	})

	securityGroup.AddIngressRule(ec2.Peer_AnyIpv4(), ec2.Port_AllIcmp(), jsii.String("WorkloadIngressRule"), jsii.Bool(true))

	ec2.NewInstance(stack, jsii.String("WorkloadEC2"), &ec2.InstanceProps{
		Vpc:          vpc,
		VpcSubnets:   &ec2.SubnetSelection{SubnetType: ec2.SubnetType_PRIVATE_ISOLATED},
		InstanceType: ec2.InstanceType_Of(ec2.InstanceClass_BURSTABLE4_GRAVITON, ec2.InstanceSize_MICRO),
		MachineImage: ec2.MachineImage_LatestAmazonLinux(&ec2.AmazonLinuxImageProps{
			CpuType:    ec2.AmazonLinuxCpuType_ARM_64,
			Generation: ec2.AmazonLinuxGeneration_AMAZON_LINUX_2,
		}),
		Role:          SSMRole,
		SecurityGroup: securityGroup,
	})
}
