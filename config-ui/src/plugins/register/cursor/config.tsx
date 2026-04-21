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

export const CursorConfig: IPluginConfig = {
  plugin: 'cursor',
  name: 'Cursor',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 6.7,
  isBeta: true,
  connection: {
    docLink: 'https://github.com/apache/incubator-devlake/blob/main/backend/plugins/cursor/README.md',
    initialValues: {
      endpoint: 'https://api.cursor.com',
      token: '',
      rateLimitPerHour: 1000,
    },
    fields: [
      'name',
      'endpoint',
      {
        key: 'token',
        label: 'API Key',
        subLabel: (
          <>
            Generate an API key from{' '}
            <a href="https://www.cursor.com/settings" target="_blank" rel="noreferrer">
              Cursor → Team Settings → Advanced
            </a>
            .
          </>
        ),
      },
      ({ values, setValues }: any) => (
        <Block key="teamId" title="Team ID">
          <Input
            placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            value={values.teamId ?? ''}
            onChange={(e) => setValues({ teamId: e.target.value })}
          />
          <p style={{ margin: '4px 0 0', color: '#7a7a7a', fontSize: 12 }}>
            Optional. Your Cursor team ID from{' '}
            <a href="https://www.cursor.com/settings" target="_blank" rel="noreferrer">
              Team Settings
            </a>
            . Used to pre-populate the data scope.
          </p>
        </Block>
      ),
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses 1,000 requests/hour for Cursor data collection. Adjust this value to throttle collection speed.',
        defaultValue: 1000,
      },
    ],
  },
  dataScope: {
    title: 'Teams',
  },
  scopeConfig: {
    entities: ['CROSS'],
    transformation: {},
  },
};
