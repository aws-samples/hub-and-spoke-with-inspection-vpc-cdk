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

from aws_cdk import CfnTag, Stack, CfnOutput
from aws_cdk import aws_ec2 as ec2
from aws_cdk import aws_iam as iam
from constructs import Construct


class WorkloadStack(Stack):
    def __init__(
        self, scope: Construct, construct_id: str, transit_gateway_id: str, cidr: str, **kwargs
    ) -> None:

        super().__init__(scope, construct_id, **kwargs)

        # VPC settings
        vpc = ec2.Vpc(
            self,
            "vpc",
            max_azs=2,
            ip_addresses=ec2.IpAddresses.cidr(cidr),
            enable_dns_hostnames=True,
            enable_dns_support=True,
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    subnet_type=ec2.SubnetType.PRIVATE_ISOLATED,
                    name="private",
                    cidr_mask=24,
                ),
            ],
        )

        CfnOutput(self, "vpc_id", value=vpc.vpc_id)

        private_subnets = vpc.select_subnets(subnet_group_name="private")

        tgw_attachment = ec2.CfnTransitGatewayAttachment(
            self,
            "tgw_attachment",
            transit_gateway_id=transit_gateway_id,
            subnet_ids=private_subnets.subnet_ids,
            vpc_id=vpc.vpc_id,
            tags=[CfnTag(key="routeTable", value="workload")],
        )

        # Default route towards Transit Gateway
        for subnet in private_subnets.subnets:
            subnet_name = subnet.node.path.split("/")[-1]
            ec2.CfnRoute(
                self,
                f"default-route-{subnet_name}",
                destination_cidr_block="0.0.0.0/0",
                transit_gateway_id=transit_gateway_id,
                route_table_id=subnet.route_table.route_table_id,
            ).node.add_dependency(tgw_attachment)

        vpc.add_interface_endpoint("SsmEndpoint", service=ec2.InterfaceVpcEndpointAwsService.SSM)
        vpc.add_interface_endpoint(
            "SsmMessagesEndpoint",
            service=ec2.InterfaceVpcEndpointAwsService.SSM_MESSAGES,
        )
        vpc.add_interface_endpoint(
            "Ec2MessagesEndpoint",
            service=ec2.InterfaceVpcEndpointAwsService.EC2_MESSAGES,
        )

        ssm_role = iam.Role(
            self,
            "SSMRole",
            assumed_by=iam.ServicePrincipal("ec2.amazonaws.com"),
            managed_policies=[
                iam.ManagedPolicy.from_aws_managed_policy_name("AmazonSSMManagedInstanceCore")
            ],
        )

        security_group = ec2.SecurityGroup(
            self, "WorkloadEC2SG", security_group_name="workload-sg", vpc=vpc
        )

        security_group.add_ingress_rule(ec2.Peer.any_ipv4(), ec2.Port.all_icmp())

        ec2.Instance(
            self,
            "WorkloadEC2",
            vpc=vpc,
            vpc_subnets=ec2.SubnetSelection(subnet_type=ec2.SubnetType.PRIVATE_ISOLATED),
            instance_type=ec2.InstanceType.of(
                ec2.InstanceClass.BURSTABLE4_GRAVITON, ec2.InstanceSize.MICRO
            ),
            machine_image=ec2.MachineImage.latest_amazon_linux(
                cpu_type=ec2.AmazonLinuxCpuType.ARM_64,
                generation=ec2.AmazonLinuxGeneration.AMAZON_LINUX_2,
            ),
            role=ssm_role,
            security_group=security_group,
        )
