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

import { Environment } from 'aws-cdk-lib';

const HUB_ACCOUNT_ID = '';

// This check is here for sample code to give meaningful error when
// HUB_ACCOUNT_ID is not set
if (!HUB_ACCOUNT_ID) {
  throw new Error('HUB_ACCOUNT_ID not set in lib/configuration.ts');
}

export const HUB_ENV: Environment = {
  account: HUB_ACCOUNT_ID,
  region: 'us-west-2',
};

export const SPOKE_ENV: Environment = {
  account: HUB_ACCOUNT_ID,
  region: 'us-west-2',
};

export const ORGANISATION_CIDR = '10.0.0.0/8';
