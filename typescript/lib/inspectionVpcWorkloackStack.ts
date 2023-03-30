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
import { Construct } from 'constructs';

interface WorkloadProps extends cdk.StackProps {
  transitGWId: string,
  cidr: string,
}

export class WorkloadStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: WorkloadProps) {
    super(scope, id, props);

    const vpc = new ec2.Vpc(this, 'VPC', {
      maxAzs: 2,
      ipAddresses: ec2.IpAddresses.cidr(props.cidr),
      enableDnsSupport: true,
      enableDnsHostnames: true,
      subnetConfiguration: [
        {
          name: 'Private',
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
          cidrMask: 24,
        },
      ],
    });

    const privateSubnets = vpc.selectSubnets({
      subnetGroupName: 'Private',
    });

    const tgwAttachment = new ec2.CfnTransitGatewayAttachment(
      this,
      'TGWAttachment',
      {
        transitGatewayId: props.transitGWId,
        subnetIds: privateSubnets.subnetIds,
        vpcId: vpc.vpcId,
        tags: [
          {
            key: 'routeTable',
            value: 'workload',
          },
        ],
      }
    );

    // Create a default route for each TGw subnet
    for (const subnet of privateSubnets.subnets) {
      const subnetName = subnet.node.path.split('/').pop()!;
      new ec2.CfnRoute(this, `DefaultRoute-${subnetName}`, {
        destinationCidrBlock: '0.0.0.0/0',
        transitGatewayId: props.transitGWId,
        routeTableId: subnet.routeTable.routeTableId,
      }).node.addDependency(tgwAttachment);
    }

    vpc.addInterfaceEndpoint('SsmEndpoint', {
      service: ec2.InterfaceVpcEndpointAwsService.SSM,
    });

    vpc.addInterfaceEndpoint('SsmMessagesEndpoint', {
      service: ec2.InterfaceVpcEndpointAwsService.SSM_MESSAGES,
    });

    vpc.addInterfaceEndpoint('Ec2MessagesEndpoint', {
      service: ec2.InterfaceVpcEndpointAwsService.EC2_MESSAGES,
    });

    const ssmRole = new iam.Role(this, 'SSMRole', {
      assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          'AmazonSSMManagedInstanceCore'
        ),
      ],
    });

    const securityGroup = new ec2.SecurityGroup(this, 'WorkloadEC2SG', {
      securityGroupName: 'workload-sg',
      vpc: vpc,
    });

    securityGroup.addIngressRule(ec2.Peer.anyIpv4(), ec2.Port.allIcmp());

    new ec2.Instance(this, 'WorkloadEC2', {
      vpc: vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      instanceType: ec2.InstanceType.of(
        ec2.InstanceClass.BURSTABLE4_GRAVITON,
        ec2.InstanceSize.MICRO
      ),
      machineImage: ec2.MachineImage.latestAmazonLinux({
        cpuType: ec2.AmazonLinuxCpuType.ARM_64,
        generation: ec2.AmazonLinuxGeneration.AMAZON_LINUX_2,
      }),
      role: ssmRole,
      securityGroup: securityGroup,
    });
  }
}
