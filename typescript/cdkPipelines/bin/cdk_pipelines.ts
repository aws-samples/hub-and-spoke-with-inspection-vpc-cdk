#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import 'source-map-support/register';
import { HUB_ENV } from '../lib/configurations';
import { NetworkPipelineStack } from '../lib/pipelineModuleStack';

const app = new cdk.App();
new NetworkPipelineStack(app, 'NetworkPipeline', {
  env: HUB_ENV,
});
