import {
    NetworkFirewallClient,
    DescribeFirewallCommand,
    DescribeFirewallCommandInput,
  } from '@aws-sdk/client-network-firewall';
  import { CloudFormationCustomResourceEvent } from 'aws-lambda';
  import { v4 as uuidv4 } from 'uuid';
  
  const anfClient = new NetworkFirewallClient({});
  
  export const handler = async (event: CloudFormationCustomResourceEvent) => {
    console.log(event);
    let responseData = {
      PhysicalResourceId: '',
      Data: {},
    };
  
    if ( 'PhysicalResourceId' in event) {
      responseData.PhysicalResourceId = event.PhysicalResourceId;
    } else {
      responseData.PhysicalResourceId = uuidv4();
    }
  
    if ( event.ResourceProperties.AvailabilityZone || event.ResourceProperties.FirewallName) {
  
      const availabilityZone = event.ResourceProperties.AvailabilityZone;
      const firewallName = event.ResourceProperties.FirewallName;
  
      try {
        const params: DescribeFirewallCommandInput = {
          FirewallName: firewallName,
        };
  
        const response = await anfClient.send(new DescribeFirewallCommand(params));
        console.log(response);
        if (response.FirewallStatus && response.FirewallStatus.SyncStates) {
          if (response.FirewallStatus.SyncStates[availabilityZone]) {
            if (response.FirewallStatus.SyncStates[availabilityZone].Attachment) {
              if (response.FirewallStatus.SyncStates[availabilityZone].Attachment?.EndpointId) {
                console.log(response.FirewallStatus.SyncStates[availabilityZone].Attachment?.EndpointId);
                responseData.Data = { EndpointId: response.FirewallStatus.SyncStates[availabilityZone].Attachment?.EndpointId };
              }
            }
          }
        }
  
      } catch (e) {
        console.log(e);
        throw new Error(String(e));
      }
    }
  
    return responseData;
  };
  
  