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

from re import A
from aws_cdk import (
    Arn,
    ArnComponents,
    CfnTag,
    CustomResource,
    Duration,
    Fn,
    RemovalPolicy,
    Stack,
)
from aws_cdk import aws_ec2 as ec2
from aws_cdk import aws_iam as iam
from aws_cdk import aws_lambda as _lambda
from aws_cdk import aws_logs as logs
from aws_cdk import aws_networkfirewall as nf
from aws_cdk import custom_resources as cr
from constructs import Construct

from .firewall_rules import NetworkFirewallRules


class NetworkFirewallStack(Stack):
    def __init__(
        self,
        scope: Construct,
        construct_id: str,
        cidr: str,
        organisation_cidr: str,
        transit_gateway_id: str,
        **kwargs,
    ) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # Inspection VPC for AWS Network Firewall
        # Creates three subnet groups that are used for TGW attachment,
        # AWS Network Firewall endpoints and NAT gateways.
        vpc = ec2.Vpc(
            self,
            "InspectionVPC",
            vpc_name="inspection-vpc",
            ip_addresses=ec2.IpAddresses.cidr(cidr),
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    name="tgw_subnet",
                    subnet_type=ec2.SubnetType.PRIVATE_ISOLATED,
                    cidr_mask=26,
                ),
                ec2.SubnetConfiguration(
                    name="firewall_subnet",
                    subnet_type=ec2.SubnetType.PRIVATE_WITH_EGRESS,
                    cidr_mask=27,
                ),
                ec2.SubnetConfiguration(
                    name="public", subnet_type=ec2.SubnetType.PUBLIC, cidr_mask=27
                ),
            ],
        )

        tgw_subnets = vpc.select_subnets(subnet_group_name="tgw_subnet")

        tgw_attachment = ec2.CfnTransitGatewayAttachment(
            self,
            "tgw_attachment",
            transit_gateway_id=transit_gateway_id,
            subnet_ids=tgw_subnets.subnet_ids,
            vpc_id=vpc.vpc_id,
            options=ec2.CfnTransitGatewayAttachment.OptionsProperty(
                appliance_mode_support="ENABLED"
            ),
            tags=[CfnTag(key="routeTable", value="inspection")],
        )

        # Default TGW route from Workload VPC to Inspection for egress.
        ec2.CfnTransitGatewayRoute(
            self,
            "inspection-tgw-route",
            destination_cidr_block="0.0.0.0/0",
            transit_gateway_attachment_id=tgw_attachment.attr_id,
            transit_gateway_route_table_id=Fn.import_value("WorkloadRouteTableId"),
        )

        # Create rules for AWS Network Firewall
        # Rules are defined in a separate file
        firewall_rules = NetworkFirewallRules(self, "NetworkFirewallRules")

        subnet_list = [
            nf.CfnFirewall.SubnetMappingProperty(subnet_id=subnet.subnet_id)
            for subnet in vpc.select_subnets(subnet_group_name="firewall_subnet").subnets
        ]

        network_fw = nf.CfnFirewall(
            self,
            "network_firewall",
            firewall_name="EgressInspectionFirewall",
            firewall_policy_arn=firewall_rules.firewall_policy.attr_firewall_policy_arn,
            subnet_mappings=subnet_list,
            vpc_id=vpc.vpc_id,
        )

        fw_flow_logs_group = logs.LogGroup(
            self,
            "FWFlowLogsGroup",
            log_group_name="NetworkFirewallFlowLogs",
            removal_policy=RemovalPolicy.DESTROY,
        )

        fw_alert_logs_group = logs.LogGroup(
            self,
            "FWAlertLogsGroup",
            log_group_name="NetworkFirewallAlertLogs",
            removal_policy=RemovalPolicy.DESTROY,
        )

        nf.CfnLoggingConfiguration(
            self,
            "FirewallLoggingConfg",
            firewall_arn=network_fw.ref,
            logging_configuration=nf.CfnLoggingConfiguration.LoggingConfigurationProperty(
                log_destination_configs=[
                    nf.CfnLoggingConfiguration.LogDestinationConfigProperty(
                        log_destination={"logGroup": fw_flow_logs_group.log_group_name},
                        log_destination_type="CloudWatchLogs",
                        log_type="FLOW",
                    ),
                    nf.CfnLoggingConfiguration.LogDestinationConfigProperty(
                        log_destination={"logGroup": fw_alert_logs_group.log_group_name},
                        log_destination_type="CloudWatchLogs",
                        log_type="ALERT",
                    ),
                ]
            ),
        )

        # Lambda function and custom action to create and delete routes to
        # Gateway Load Balancer endpoints in correct AZ
        route_lambda_role = iam.Role(
            self,
            "routeLambdaRole",
            assumed_by=iam.ServicePrincipal("lambda.amazonaws.com"),
            managed_policies=[
                iam.ManagedPolicy.from_aws_managed_policy_name(
                    "service-role/AWSLambdaBasicExecutionRole"
                )
            ],
        )

        route_lambda_role.add_to_policy(
            iam.PolicyStatement(
                effect=iam.Effect.ALLOW,
                actions=["network-firewall:DescribeFirewall"],
                resources=[network_fw.attr_firewall_arn],
            )
        )

        cloud_wan_subnets_arns = [
            Arn.format(
                ArnComponents(
                    service="ec2",
                    resource=f"route-table/{subnet.route_table.route_table_id}",
                ),
                stack=self,
            )
            for subnet in vpc.select_subnets(subnet_group_name="tgw_subnet").subnets
        ]

        public_subnet_arns = [
            Arn.format(
                ArnComponents(
                    service="ec2",
                    resource=f"route-table/{subnet.route_table.route_table_id}",
                ),
                stack=self,
            )
            for subnet in vpc.select_subnets(subnet_group_name="public").subnets
        ]

        route_lambda_role.add_to_policy(
            iam.PolicyStatement(
                effect=iam.Effect.ALLOW,
                actions=["ec2:CreateRoute", "ec2:DeleteRoute"],
                resources=cloud_wan_subnets_arns + public_subnet_arns,
            )
        )

        custom_route_lambda = _lambda.Function(
            self,
            "RoutesFunction",
            runtime=_lambda.Runtime.PYTHON_3_9,
            handler="index.lambda_handler",
            role=route_lambda_role,
            timeout=Duration.seconds(20),
            code=_lambda.Code.from_asset("lambda/routes"),
        )

        provider = cr.Provider(
            self,
            "provider",
            on_event_handler=custom_route_lambda,
            log_retention=logs.RetentionDays.ONE_DAY,
        )
        # TODO: not copied over the dependency on the Lambda as it should be
        # implicit; verify.

        # Create a default route towards AWS Network firewall endpoints. Select
        # all subnets in group CloudWANAttacment and use custom lambda function
        # to find AWS Network Firewall endpoint that is in same availability
        # zone as subnet.
        for subnet in vpc.select_subnets(subnet_group_name="tgw_subnet").subnets:
            subnet_name = subnet.node.path.split("/")[-1]
            CustomResource(
                self,
                f"FirewallRoute-{subnet_name}",
                properties={
                    "FirewallArn": network_fw.attr_firewall_arn,
                    "SubnetAz": subnet.availability_zone,
                    "RouteTableId": subnet.route_table.route_table_id,
                    "DestinationCidr": "0.0.0.0/0",
                },
                service_token=provider.service_token,
            )

        for subnet in vpc.select_subnets(subnet_group_name="public").subnets:
            subnet_name = subnet.node.path.split("/")[-1]
            CustomResource(
                self,
                f"ReturnRoute-{subnet_name}",
                properties={
                    "FirewallArn": network_fw.attr_firewall_arn,
                    "SubnetAz": subnet.availability_zone,
                    "RouteTableId": subnet.route_table.route_table_id,
                    "DestinationCidr": organisation_cidr,
                },
                service_token=provider.service_token,
            )

        # Route back towards workload VPCs. Organisation CIDR as the target.
        for subnet in vpc.select_subnets(subnet_group_name="firewall_subnet").subnets:
            subnet_name = subnet.node.path.split("/")[-1]
            ec2.CfnRoute(
                self,
                f"organisation-route-{subnet_name}",
                route_table_id=subnet.route_table.route_table_id,
                destination_cidr_block=organisation_cidr,
                transit_gateway_id=transit_gateway_id,
            ).node.add_dependency(tgw_attachment)
