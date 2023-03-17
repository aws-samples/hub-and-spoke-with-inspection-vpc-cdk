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

import time

import boto3

ec2_client = boto3.client("ec2")
cfn_client = boto3.client("cloudformation")


def handler(event, context):

    print(event)
    if event["detail"]["eventName"] != "CreateTransitGatewayVpcAttachment":
        raise RuntimeError("Incorrect event type")

    vpc_type = event["detail"]["requestParameters"]["CreateTransitGatewayVpcAttachmentRequest"][
        "TagSpecifications"
    ]["Tag"]["Value"]
    attachment_id = event["detail"]["responseElements"][
        "CreateTransitGatewayVpcAttachmentResponse"
    ]["transitGatewayVpcAttachment"]["transitGatewayAttachmentId"]

    # Route Table Ids
    exports = cfn_client.list_exports()
    for export in exports["Exports"]:
        if export["Name"] == "WorkloadRouteTableId":
            workload_route_table_id = export["Value"]
        if export["Name"] == "InspectionRouteTableId":
            inspection_route_table_id = export["Value"]

    # Route table id for association
    association_route_table_id = ""
    if vpc_type == "workload":
        association_route_table_id = workload_route_table_id
    elif vpc_type == "inspection":
        association_route_table_id = inspection_route_table_id

    # Get attachment ID
    describe_response = ec2_client.describe_transit_gateway_attachments(
        TransitGatewayAttachmentIds=[attachment_id]
    )

    associated = False
    if "Association" in describe_response["TransitGatewayAttachments"][0].keys():
        print("Attachment was associated. Removing association")
        disassociate_response = ec2_client.disassociate_transit_gateway_route_table(
            TransitGatewayAttachmentId=attachment_id,
            TransitGatewayRouteTableId=describe_response["TransitGatewayAttachments"][0][
                "Association"
            ]["TransitGatewayRouteTableId"],
        )
        associated = True
        print(disassociate_response)

    # If attachment is already associated, remove association and then attach again.
    while associated:
        describe_response = ec2_client.describe_transit_gateway_attachments(
            TransitGatewayAttachmentIds=[attachment_id]
        )
        if "Association" in describe_response["TransitGatewayAttachments"][0].keys():
            time.sleep(2)
            print("Sleeping for 2 seconds to wait for disassociation")
        else:
            time.sleep(2)
            associated = False

    print("Associatie attachment to route table")
    ec2_client.associate_transit_gateway_route_table(
        TransitGatewayAttachmentId=attachment_id,
        TransitGatewayRouteTableId=association_route_table_id,
    )

    # Enable propagation to Inspection route table.
    if vpc_type == "workload":
        print("Enable route propagation to inspection route table")
        ec2_client.enable_transit_gateway_route_table_propagation(
            TransitGatewayRouteTableId=inspection_route_table_id,
            TransitGatewayAttachmentId=attachment_id,
        )
