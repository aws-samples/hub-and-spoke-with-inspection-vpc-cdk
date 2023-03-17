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
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import { Construct } from 'constructs';
import { NodejsFunction } from 'aws-cdk-lib/aws-lambda-nodejs';
import * as path from 'path';

export class TransitGatewayStack extends cdk.Stack {
  public readonly transitGateway: ec2.CfnTransitGateway;
  public readonly workloadRouteTable: ec2.CfnTransitGatewayRouteTable
  private readonly inspectionRouteTable: ec2.CfnTransitGatewayRouteTable

  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    this.transitGateway = new ec2.CfnTransitGateway(this, 'TransitGateway', {
      description: 'TransitGateway-' + this.region,
      defaultRouteTableAssociation: 'disable',
      defaultRouteTablePropagation: 'disable',
      tags: [
        {
          key: 'Name',
          value: this.region + 'transit-gateway',
        },
      ],
    });

    this.workloadRouteTable = new ec2.CfnTransitGatewayRouteTable(
      this,
      'WorkloadRouteTable',
      {
        transitGatewayId: this.transitGateway.attrId,
        tags: [
          {
            key: 'Name',
            value: 'workload-route-table',
          },
        ],
      }
    );

    this.inspectionRouteTable = new ec2.CfnTransitGatewayRouteTable(
      this,
      'InspectionRouteTable',
      {
        transitGatewayId: this.transitGateway.attrId,
        tags: [
          {
            key: 'Name',
            value: 'inspection-route-table',
          },
        ],
      }
    );

    this.createEventHandling();
  }

  private createEventHandling() {
    const attachmentLambdaRole = new iam.Role(this, 'AttachmentLambdaRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          'service-role/AWSLambdaBasicExecutionRole'
        ),
      ],
    });

    attachmentLambdaRole.addToPolicy(
      new iam.PolicyStatement({
        actions: [
          'ec2:AssociateTransitGatewayRouteTable',
          'ec2:DescribeTransitGatewayAttachments',
          'ec2:DisassociateTransitGatewayRouteTable',
          'ec2:EnableTransitGatewayRouteTablePropagation',
        ],
        effect: iam.Effect.ALLOW,
        resources: ['*'],
      })
    );

    const tgwRouteLambda = new NodejsFunction(this, 'TGWAttachmentFunction', {
      entry: path.join(__dirname, 'lambda/attachment/index.ts'),
      runtime: lambda.Runtime.NODEJS_18_X,
      handler: 'handler',
      role: attachmentLambdaRole,
      timeout: cdk.Duration.seconds(120),
      environment: {
        workloadRouteTableID: this.workloadRouteTable.ref,
        inspectionRouteTableID: this.inspectionRouteTable.ref,
      }
    });

    const eventPattern = {
      source: ['aws.ec2'],
      detail: {
        eventName: ['CreateTransitGatewayVpcAttachment'],
      },
    };

    new events.Rule(this, 'TGWAttachmentCreated', {
      eventPattern: eventPattern,
      targets: [new targets.LambdaFunction(tgwRouteLambda)],
    });
  }
}
