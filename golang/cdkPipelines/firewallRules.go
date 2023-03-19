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

package cdkPipelines

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	firewall "github.com/aws/aws-cdk-go/awscdk/v2/awsnetworkfirewall"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type FirewallRuleStackProps struct {
	awscdk.StackProps
}

type FirewallRulesStackOutputs struct {
	awscdk.Stack
	fwPolicyArn *string
}

func NetworkFirewallRules(scope constructs.Construct, id string, props *FirewallRuleStackProps) FirewallRulesStackOutputs {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	fwAllowStatelessRuleGroup := firewall.NewCfnRuleGroup(scope, jsii.String("fwAllowStatelessRuleGroup"), &firewall.CfnRuleGroupProps{
		Capacity:      jsii.Number(10),
		RuleGroupName: jsii.String("AllowStateless"),
		Type:          jsii.String("STATELESS"),
		RuleGroup: firewall.CfnRuleGroup_RuleGroupProperty{
			RulesSource: firewall.CfnRuleGroup_RulesSourceProperty{
				StatelessRulesAndCustomActions: firewall.CfnRuleGroup_StatelessRulesAndCustomActionsProperty{
					StatelessRules: []interface{}{
						&firewall.CfnRuleGroup_StatelessRuleProperty{
							Priority: jsii.Number(1),
							RuleDefinition: &firewall.CfnRuleGroup_RuleDefinitionProperty{
								Actions: jsii.Strings("aws:pass"),

								MatchAttributes: &firewall.CfnRuleGroup_MatchAttributesProperty{
									Protocols: []interface{}{
										jsii.Number(1),
									},
									Sources: []interface{}{
										&firewall.CfnRuleGroup_AddressProperty{
											AddressDefinition: jsii.String("0.0.0.0/0"),
										},
									},
									Destinations: []interface{}{
										&firewall.CfnRuleGroup_AddressProperty{
											AddressDefinition: jsii.String("0.0.0.0/0"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	fwAllowRuleGroup := firewall.NewCfnRuleGroup(scope, jsii.String("fwAllowRuleGroup"), &firewall.CfnRuleGroupProps{
		Capacity:      jsii.Number(10),
		RuleGroupName: jsii.String("AllowRules"),
		Type:          jsii.String("STATEFUL"),
		Description:   jsii.String("Allow traffic to Internet"),
		RuleGroup: firewall.CfnRuleGroup_RuleGroupProperty{
			RulesSource: firewall.CfnRuleGroup_RulesSourceProperty{
				StatefulRules: []interface{}{
					&firewall.CfnRuleGroup_StatefulRuleProperty{
						Action: jsii.String("PASS"),
						Header: &firewall.CfnRuleGroup_HeaderProperty{
							Destination:     jsii.String("ANY"),
							DestinationPort: jsii.String("80"),
							Source:          jsii.String("10.0.0.0/8"),
							SourcePort:      jsii.String("ANY"),
							Protocol:        jsii.String("TCP"),
							Direction:       jsii.String("FORWARD"),
						},
						RuleOptions: []interface{}{
							&firewall.CfnRuleGroup_RuleOptionProperty{
								Keyword: jsii.String("sid:1"),
							},
						},
					},
					&firewall.CfnRuleGroup_StatefulRuleProperty{
						Action: jsii.String("PASS"),
						Header: &firewall.CfnRuleGroup_HeaderProperty{
							Destination:     jsii.String("ANY"),
							DestinationPort: jsii.String("443"),
							Source:          jsii.String("10.0.0.0/8"),
							SourcePort:      jsii.String("ANY"),
							Protocol:        jsii.String("TCP"),
							Direction:       jsii.String("FORWARD"),
						},
						RuleOptions: []interface{}{
							&firewall.CfnRuleGroup_RuleOptionProperty{
								Keyword: jsii.String("sid:2"),
							},
						},
					},
					&firewall.CfnRuleGroup_StatefulRuleProperty{
						Action: jsii.String("PASS"),
						Header: &firewall.CfnRuleGroup_HeaderProperty{
							Destination:     jsii.String("ANY"),
							DestinationPort: jsii.String("123"),
							Source:          jsii.String("10.0.0.0/8"),
							SourcePort:      jsii.String("ANY"),
							Protocol:        jsii.String("UDP"),
							Direction:       jsii.String("FORWARD"),
						},
						RuleOptions: []interface{}{
							&firewall.CfnRuleGroup_RuleOptionProperty{
								Keyword: jsii.String("sid:3"),
							},
						},
					},
				},
			},
		},
	})

	fwDenyRuleGroup := firewall.NewCfnRuleGroup(scope, jsii.String("fwDenyRuleGroup"), &firewall.CfnRuleGroupProps{
		Capacity:      jsii.Number(10),
		RuleGroupName: jsii.String("DenyAll"),
		Type:          jsii.String("STATEFUL"),
		Description:   jsii.String("Deny all other traffic"),
		RuleGroup: firewall.CfnRuleGroup_RuleGroupProperty{
			RulesSource: firewall.CfnRuleGroup_RulesSourceProperty{
				StatefulRules: []interface{}{
					&firewall.CfnRuleGroup_StatefulRuleProperty{
						Action: jsii.String("DROP"),
						Header: &firewall.CfnRuleGroup_HeaderProperty{
							Destination:     jsii.String("ANY"),
							DestinationPort: jsii.String("ANY"),
							Source:          jsii.String("ANY"),
							SourcePort:      jsii.String("ANY"),
							Protocol:        jsii.String("IP"),
							Direction:       jsii.String("FORWARD"),
						},
						RuleOptions: []interface{}{
							&firewall.CfnRuleGroup_RuleOptionProperty{
								Keyword: jsii.String("sid:100"),
							},
						},
					},
				},
			},
		},
	})

	fwPolicy := firewall.NewCfnFirewallPolicy(scope, jsii.String("FwPolicy"), &firewall.CfnFirewallPolicyProps{
		FirewallPolicy: &firewall.CfnFirewallPolicy_FirewallPolicyProperty{
			StatelessDefaultActions:         jsii.Strings("aws:forward_to_sfe"),
			StatelessFragmentDefaultActions: jsii.Strings("aws:forward_to_sfe"),
			StatelessRuleGroupReferences: []interface{}{
				&firewall.CfnFirewallPolicy_StatelessRuleGroupReferenceProperty{
					Priority:    jsii.Number(1),
					ResourceArn: fwAllowStatelessRuleGroup.AttrRuleGroupArn(),
				},
			},
			StatefulRuleGroupReferences: []interface{}{
				&firewall.CfnFirewallPolicy_StatefulRuleGroupReferenceProperty{
					ResourceArn: fwAllowRuleGroup.AttrRuleGroupArn(),
				},
				&firewall.CfnFirewallPolicy_StatefulRuleGroupReferenceProperty{
					ResourceArn: fwDenyRuleGroup.AttrRuleGroupArn(),
				},
			},
		},
		FirewallPolicyName: jsii.String("SamplePolicy"),
	})

	var outputs FirewallRulesStackOutputs
	outputs.Stack = stack
	outputs.fwPolicyArn = fwPolicy.AttrFirewallPolicyArn()

	return outputs
}
