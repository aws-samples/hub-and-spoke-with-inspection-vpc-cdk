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

import logging
from typing import Dict
import boto3
import botocore
import json
import urllib3

logger = logging.getLogger(__name__)

http = urllib3.PoolManager()
nfw = boto3.client("network-firewall")
ec2 = boto3.client("ec2")



def send_response(event, context, response, responseData):
    '''Send a response to CloudFormation to handle the custom resource.'''

    responseBody = {
        'Status': response,
        'Reason': f'See details in CloudWatch Log Stream: {context.log_stream_name}',
        'PhysicalResourceId': context.log_stream_name,
        'StackId': event['StackId'],
        'RequestId': event['RequestId'],
        'LogicalResourceId': event['LogicalResourceId'],
        'Data': responseData
    }

    logger.info('RESPONSE BODY: \n' + json.dumps(responseBody))

    responseUrl = event['ResponseURL']
    json_responseBody = json.dumps(responseBody)
    headers = {
        'content-type': '',
        'content-length': str(len(json_responseBody))
    }
    try:
        response = http.request('PUT', responseUrl, headers=headers,
                                body=json_responseBody)
        logger.info("Status code: " + response.status)

    except Exception as e:
        logger.warning("send(..) failed executing requests.put(..): " + str(e))

    return True

def get_data(firewall_arn: str) -> Dict[str, str]:
    response = nfw.describe_firewall(FirewallArn=firewall_arn)
    return {
        k: v["Attachment"]["EndpointId"]
        for k, v in response["FirewallStatus"]["SyncStates"].items()
    }

def lambda_handler(event, context):
    if event['RequestType'] == 'Create':
        try:
            firewall_arn = event["ResourceProperties"]["FirewallArn"]
            subnet_az = event["ResourceProperties"]["SubnetAz"]
            destination_cidr = event["ResourceProperties"]["DestinationCidr"]
            route_table_id = event["ResourceProperties"]["RouteTableId"]

            endpoints = get_data(firewall_arn)
            ec2.create_route(
                DestinationCidrBlock=destination_cidr,
                RouteTableId=route_table_id,
                VpcEndpointId=endpoints[subnet_az],
            )
            response = 'SUCCESS'
            data = {}
        except botocore.exceptions.ClientError as error:
            logger.error(f"error due to {error}")
            response = 'FAILED'
            data = {"Error": "Create route failed for firewall"}

        send_response(event, context, response, data)

    elif event['RequestType'] == 'Update':
        response = 'SUCCESS'
        data = {}
        send_response(event, context, response, data)
    elif event['RequestType'] == 'Delete':
        try:
            route_table_id = event["ResourceProperties"]["RouteTableId"]
            destination_cidr = event["ResourceProperties"]["DestinationCidr"]
            ec2.delete_route(DestinationCidrBlock=destination_cidr,
                            RouteTableId=route_table_id)
            response = 'SUCCESS'
            data = {}
        except botocore.exceptions.ClientError as error:
            logger.error(f"error due to {error}")
            response = 'FAILED'
            data = {"Error": "Delete route failed for firewall"}

        send_response(event, context, response, data)



