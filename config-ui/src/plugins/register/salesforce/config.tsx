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

import { Input, Select } from 'antd';

import { Block } from '@/components/block';
import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';

const AUTH_MODE_OPTIONS = [
  { label: 'Access Token', value: 'access_token' },
  { label: 'Refresh Token', value: 'refresh_token' },
];

const validateConnection = (values: Record<string, any>) => {
  const authMode = values.authMode ?? 'access_token';

  return {
    instanceUrl: authMode === 'access_token' && !(values.instanceUrl ?? '').trim() ? 'required' : '',
    accessToken: authMode === 'access_token' && !(values.accessToken ?? '').trim() ? 'required' : '',
    refreshToken: authMode === 'refresh_token' && !(values.refreshToken ?? '').trim() ? 'required' : '',
    clientId: authMode === 'refresh_token' && !(values.clientId ?? '').trim() ? 'required' : '',
    clientSecret: authMode === 'refresh_token' && !(values.clientSecret ?? '').trim() ? 'required' : '',
  };
};

export const SalesforceConfig: IPluginConfig = {
  plugin: 'salesforce',
  name: 'Salesforce',
  icon: () => <Icon />,
  sort: 7.1,
  isBeta: true,
  connection: {
    docLink: 'https://developer.salesforce.com/docs/apis',
    initialValues: {
      authMode: 'access_token',
      loginUrl: 'https://login.salesforce.com',
      instanceUrl: '',
      accessToken: '',
      refreshToken: '',
      clientId: '',
      clientSecret: '',
      apiVersion: 'v61.0',
      rateLimitPerHour: 5000,
    },
    fields: [
      'name',
      ({ values, setValues, setErrors }: any) => (
        <Block key="authMode" title="Authentication Mode" required>
          <Select
            style={{ width: '100%' }}
            value={values.authMode ?? 'access_token'}
            options={AUTH_MODE_OPTIONS}
            onChange={(value) => {
              const nextValues = { ...values, authMode: value };
              setValues({ authMode: value });
              setErrors(validateConnection(nextValues));
            }}
          />
        </Block>
      ),
      ({ values, errors, setValues, setErrors }: any) =>
        (values.authMode ?? 'access_token') === 'access_token' ? (
          <Block key="instanceUrl" title="Instance URL" required>
            <Input
              placeholder="https://your-instance.my.salesforce.com"
              value={values.instanceUrl ?? ''}
              status={errors.instanceUrl ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...values, instanceUrl: e.target.value };
                setValues({ instanceUrl: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.instanceUrl && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Instance URL is required.</p>}
          </Block>
        ) : null,
      ({ values, errors, setValues, setErrors }: any) =>
        (values.authMode ?? 'access_token') === 'access_token' ? (
          <Block key="accessToken" title="Access Token" required>
            <Input.Password
              placeholder="Salesforce access token"
              value={values.accessToken ?? ''}
              status={errors.accessToken ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...values, accessToken: e.target.value };
                setValues({ accessToken: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.accessToken && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Access token is required.</p>}
          </Block>
        ) : null,
      ({ values, setValues }: any) =>
        (values.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="loginUrl" title="Login URL">
            <Input
              placeholder="https://login.salesforce.com"
              value={values.loginUrl ?? ''}
              onChange={(e) => setValues({ loginUrl: e.target.value })}
            />
            <p style={{ margin: '4px 0 0', color: '#7a7a7a', fontSize: 12 }}>
              Use <code>https://test.salesforce.com</code> for sandbox refresh-token authentication.
            </p>
          </Block>
        ) : null,
      ({ values, errors, setValues, setErrors }: any) =>
        (values.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="clientId" title="Client ID" required>
            <Input
              placeholder="Salesforce OAuth client ID"
              value={values.clientId ?? ''}
              status={errors.clientId ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...values, clientId: e.target.value };
                setValues({ clientId: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.clientId && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Client ID is required.</p>}
          </Block>
        ) : null,
      ({ values, errors, setValues, setErrors }: any) =>
        (values.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="clientSecret" title="Client Secret" required>
            <Input.Password
              placeholder="Salesforce OAuth client secret"
              value={values.clientSecret ?? ''}
              status={errors.clientSecret ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...values, clientSecret: e.target.value };
                setValues({ clientSecret: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.clientSecret && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Client secret is required.</p>}
          </Block>
        ) : null,
      ({ values, errors, setValues, setErrors }: any) =>
        (values.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="refreshToken" title="Refresh Token" required>
            <Input.Password
              placeholder="Salesforce refresh token"
              value={values.refreshToken ?? ''}
              status={errors.refreshToken ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...values, refreshToken: e.target.value };
                setValues({ refreshToken: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.refreshToken && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Refresh token is required.</p>}
          </Block>
        ) : null,
      ({ values, setValues }: any) => (
        <Block key="apiVersion" title="API Version">
          <Input
            placeholder="v61.0"
            value={values.apiVersion ?? ''}
            onChange={(e) => setValues({ apiVersion: e.target.value })}
          />
        </Block>
      ),
      'proxy',
      {
        key: 'rateLimitPerHour',
        defaultValue: 5000,
      },
    ],
  },
  dataScope: {
    title: 'Organizations',
  },
  scopeConfig: {
    entities: ['CROSS'],
    transformation: {
      objectTypes: ['Account', 'Contact', 'Lead', 'Opportunity', 'Case', 'Task', 'Event'],
      useCdc: false,
    },
  },
};
