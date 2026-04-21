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

import { Input } from 'antd';

import { Block } from '@/components/block';
import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';

export const ClaudeConfig: IPluginConfig = {
  plugin: 'claude',
  name: 'Claude',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 6.6,
  isBeta: true,
  connection: {
    docLink: 'https://github.com/apache/incubator-devlake/blob/main/backend/plugins/claude/README.md',
    initialValues: {
      endpoint: 'https://api.anthropic.com/v1',
      organizationId: '',
      token: '',
      rateLimitPerHour: 1000,
    },
    fields: [
      'name',
      'endpoint',
      ({ values, setValues }: any) => (
        <Block key="organizationId" title="Organization ID">
          <Input
            placeholder="org-xxxxxxxxxxxxxxxx"
            value={values.organizationId ?? ''}
            onChange={(e) => setValues({ organizationId: e.target.value })}
          />
          <p style={{ margin: '4px 0 0', color: '#7a7a7a', fontSize: 12 }}>
            Optional. Only used as a label — the Admin API key already scopes requests to your organization.
          </p>
        </Block>
      ),
      {
        key: 'token',
        label: 'Admin API Key',
        subLabel: (
          <>
            Requires an <strong>Admin API key</strong> (<code>sk-ant-admin-...</code>) from{' '}
            <a href="https://platform.claude.com/settings/admin-keys" target="_blank" rel="noreferrer">
              Console → Admin API Keys
            </a>
            . Only available for <strong>organization accounts</strong> (not individual accounts). Standard user API
            keys (<code>sk-ant-api03-...</code>) cannot access the Claude Code Analytics API.
          </>
        ),
      },
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses 1,000 requests/hour for Claude data collection. Adjust this value to throttle collection speed.',
        defaultValue: 1000,
      },
    ],
  },
  dataScope: {
    title: 'Organizations',
  },
  scopeConfig: {
    entities: ['CROSS'],
    transformation: {},
  },
};
