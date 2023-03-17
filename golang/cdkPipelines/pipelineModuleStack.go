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
	codecommit "github.com/aws/aws-cdk-go/awscdk/v2/awscodecommit"
	"github.com/aws/aws-cdk-go/awscdk/v2/pipelines"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type PipelineStackProps struct {
	awscdk.StackProps
}

func NetworkPipelineStack(scope constructs.Construct, id string, props *PipelineStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	sourceRepo := codecommit.NewRepository(stack, jsii.String("sourceRepo"), &codecommit.RepositoryProps{
		RepositoryName: jsii.String("network-pipeline-repo"),
	})

	pipeline := pipelines.NewCodePipeline(stack, jsii.String("cdkpipeline"), &pipelines.CodePipelineProps{
		PipelineName: jsii.String("WorkshopPipeline"),
		Synth: pipelines.NewShellStep(jsii.String("build-and-synth"), &pipelines.ShellStepProps{
			Input: pipelines.CodePipelineSource_CodeCommit(sourceRepo, jsii.String("main"), nil),
			Commands: jsii.Strings(
				"npm install -g aws-cdk",
				"goenv install 1.18.3",
				"goenv local 1.18.3",
				"npx cdk synth",
			),
		}),
	})

	deployFirewallStack := NetworkWorkshopInspectStage(
		stack, "DeployInspection", &NetworkWorkshopInspectStageProps{
			awscdk.StageProps{
				Env: &HubEnv,
			},
		})

	pipeline.AddStage(deployFirewallStack, nil)

	return stack
}
