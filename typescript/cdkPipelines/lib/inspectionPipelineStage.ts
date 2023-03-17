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
import { Construct } from 'constructs';
import { TransitGatewayStack } from '../lib/inspectionTransitGateway';
import { WorkloadStack } from '../lib/inspectionVpcWorkloackStack';
import { ORGANISATION_CIDR } from './configurations';
import { NetworkFirewallStack } from './inspectionStack';

export class NetworkWorkshopInspectionStage extends cdk.Stage {
  constructor(scope: Construct, id: string, props?: cdk.StageProps) {
    super(scope, id, props);

    const tgw = new TransitGatewayStack(this, 'TransitGateway');

    new NetworkFirewallStack(this, 'Inspection', {
      cidr: '10.100.0.0/16',
      orgCidr: ORGANISATION_CIDR,
      transitGWId: tgw.transitGateway.attrId,
      workloadRouteTableID: tgw.workloadRouteTable.ref,
    });

    new WorkloadStack(this, 'Workload', {
      transitGWId: tgw.transitGateway.attrId,
      cidr: '10.110.0.0/16',
    });

    new WorkloadStack(this, 'Workload2', {
      transitGWId: tgw.transitGateway.attrId,
      cidr: '10.111.0.0/16',
    });
  }
}
