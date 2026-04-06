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

export const CodexConfig: IPluginConfig = {
  plugin: 'codex',
  name: 'OpenAI Codex',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 6.8,
  isBeta: true,
  connection: {
    docLink: 'https://platform.openai.com/docs/api-reference',
    initialValues: {
      endpoint: 'https://api.openai.com/v1',
      token: '',
      projectId: '',
      rateLimitPerHour: 1000,
    },
    fields: [
      'name',
      'endpoint',
      ({ values, setValues }: any) => (
        <Block key="projectId" title="Project ID">
          <Input
            placeholder="Your Codex project ID (optional)"
            value={values.projectId ?? ''}
            onChange={(e) => setValues({ projectId: e.target.value })}
          />
          <p style={{ margin: '4px 0 0', color: '#7a7a7a', fontSize: 12 }}>
            Optional. Used to pre-populate the data scope.
          </p>
        </Block>
      ),
      {
        key: 'token',
        label: 'API Key',
        subLabel: (
          <>
            Enter your OpenAI API key from{' '}
            <a href="https://platform.openai.com/api-keys" target="_blank" rel="noreferrer">
              OpenAI Platform
            </a>
            .
          </>
        ),
      },
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses 1,000 requests/hour for Codex data collection. Adjust this value to throttle collection speed.',
        defaultValue: 1000,
      },
    ],
  },
  dataScope: {
    title: 'Projects',
  },
  scopeConfig: {
    entities: ['CROSS'],
    transformation: {},
  },
};
