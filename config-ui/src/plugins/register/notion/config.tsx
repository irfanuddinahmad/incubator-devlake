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

import { CopyOutlined } from '@ant-design/icons';
import { Button, Checkbox, Input, message } from 'antd';

import { Block } from '@/components/block';
import { IPluginConfig } from '@/types';

import Icon from '@/images/plugin-icon.svg?react';

export const NotionConfig: IPluginConfig = {
  plugin: 'notion',
  name: 'Notion',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 7.0,
  isBeta: true,
  connection: {
    docLink: 'https://developers.notion.com/reference/webhooks',
    initialValues: {
      endpoint: 'https://api.notion.com',
      token: '',
      workspaceId: '',
      apiVersion: '2026-03-11',
      enableWebhook: false,
      webhookSharedKey: '',
      rateLimitPerHour: 10800,
    },
    fields: [
      'name',
      'endpoint',
      {
        key: 'token',
        label: 'Internal Integration Token',
      },
      ({ values, setValues }: any) => (
        <Block key="workspaceId" title="Workspace ID">
          <Input
            placeholder="Notion workspace ID"
            value={values.workspaceId ?? ''}
            onChange={(e) => setValues({ workspaceId: e.target.value })}
          />
        </Block>
      ),
      ({ values, setValues }: any) => (
        <Block key="apiVersion" title="Notion API Version">
          <Input
            placeholder="2026-03-11"
            value={values.apiVersion ?? ''}
            onChange={(e) => setValues({ apiVersion: e.target.value })}
          />
        </Block>
      ),
      ({ values, setValues, setErrors }: any) => (
        <Block key="enableWebhook" title="Webhook Ingestion">
          <Checkbox
            checked={!!values.enableWebhook}
            onChange={(e) => {
              const enabled = e.target.checked;
              setValues({ enableWebhook: enabled });
              setErrors({ webhookSharedKey: enabled && !(values.webhookSharedKey ?? '').trim() ? 'required' : '' });
            }}
          >
            Enable webhook ingestion in addition to polling
          </Checkbox>
        </Block>
      ),
      ({ values, errors, setValues, setErrors }: any) => (
        <Block
          key="webhookSharedKey"
          title="Webhook Shared Key"
          description="Use Notion webhook verification_token for HMAC signature validation."
        >
          <Input.Password
            placeholder="Notion verification token"
            value={values.webhookSharedKey ?? ''}
            status={values.enableWebhook && errors.webhookSharedKey ? 'error' : undefined}
            onChange={(e) => {
              setValues({ webhookSharedKey: e.target.value });
              setErrors({ webhookSharedKey: values.enableWebhook && !e.target.value.trim() ? 'required' : '' });
            }}
          />
          {values.enableWebhook && errors.webhookSharedKey && (
            <p style={{ color: '#ff4d4f', marginTop: 8 }}>Webhook shared key is required when webhook is enabled.</p>
          )}
          {!values.enableWebhook && (
            <p style={{ marginTop: 8 }}>Turn on webhook ingestion to enforce webhook shared key validation.</p>
          )}
        </Block>
      ),
      ({ values }: any) => {
        const webhookUrl = `${window.location.origin}/plugins/notion/connections/{connectionId}/scopes/{scopeId}/webhook`;
        return (
          <Block
            key="webhookUrl"
            title="Webhook URL Template"
            description="Replace {connectionId} and {scopeId} after creating connection and scope."
          >
            <Input
              readOnly
              value={webhookUrl}
              disabled={!values.enableWebhook}
              addonAfter={
                <Button
                  type="text"
                  icon={<CopyOutlined />}
                  disabled={!values.enableWebhook}
                  onClick={() => {
                    void navigator.clipboard.writeText(webhookUrl);
                    void message.success('Webhook URL copied.');
                  }}
                />
              }
            />
          </Block>
        );
      },
      'proxy',
      {
        key: 'rateLimitPerHour',
        defaultValue: 10800,
      },
    ],
  },
  dataScope: {
    title: 'Scopes',
  },
  scopeConfig: {
    entities: ['CROSS'],
    transformation: {},
  },
};
