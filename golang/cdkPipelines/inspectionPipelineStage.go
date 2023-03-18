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
	"github.com/aws/constructs-go/constructs/v10"
)

type NetworkWorkshopInspectStageProps struct {
	awscdk.StageProps
}

func NetworkWorkshopInspectStage(scope constructs.Construct, id string, props *NetworkWorkshopInspectStageProps) awscdk.Stage {
	var sprops awscdk.StageProps
	if props != nil {
		sprops = props.StageProps
	}

	stage := awscdk.NewStage(scope, &id, &sprops)

	tgw := InspectionTgwStack(stage, "TransitGateway", nil)

	NetworkFirewallStack(stage, "Inspection", &NetworkFirewallStackProps{
		cidr:        "10.100.0.0/16",
		orgCidr:     OrganizationCidr,
		transitGWId: tgw.tgWId,
	})

	InspectionWorkloadStack(stage, "Workload1", &InspectionWorkloadStackProps{
		cidr:        "10.110.0.0/16",
		transitGWId: tgw.tgWId,
	})

	InspectionWorkloadStack(stage, "Workload2", &InspectionWorkloadStackProps{
		cidr:        "10.111.0.0/16",
		transitGWId: tgw.tgWId,
	})

	return stage
}
