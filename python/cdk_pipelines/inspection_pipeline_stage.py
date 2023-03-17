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

from aws_cdk import Stage
from constructs import Construct

from cdk_pipelines.inspection_transit_gateway import TransitGatewayStack
from cdk_pipelines.inspection_stack import NetworkFirewallStack
from cdk_pipelines.inspection_vpc_workload_stack import WorkloadStack
from cdk_pipelines.configurations import ORGANISATION_CIDR


class NetworkWorkshopInspectionStage(Stage):
    def __init__(self, scope: Construct, id: str, **kwargs):
        super().__init__(scope, id, **kwargs)

        tgw = TransitGatewayStack(self, "TransitGateway")

        NetworkFirewallStack(
            self,
            "Inspection",
            cidr="10.100.0.0/16",
            organisation_cidr=ORGANISATION_CIDR,
            transit_gateway_id=tgw.transit_gateway.attr_id,
        )

        WorkloadStack(
            self, "Workload", transit_gateway_id=tgw.transit_gateway.attr_id, cidr="10.110.0.0/16"
        )

        WorkloadStack(
            self, "Workload2", transit_gateway_id=tgw.transit_gateway.attr_id, cidr="10.111.0.0/16"
        )
