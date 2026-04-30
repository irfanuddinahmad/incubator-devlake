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

const getConnectionValues = (initialValues: Record<string, any>, values: Record<string, any>) => ({
  ...initialValues,
  ...values,
});

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
      loginUrl: '',
      instanceUrl: '',
      accessToken: '',
      refreshToken: '',
      clientId: '',
      clientSecret: '',
      apiVersion: '',
      rateLimitPerHour: 5000,
    },
    fields: [
      'name',
      ({ initialValues, values, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (
          <Block key="authMode" title="Authentication Mode" required>
            <Select
              style={{ width: '100%' }}
              value={formValues.authMode ?? 'access_token'}
              options={AUTH_MODE_OPTIONS}
              onChange={(value) => {
                const changedValues: Record<string, string> = { authMode: value };
                if (value === 'refresh_token' && formValues.instanceUrl) {
                  changedValues.loginUrl = formValues.instanceUrl;
                }
                if (value === 'access_token' && formValues.loginUrl) {
                  changedValues.instanceUrl = formValues.loginUrl;
                }
                const nextValues = { ...formValues, ...changedValues };
                setValues(changedValues);
                setErrors(validateConnection(nextValues));
              }}
            />
          </Block>
        );
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'access_token' ? (
          <Block key="instanceUrl" title="Instance URL" required>
            <Input
              name="salesforce-instance-url"
              autoComplete="off"
              placeholder="https://your-instance.my.salesforce.com"
              value={formValues.instanceUrl ?? ''}
              status={errors.instanceUrl ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...formValues, instanceUrl: e.target.value };
                setValues({ instanceUrl: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.instanceUrl && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Instance URL is required.</p>}
          </Block>
        ) : null;
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'access_token' ? (
          <Block key="accessToken" title="Access Token" required>
            <Input.Password
              name="salesforce-access-token"
              autoComplete="new-password"
              placeholder="Salesforce access token"
              value={formValues.accessToken ?? ''}
              status={errors.accessToken ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...formValues, accessToken: e.target.value };
                setValues({ accessToken: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.accessToken && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Access token is required.</p>}
          </Block>
        ) : null;
      },
      ({ initialValues, values, setValues }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="loginUrl" title="Login URL">
            <Input
              name="salesforce-login-url"
              autoComplete="off"
              placeholder="https://login.salesforce.com"
              value={formValues.loginUrl ?? ''}
              onChange={(e) => setValues({ loginUrl: e.target.value })}
            />
            <p style={{ margin: '4px 0 0', color: '#7a7a7a', fontSize: 12 }}>
              Use <code>https://test.salesforce.com</code> for sandbox refresh-token authentication.
            </p>
          </Block>
        ) : null;
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="clientId" title="Client ID" required>
            <Input
              name="salesforce-client-id"
              autoComplete="off"
              placeholder="Salesforce OAuth client ID"
              value={formValues.clientId ?? ''}
              status={errors.clientId ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...formValues, clientId: e.target.value };
                setValues({ clientId: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.clientId && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Client ID is required.</p>}
          </Block>
        ) : null;
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="clientSecret" title="Client Secret" required>
            <Input.Password
              name="salesforce-client-secret"
              autoComplete="new-password"
              placeholder="Salesforce OAuth client secret"
              value={formValues.clientSecret ?? ''}
              status={errors.clientSecret ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...formValues, clientSecret: e.target.value };
                setValues({ clientSecret: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.clientSecret && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Client secret is required.</p>}
          </Block>
        ) : null;
      },
      ({ initialValues, values, errors, setValues, setErrors }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (formValues.authMode ?? 'access_token') === 'refresh_token' ? (
          <Block key="refreshToken" title="Refresh Token" required>
            <Input.Password
              name="salesforce-refresh-token"
              autoComplete="new-password"
              placeholder="Salesforce refresh token"
              value={formValues.refreshToken ?? ''}
              status={errors.refreshToken ? 'error' : undefined}
              onChange={(e) => {
                const nextValues = { ...formValues, refreshToken: e.target.value };
                setValues({ refreshToken: e.target.value });
                setErrors(validateConnection(nextValues));
              }}
            />
            {errors.refreshToken && <p style={{ color: '#ff4d4f', marginTop: 8 }}>Refresh token is required.</p>}
          </Block>
        ) : null;
      },
      ({ initialValues, values, setValues }: any) => {
        const formValues = getConnectionValues(initialValues, values);

        return (
          <Block key="apiVersion" title="API Version">
            <Input
              name="salesforce-api-version"
              autoComplete="off"
              placeholder="v61.0"
              value={formValues.apiVersion ?? ''}
              onChange={(e) => setValues({ apiVersion: e.target.value })}
            />
          </Block>
        );
      },
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
