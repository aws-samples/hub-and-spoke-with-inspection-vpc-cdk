import {
    EC2Client,
    DisassociateTransitGatewayRouteTableCommand,
    DescribeTransitGatewayAttachmentsCommand,
    AssociateTransitGatewayRouteTableCommand,
    EnableTransitGatewayRouteTablePropagationCommand
} from '@aws-sdk/client-ec2';
import { setTimeout } from 'timers/promises';

const ec2Client = new EC2Client({})

export const handler = async (event: any) => {
    let vpcType: string;
    let attachmentID: string;
    
    // verify that required parameters are set
    if (process.env.workloadRouteTableID == undefined || 
        process.env.inspectionRouteTableID == undefined) {
            throw new Error("route tables IDs are missing")
    }

    // validate that event has required parameters
    try {
    vpcType = event["detail"]["requestParameters"]
        ["CreateTransitGatewayVpcAttachmentRequest"]["TagSpecifications"]
        ["Tag"]["Value"];
    attachmentID = event["detail"]["responseElements"]
        ["CreateTransitGatewayVpcAttachmentResponse"]
        ["transitGatewayVpcAttachment"]["transitGatewayAttachmentId"]
    } catch {
        throw new Error("required attribute missing");
    };

    if (vpcType != "workload" && vpcType != "inspection") {
        throw new Error("vpc type is not workload or inspection")
    }

    let associationRouteTableID = ""
    if (vpcType == "workload") {
        associationRouteTableID = process.env.workloadRouteTableID
    } else associationRouteTableID = process.env.inspectionRouteTableID

    const describeAttachmentCommand = new DescribeTransitGatewayAttachmentsCommand({
        TransitGatewayAttachmentIds: [attachmentID]
    });
    const tgwAttachments = await ec2Client.send(describeAttachmentCommand);

    if (!tgwAttachments.TransitGatewayAttachments) {
        throw new Error("attachment not found in TGW");
    };

    // If the attachment is not yet ready. Wait for it to come available.
    if (tgwAttachments.TransitGatewayAttachments[0].State == 'pending') {
       await waitAvailable(attachmentID);
    };

    // Disassociate the attachment if already attached
    if (tgwAttachments.TransitGatewayAttachments[0].Association !== undefined) {
        const routeTableID = tgwAttachments.TransitGatewayAttachments[0].Association?.TransitGatewayRouteTableId

        const disassociateCommand = new DisassociateTransitGatewayRouteTableCommand({
            TransitGatewayAttachmentId: attachmentID,
            TransitGatewayRouteTableId: routeTableID,
        });

        let associated = true;
        while (associated) {
            // Disassociate attachment
            await ec2Client.send(disassociateCommand);

            // Wait for attachment to get disassociated
            await setTimeout(2000);

            let attachments = await ec2Client.send(describeAttachmentCommand);
            
            if (attachments.TransitGatewayAttachments) {
                if (attachments.TransitGatewayAttachments[0].Association == undefined) {
                    associated = false;
                };
            };
        };
    }

    //associate attachment with correct route table
    const attachCommand = new AssociateTransitGatewayRouteTableCommand({
        TransitGatewayAttachmentId: attachmentID,
        TransitGatewayRouteTableId: associationRouteTableID,
    });

    await ec2Client.send(attachCommand);

    if (vpcType == "workload") {
        await ec2Client.send(new EnableTransitGatewayRouteTablePropagationCommand({
            TransitGatewayRouteTableId: process.env.inspectionRouteTableID,
            TransitGatewayAttachmentId: attachmentID,
        }));
    };

}

const waitAvailable = async(attachmentID: string): Promise<void> => {
    const describeAttachmentCommand = new DescribeTransitGatewayAttachmentsCommand({
        TransitGatewayAttachmentIds: [attachmentID]
    });

    let pending = true;
    while (pending) {

        // Wait for attachment to get created
        await setTimeout(2000);

        const attachmentStatus = await ec2Client.send(describeAttachmentCommand);
            if (attachmentStatus.TransitGatewayAttachments) {
                if (attachmentStatus.TransitGatewayAttachments[0].State !== 'pending') {
                    pending = false;
                }
            }
    }
    return
}