# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.

# Permission is hereby granted, free of charge, to any person obtaining a copy of this
# software and associated documentation files (the "Software"), to deal in the Software
# without restriction, including without limitation the rights to use, copy, modify,
# merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
# INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
# PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
# HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
# OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
# SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

import os.path as path
from aws_cdk import Stack
from aws_cdk import aws_codecommit as codecommit
from aws_cdk import pipelines as cdk_pipeline
from constructs import Construct

from cdk_pipelines.inspection_pipeline_stage import NetworkWorkshopInspectionStage
from cdk_pipelines.configurations import HUB_ENV


class NetworkPipelineStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        source_repo = codecommit.Repository(
            self, "source_repo", repository_name="network-pipeline-repo"
        )

        pipeline = cdk_pipeline.CodePipeline(
            self,
            "cdkpipeline",
            synth=cdk_pipeline.ShellStep(
                "build-and-synth",
                input=cdk_pipeline.CodePipelineSource.code_commit(source_repo, "main"),
                commands=[
                    "npm install -g aws-cdk",
                    "pip install -r requirements.txt",
                    "cdk synth",
                ],
            ),
        )

        # Inspection Module Stage
        deploy_firewall_stack = NetworkWorkshopInspectionStage(
            self, "DeployInspection", env=HUB_ENV
        )

        pipeline.add_stage(deploy_firewall_stack)
