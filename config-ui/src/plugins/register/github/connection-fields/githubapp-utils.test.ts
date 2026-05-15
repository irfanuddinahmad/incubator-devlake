/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { equal, deepEqual } from 'node:assert/strict';
import { test } from 'node:test';

import {
  buildGithubInstallationOptions,
  invalidateGithubAppConfig,
  isMaskedGithubAppSecret,
  shouldValidateGithubAppConfig,
} from './githubapp-utils';

test('detects masked GitHub App private keys returned by the API', () => {
  equal(
    isMaskedGithubAppSecret('-----BEGIN RSA PRIVATE KEY-----\nMIIEpA********END\n-----END RSA PRIVATE KEY-----'),
    true,
  );
  equal(isMaskedGithubAppSecret('-----BEGIN RSA PRIVATE KEY-----\nMIIEpAABCDEF\n-----END RSA PRIVATE KEY-----'), false);
});

test('does not validate saved masked GitHub App private keys', () => {
  equal(
    shouldValidateGithubAppConfig(
      'https://api.github.com/',
      '12345',
      '-----BEGIN RSA PRIVATE KEY-----\nMIIEpA********END\n-----END RSA PRIVATE KEY-----',
    ),
    false,
  );
  equal(
    shouldValidateGithubAppConfig(
      'https://api.github.com/',
      '12345',
      '-----BEGIN RSA PRIVATE KEY-----\nMIIEpAABCDEF\n-----END RSA PRIVATE KEY-----',
    ),
    true,
  );
});

test('keeps saved installation visible when installations cannot be reloaded', () => {
  deepEqual(buildGithubInstallationOptions(undefined, 98765), [{ value: 98765, label: 'Saved installation (98765)' }]);
});

test('does not duplicate saved installation option when GitHub returns it', () => {
  deepEqual(
    buildGithubInstallationOptions(
      [
        {
          id: 98765,
          account: {
            login: 'apache',
          },
        },
      ],
      98765,
    ),
    [{ value: 98765, label: 'apache' }],
  );
});

test('changing GitHub App ID clears stale validation and returns to untested state', () => {
  deepEqual(
    invalidateGithubAppConfig(
      {
        appId: '12345',
        secretKey: 'private-key',
        installationId: 98765,
        status: 'valid',
        from: 'old-app',
        installations: [
          {
            id: 98765,
            account: {
              login: 'apache',
            },
          },
        ],
      },
      { appId: '67890' },
    ),
    {
      appId: '67890',
      secretKey: 'private-key',
      installationId: undefined,
      status: 'idle',
      from: undefined,
      installations: undefined,
    },
  );
});

test('changing GitHub App private key clears stale validation and returns to untested state', () => {
  deepEqual(
    invalidateGithubAppConfig(
      {
        appId: '12345',
        secretKey: 'old-private-key',
        installationId: 98765,
        status: 'valid',
        from: 'old-app',
        installations: [
          {
            id: 98765,
            account: {
              login: 'apache',
            },
          },
        ],
      },
      { secretKey: 'new-private-key' },
    ),
    {
      appId: '12345',
      secretKey: 'new-private-key',
      installationId: undefined,
      status: 'idle',
      from: undefined,
      installations: undefined,
    },
  );
});

test('clearing a required GitHub App credential resets status to idle', () => {
  deepEqual(
    invalidateGithubAppConfig(
      {
        appId: '12345',
        secretKey: 'private-key',
        installationId: 98765,
        status: 'valid',
        from: 'old-app',
      },
      { secretKey: '' },
    ),
    {
      appId: '12345',
      secretKey: '',
      installationId: undefined,
      status: 'idle',
      from: undefined,
      installations: undefined,
    },
  );
});
