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
	"github.com/aws/aws-cdk-go/awscdk/v2"
	ec2 "github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type VPCInMonitorStackProps struct {
	awscdk.StackProps
}

type vpcInMonitorStackOutputs struct {
	constructs.Construct
	vpc             ec2.Vpc
	publicSubnets   *[]ec2.ISubnet
	privateSubnets  *[]ec2.ISubnet
	isolatedSubnets *[]ec2.ISubnet
}

type IvpcInMonitorStac interface {
	constructs.Construct
	Vpc() ec2.Vpc
	PublicSubnets() *[]ec2.ISubnet
	PrivateSubnets() *[]ec2.ISubnet
	IsolatedSubnets() *[]ec2.ISubnet
}

func (o *vpcInMonitorStackOutputs) PublicSubnets() *[]ec2.ISubnet {
	return o.publicSubnets
}

func (o *vpcInMonitorStackOutputs) PrivateSubnets() *[]ec2.ISubnet {
	return o.privateSubnets
}

func (o *vpcInMonitorStackOutputs) IsolatedSubnets() *[]ec2.ISubnet {
	return o.isolatedSubnets
}

func (o *vpcInMonitorStackOutputs) Vpc() ec2.Vpc {
	return o.vpc
}

func VPCInMonitorStack(scope constructs.Construct, id string, cidr string, props *VPCInMonitorStackProps) IvpcInMonitorStac {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := ec2.NewVpc(stack, jsii.String("vpc"), &ec2.VpcProps{
		MaxAzs:             jsii.Number(2),
		IpAddresses:        ec2.IpAddresses_Cidr(&cidr),
		EnableDnsHostnames: jsii.Bool(true),
		EnableDnsSupport:   jsii.Bool(true),
		SubnetConfiguration: &[]*ec2.SubnetConfiguration{
			{
				SubnetType: ec2.SubnetType_PUBLIC,
				Name:       jsii.String("public"),
				CidrMask:   jsii.Number(24),
			},
			{
				SubnetType: ec2.SubnetType_PRIVATE_WITH_EGRESS,
				Name:       jsii.String("private"),
				CidrMask:   jsii.Number(24),
			},
			{
				SubnetType: ec2.SubnetType_PRIVATE_ISOLATED,
				Name:       jsii.String("isolated"),
				CidrMask:   jsii.Number(24),
			},
		},
		NatGateways: jsii.Number(2),
	})

	awscdk.NewCfnOutput(stack, jsii.String("vpcId"), &awscdk.CfnOutputProps{
		Value:      vpc.VpcId(),
		ExportName: jsii.String("vpcId"),
	})

	return &vpcInMonitorStackOutputs{stack, vpc, vpc.IsolatedSubnets(), vpc.PrivateSubnets(), vpc.PublicSubnets()}
}
