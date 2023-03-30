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
import * as networkfirewall from 'aws-cdk-lib/aws-networkfirewall';

export class NetworkFirewallRules extends cdk.Stack {
  public readonly fwPolicy: networkfirewall.CfnFirewallPolicy;
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const fwAllowStatelessRuleGroup = new networkfirewall.CfnRuleGroup(
      this,
      'fwAllowStatelessRuleGroup',
      {
        capacity: 10,
        ruleGroupName: 'AllowStateless',
        type: 'STATELESS',
        ruleGroup: {
          rulesSource: {
            statelessRulesAndCustomActions: {
              statelessRules: [
                {
                  priority: 1,
                  ruleDefinition: {
                    actions: ['aws:pass'],
                    matchAttributes: {
                      protocols: [1],
                      sources: [
                        {
                          addressDefinition: '0.0.0.0/0',
                        },
                      ],
                      destinations: [
                        {
                          addressDefinition: '0.0.0.0/0',
                        },
                      ],
                    },
                  },
                },
              ],
            },
          },
        },
      }
    );

    const fwAllowRuleGroup = new networkfirewall.CfnRuleGroup(
      this,
      'fwAllowRuleGroup',
      {
        capacity: 10,
        ruleGroupName: 'AllowRules',
        type: 'STATEFUL',
        ruleGroup: {
          rulesSource: {
            statefulRules: [
              {
                action: 'PASS',
                header: {
                  destination: 'ANY',
                  destinationPort: '80',
                  source: '10.0.0.0/8',
                  sourcePort: 'ANY',
                  protocol: 'TCP',
                  direction: 'FORWARD',
                },
                ruleOptions: [
                  {
                    keyword: 'sid:1',
                  },
                ],
              },
              {
                action: 'PASS',
                header: {
                  destination: 'ANY',
                  destinationPort: '443',
                  source: '10.0.0.0/8',
                  sourcePort: 'ANY',
                  protocol: 'TCP',
                  direction: 'FORWARD',
                },
                ruleOptions: [
                  {
                    keyword: 'sid:2',
                  },
                ],
              },
              {
                action: 'PASS',
                header: {
                  destination: 'ANY',
                  destinationPort: '123',
                  source: '10.0.0.0/8',
                  sourcePort: 'ANY',
                  protocol: 'UDP',
                  direction: 'FORWARD',
                },
                ruleOptions: [
                  {
                    keyword: 'sid:3',
                  },
                ],
              },
            ],
          },
        },
      }
    );

    const fwDenyRuleGroup = new networkfirewall.CfnRuleGroup(
      this,
      'fwDenyRuleGroup',
      {
        capacity: 10,
        ruleGroupName: 'DenyAll',
        type: 'STATEFUL',
        description: 'Deny all other traffice',
        ruleGroup: {
          rulesSource: {
            statefulRules: [
              {
                action: 'DROP',
                header: {
                  destination: 'ANY',
                  destinationPort: 'ANY',
                  source: 'ANY',
                  sourcePort: 'ANY',
                  protocol: 'IP',
                  direction: 'FORWARD',
                },
                ruleOptions: [
                  {
                    keyword: 'sid:100',
                  },
                ],
              },
            ],
          },
        },
      }
    );

    const fwPolicy = new networkfirewall.CfnFirewallPolicy(this, 'FwPolicy', {
      firewallPolicy: {
        statelessDefaultActions: ['aws:forward_to_sfe'],
        statelessFragmentDefaultActions: ['aws:forward_to_sfe'],
        statelessRuleGroupReferences: [
          {
            priority: 1,
            resourceArn: fwAllowStatelessRuleGroup.attrRuleGroupArn,
          },
        ],
        statefulRuleGroupReferences: [
          {
            resourceArn: fwAllowRuleGroup.attrRuleGroupArn,
          },
          {
            resourceArn: fwDenyRuleGroup.attrRuleGroupArn,
          },
        ],
      },
      firewallPolicyName: 'SamplePolicy',
    });

    this.fwPolicy = fwPolicy;
  }
}
