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
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	iam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type InspectionTgwStackProps struct {
	awscdk.StackProps
}

type InspectionTgwStackOutputs struct {
	awscdk.Stack
	tgWId *string
}

func InspectionTgwStack(scope constructs.Construct, id string, props *InspectionTgwStackProps) InspectionTgwStackOutputs {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	TransitGateway := ec2.NewCfnTransitGateway(stack, jsii.String("TransitGateway"), &ec2.CfnTransitGatewayProps{
		Description:                  jsii.String("TransitGateway"), //To do: Region suffix
		DefaultRouteTableAssociation: jsii.String("disable"),
		DefaultRouteTablePropagation: jsii.String("disable"),
		Tags: &[]*awscdk.CfnTag{
			{
				Key:   jsii.String("Name"),
				Value: jsii.String("transit-gateway"), //To do: Region suffix/prefix
			},
		},
	})

	WorkLoadRt := ec2.NewCfnTransitGatewayRouteTable(stack, jsii.String("WorkloadRouteTable"), &ec2.CfnTransitGatewayRouteTableProps{
		TransitGatewayId: TransitGateway.AttrId(),
		Tags: &[]*awscdk.CfnTag{
			{
				Key:   jsii.String("Name"),
				Value: jsii.String("Workload"), //To do: Region suffix/prefix
			},
		},
	})

	awscdk.NewCfnOutput(stack, jsii.String("workload-rt-output"), &awscdk.CfnOutputProps{
		Value:      WorkLoadRt.Ref(),
		ExportName: jsii.String("WorkloadRouteTableId"),
	})

	InspectionRt := ec2.NewCfnTransitGatewayRouteTable(stack, jsii.String("inspection-route-table"), &ec2.CfnTransitGatewayRouteTableProps{
		TransitGatewayId: TransitGateway.AttrId(),
		Tags: &[]*awscdk.CfnTag{
			{
				Key:   jsii.String("Name"),
				Value: jsii.String("InspectionRouteTable"), //To do: Region suffix/prefix
			},
		},
	})
	awscdk.NewCfnOutput(stack, jsii.String("inspection-rt-output"), &awscdk.CfnOutputProps{
		Value:      InspectionRt.Ref(),
		ExportName: jsii.String("InspectionRouteTableId"),
	})

	//To do/check from Py project: Self.TransitGateway = TransitGateway

	CreateEventHandling(stack)

	var outputs InspectionTgwStackOutputs
	outputs.Stack = stack
	outputs.tgWId = TransitGateway.AttrId()

	return outputs
}

func CreateEventHandling(scope constructs.Construct) {
	AWSLambdaBasicExecPolicy := iam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore"))

	AttachmentLambdaRole := iam.NewRole(scope, jsii.String("attachmentLambdaRole"), &iam.RoleProps{
		AssumedBy:       iam.NewServicePrincipal(jsii.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]iam.IManagedPolicy{AWSLambdaBasicExecPolicy},
	})

	AttachmentLambdaRole.AddToPolicy(
		iam.NewPolicyStatement(&iam.PolicyStatementProps{
			Actions: &[]*string{
				jsii.String("ec2:AssociateTransitGatewayRouteTable"),
				jsii.String("ec2:DescribeTransitGatewayAttachments"),
				jsii.String("ec2:DisassociateTransitGatewayRouteTable"),
				jsii.String("ec2:EnableTransitGatewayRouteTablePropagation"),
				jsii.String("cloudformation:ListExports"),
			},
			Effect: iam.Effect_ALLOW,
			Resources: &[]*string{
				jsii.String("*"),
			},
		}),
	)

	TgwRouteLambda := lambda.NewFunction(scope, jsii.String("TGWAttachmentFunction"), &lambda.FunctionProps{
		Runtime: lambda.Runtime_PYTHON_3_9(),
		Handler: jsii.String("index.handler"),
		Role:    AttachmentLambdaRole,
		Timeout: awscdk.Duration_Seconds(jsii.Number(60)),
		Code:    lambda.Code_FromAsset(jsii.String("lambda/attachment"), nil),
	})

	EventPatternRule := awsevents.NewRule(scope, jsii.String("TGWAttachmentCreated"), &awsevents.RuleProps{
		EventPattern: &awsevents.EventPattern{
			Source: &[]*string{
				jsii.String("aws.ec2"),
			},
			Detail: &map[string]interface{}{
				"eventName": []interface{}{"CreateTransitGatewayVpcAttachment"},
			},
		},
	})

	EventPatternRule.AddTarget(awseventstargets.NewLambdaFunction(TgwRouteLambda, nil))

}
