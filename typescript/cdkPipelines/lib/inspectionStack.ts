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
import { Construct } from 'constructs';
import { CustomResource, Duration, RemovalPolicy } from 'aws-cdk-lib';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as nodejs from 'aws-cdk-lib/aws-lambda-nodejs';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as nf from 'aws-cdk-lib/aws-networkfirewall';
import * as cr from 'aws-cdk-lib/custom-resources';
import { NetworkFirewallRules } from '../lib/firewallRules';
import { ORGANISATION_CIDR } from '../lib/configurations';

interface NetworkFirewallProps extends cdk.StackProps {
  cidr: string;
  orgCidr: string;
  transitGWId: string;
  workloadRouteTableID: string;
}

export class NetworkFirewallStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: NetworkFirewallProps) {
    super(scope, id, props);

    const vpc = new ec2.Vpc(this, 'InspectionVPC', {
      ipAddresses: ec2.IpAddresses.cidr(props.cidr),
      subnetConfiguration: [
        {
          name: 'TGWSubnets',
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
          cidrMask: 26,
        },
        {
          name: 'FirewallSubnets',
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
          cidrMask: 27,
        },
        {
          name: 'Public',
          subnetType: ec2.SubnetType.PUBLIC,
          cidrMask: 27,
        },
      ],
    });

    const tGWSubnets = vpc.selectSubnets({
      subnetGroupName: 'TGWSubnets',
    });

    const pubSubnets = vpc.selectSubnets({
      subnetGroupName: 'Public',
    });

    const tGWAttachment = new ec2.CfnTransitGatewayAttachment(
      this,
      'TGWAttachment',
      {
        transitGatewayId: props.transitGWId,
        subnetIds: tGWSubnets.subnetIds,
        vpcId: vpc.vpcId,
        options: {
          applianceModeSupport: 'ENABLED',
        },
        tags: [
          {
            key: 'routeTable',
            value: 'inspection',
          },
        ],
      }
    );

    new ec2.CfnTransitGatewayRoute(this, 'TGWRoute', {
      destinationCidrBlock: '0.0.0.0/0',
      transitGatewayAttachmentId: tGWAttachment.attrId,
      transitGatewayRouteTableId: props.workloadRouteTableID,
    });

    const firewallRules = new NetworkFirewallRules(
      this,
      'NetworkFirewallRules'
    );

    const fwSubnets = vpc.selectSubnets({
      subnetGroupName: 'FirewallSubnets',
    });

    let subnetList: nf.CfnFirewall.SubnetMappingProperty[] = [];
    for (const subnet of fwSubnets.subnets) {
      const subnetMappingProperty: nf.CfnFirewall.SubnetMappingProperty = {
        subnetId: subnet.subnetId,
      };
      subnetList.push(subnetMappingProperty);
    }

    const networkFw = new nf.CfnFirewall(this, 'NetworkFirewall', {
      firewallName: 'EgressInspectionFirewall',
      firewallPolicyArn: firewallRules.fwPolicy.attrFirewallPolicyArn,
      subnetMappings: subnetList,
      vpcId: vpc.vpcId,
    });

    const fwFlowLogsGroup = new logs.LogGroup(this, 'FwFlowLogsGroup', {
      logGroupName: 'NetworkFirewallFlowLogs',
      removalPolicy: RemovalPolicy.DESTROY,
    });

    const fwAlertLogGroup = new logs.LogGroup(this, 'FWAlertLogsGroup', {
      logGroupName: 'NetworkFirewallAlertLogs',
      removalPolicy: RemovalPolicy.DESTROY,
    });

    new nf.CfnLoggingConfiguration(this, 'FirewallLoggingConfg', {
      firewallArn: networkFw.ref,
      loggingConfiguration: {
        logDestinationConfigs: [
          {
            logDestination: {
              logGroup: fwFlowLogsGroup.logGroupName,
            },
            logDestinationType: 'CloudWatchLogs',
            logType: 'FLOW',
          },
          {
            logDestination: {
              logGroup: fwAlertLogGroup.logGroupName,
            },
            logDestinationType: 'CloudWatchLogs',
            logType: 'ALERT',
          },
        ],
      },
    });

    const routeLambdaRole = new iam.Role(this, 'RouteLambdaRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          'service-role/AWSLambdaBasicExecutionRole'
        ),
      ],
    });

    routeLambdaRole.addToPolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        actions: ['network-firewall:DescribeFirewall'],
        resources: [networkFw.attrFirewallArn],
      })
    );

    const customRouteLambda = new nodejs.NodejsFunction(this, 'endpoint', {
      functionName: 'CustomRouteLambda',
      role: routeLambdaRole,
      timeout: Duration.seconds(30),
    });

    const provider = new cr.Provider(this, 'Provider', {
      onEventHandler: customRouteLambda,
      logRetention: logs.RetentionDays.ONE_DAY,
    });

    vpc
      .selectSubnets({ subnetGroupName: 'Public' })
      .subnets.forEach((subnet) => {
        const subnetName = subnet.node.path.split('/').pop(); // E.g. TransitGatewayStack/InspectionVPC/PublicSubnet1

        // Custom resource returns AWS Network Firewall endpoint ID in correct availability zone.
        const endpoint = new CustomResource(
          this,
          `AnfEndpointFor-${subnetName}`,
          {
            serviceToken: provider.serviceToken,
            properties: {
              FirewallName: networkFw.firewallName,
              AvailabilityZone: subnet.availabilityZone,
            },
          }
        );

        // Create default route towards firewall endpoint from Public subnets.
        new ec2.CfnRoute(this, `${subnetName}AnfRoute`, {
          destinationCidrBlock: ORGANISATION_CIDR,
          routeTableId: subnet.routeTable.routeTableId,
          vpcEndpointId: endpoint.getAttString('EndpointId'),
        });
      });

    vpc
      .selectSubnets({ subnetGroupName: 'TGWSubnets' })
      .subnets.forEach((subnet) => {
        const subnetName = subnet.node.path.split('/').pop(); // E.g. TransitGatewayStack/InspectionVPC/PublicSubnet1

        // Custom resource returns AWS Network Firewall endpoint ID in correct availability zone.
        const endpoint = new CustomResource(
          this,
          `AnfEndpointFor-${subnetName}`,
          {
            serviceToken: provider.serviceToken,
            properties: {
              FirewallName: networkFw.firewallName,
              AvailabilityZone: subnet.availabilityZone,
            },
          }
        );

        // Create default route towards firewall endpoint from TGW subnets.
        new ec2.CfnRoute(this, `${subnetName}AnfRoute`, {
          destinationCidrBlock: '0.0.0.0/0',
          routeTableId: subnet.routeTable.routeTableId,
          vpcEndpointId: endpoint.getAttString('EndpointId'),
        });
      });

    vpc
      .selectSubnets({ subnetGroupName: 'FirewallSubnets' })
      .subnets.forEach((subnet) => {
        const subnetName = subnet.node.path.split('/').pop(); // E.g. TransitGatewayStack/InspectionVPC/PublicSubnet1

        // Create route towards organisation network from firewall subnets.
        new ec2.CfnRoute(this, `${subnetName}AnfRoute`, {
          destinationCidrBlock: ORGANISATION_CIDR,
          routeTableId: subnet.routeTable.routeTableId,
          transitGatewayId: props.transitGWId,
        });
      });
  }
}
