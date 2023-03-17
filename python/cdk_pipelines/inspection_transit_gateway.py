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

from aws_cdk import CfnOutput, CfnTag, Duration, Stack
from aws_cdk import aws_ec2 as ec2
from aws_cdk import aws_events as events
from aws_cdk import aws_events_targets as targets
from aws_cdk import aws_iam as iam
from aws_cdk import aws_lambda as _lambda
from constructs import Construct


class TransitGatewayStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # Create TGW
        transit_gateway = ec2.CfnTransitGateway(
            self,
            "transit_gateway",
            description="TransitGateway-" + self.region,
            default_route_table_association="disable",
            default_route_table_propagation="disable",
            tags=[CfnTag(key="Name", value=self.region + "transit-gateway")],
        )

        workload_rt = ec2.CfnTransitGatewayRouteTable(
            self,
            "workload-route-table",
            transit_gateway_id=transit_gateway.attr_id,
            tags=[CfnTag(key="Name", value="workload")],
        )

        CfnOutput(
            self,
            "workload-rt-output",
            value=workload_rt.ref,
            export_name="WorkloadRouteTableId",
        )

        inspection_rt = ec2.CfnTransitGatewayRouteTable(
            self,
            "inspection-route-table",
            transit_gateway_id=transit_gateway.attr_id,
            tags=[CfnTag(key="Name", value="inspection-route-table")],
        )
        CfnOutput(
            self,
            "inspeciton-rt-output",
            value=inspection_rt.ref,
            export_name="InspectionRouteTableId",
        )

        self.transit_gateway = transit_gateway

        self._create_event_handling()

    def _create_event_handling(self):
        attachment_lambda_role = iam.Role(
            self,
            "attachmentLambdaRole",
            assumed_by=iam.ServicePrincipal("lambda.amazonaws.com"),
            managed_policies=[
                iam.ManagedPolicy.from_aws_managed_policy_name(
                    "service-role/AWSLambdaBasicExecutionRole"
                )
            ],
        )

        attachment_lambda_role.add_to_policy(
            iam.PolicyStatement(
                actions=[
                    "ec2:AssociateTransitGatewayRouteTable",
                    "ec2:DescribeTransitGatewayAttachments",
                    "ec2:DisassociateTransitGatewayRouteTable",
                    "ec2:EnableTransitGatewayRouteTablePropagation",
                    "cloudformation:ListExports",
                ],
                effect=iam.Effect.ALLOW,
                resources=["*"],
            )
        )

        tgw_route_lambda = _lambda.Function(
            self,
            "TGWAttachmentFunction",
            runtime=_lambda.Runtime.PYTHON_3_9,
            handler="index.handler",
            role=attachment_lambda_role,
            timeout=Duration.seconds(60),
            code=_lambda.Code.from_asset("lambda/attachment"),
        )

        event_pattern = events.EventPattern(
            source=["aws.ec2"],
            detail={"eventName": ["CreateTransitGatewayVpcAttachment"]},
        )

        events.Rule(
            self,
            "TGWAttachmentCreated",
            event_pattern=event_pattern,
            targets=[targets.LambdaFunction(tgw_route_lambda)],
        )
