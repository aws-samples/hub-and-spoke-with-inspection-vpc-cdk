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

import * as cdk from 'aws-cdk-lib';
import * as codecommit from 'aws-cdk-lib/aws-codecommit';
import {
  CodeBuildStep,
  CodePipeline,
  CodePipelineSource,
} from 'aws-cdk-lib/pipelines';
import { Construct } from 'constructs';
import { NetworkWorkshopInspectionStage } from '../lib/inspectionPipelineStage';
import { HUB_ENV } from './configurations';

export class NetworkPipelineStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const sourceRepo = new codecommit.Repository(
      this,
      'NetworkPipelineRepo',
      {
        repositoryName: 'network-pipeline-repo',
      }
    );

    const pipeline = new CodePipeline(this, 'CdkPipeline', {
      pipelineName: 'WorkshopPipeline',
      synth: new CodeBuildStep('build-and-synth', {
        input: CodePipelineSource.codeCommit(sourceRepo, 'main'),
        installCommands: ['npm ci'],
        commands: ['npm run build', 'npx cdk synth'],
      }),
      dockerEnabledForSynth: true,
    });

    const deployFirewallStack = new NetworkWorkshopInspectionStage(
      this,
      'DeployInspection',
      {
        env: HUB_ENV,
      }
    );

    pipeline.addStage(deployFirewallStack);
  }
}
