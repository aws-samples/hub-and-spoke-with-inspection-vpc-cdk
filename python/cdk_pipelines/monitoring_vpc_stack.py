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

from aws_cdk import CfnOutput, Stack
from aws_cdk import aws_ec2 as ec2
from constructs import Construct


class VpcStack(Stack):
    def __init__(self, scope: Construct, construct_id: str, cidr: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # VPC settings
        self.vpc = ec2.Vpc(
            self,
            "vpc",
            max_azs=2,
            ip_addresses=ec2.IpAddresses.cidr(cidr),
            enable_dns_hostnames=True,
            enable_dns_support=True,
            # configuration will create 3 groups in 2 AZs = 6 subnets.
            subnet_configuration=[
                ec2.SubnetConfiguration(
                    subnet_type=ec2.SubnetType.PUBLIC, name="public", cidr_mask=24
                ),
                ec2.SubnetConfiguration(
                    subnet_type=ec2.SubnetType.PRIVATE_WITH_EGRESS,
                    name="private",
                    cidr_mask=24,
                ),
                ec2.SubnetConfiguration(
                    subnet_type=ec2.SubnetType.PRIVATE_ISOLATED,
                    name="isolated",
                    cidr_mask=24,
                ),
            ],
            nat_gateways=2,
        )

        CfnOutput(self, "vpc_id", value=self.vpc.vpc_id)

    @property
    def public_subnets(self):
        return self.vpc.public_subnets

    @property
    def private_subnets(self):
        return self.vpc.private_subnets

    @property
    def isolated_subnets(self):
        return self.vpc.isolated_subnets
