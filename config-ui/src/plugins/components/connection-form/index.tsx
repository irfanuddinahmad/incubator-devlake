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

import { useState, useEffect, useMemo } from 'react';
import { isEqual, pick } from 'lodash';
import { Flex, Alert, Button } from 'antd';

import API from '@/api';
import { useAppDispatch, useAppSelector } from '@/hooks';
import { ExternalLink } from '@/components';
import { addConnection, updateConnection } from '@/features';
import { selectConnection } from '@/features/connections';
import { getPluginConfig } from '@/plugins';
import { operator } from '@/utils';

import { Form } from './fields';
import { CONNECTION_FORM_FIELDS, buildConnectionSavePayload } from './payload';

interface Props {
  plugin: string;
  connectionId?: ID;
  onSuccess?: (id: ID) => void;
}

export const ConnectionForm = ({ plugin, connectionId, onSuccess }: Props) => {
  const [type, setType] = useState<'create' | 'update'>('create');
  const [values, setValues] = useState<any>({});
  const [errors, setErrors] = useState<Record<string, any>>({});
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [connectionDetail, setConnectionDetail] = useState<any>();

  const dispatch = useAppDispatch();
  const connection = useAppSelector((state) => selectConnection(state, `${plugin}-${connectionId}`));
  const selectedConnection = connectionDetail ?? connection;

  useEffect(() => {
    setType(connectionId ? 'update' : 'create');
  }, [connectionId]);

  useEffect(() => {
    let canceled = false;

    setConnectionDetail(undefined);
    setErrors({});

    if (!connectionId) {
      setValues({});
      return;
    }

    setValues(connection ? pick(connection, CONNECTION_FORM_FIELDS) : {});

    API.connection
      .get(plugin, connectionId)
      .then((res) => {
        if (canceled) {
          return;
        }

        setConnectionDetail(res);
        setValues(pick(res, CONNECTION_FORM_FIELDS));
        setErrors({});
      })
      .catch(() => undefined);

    return () => {
      canceled = true;
    };
  }, [plugin, connectionId]);

  const {
    name,
    connection: { docLink, fields, initialValues },
  } = getPluginConfig(plugin);

  const disabled = useMemo(() => {
    return Object.values(errors).some((value) => value);
  }, [errors]);

  const handleTest = async () => {
    await operator(
      () =>
        type === 'update' && connectionId
          ? API.connection.test(plugin, connectionId, {
              endpoint: isEqual(selectedConnection?.endpoint, values.endpoint) ? undefined : values.endpoint,
              authMethod: isEqual(selectedConnection?.authMethod, values.authMethod) ? undefined : values.authMethod,
              authMode: isEqual((selectedConnection as any)?.authMode, values.authMode) ? undefined : values.authMode,
              username: isEqual(selectedConnection?.username, values.username) ? undefined : values.username,
              password: isEqual(selectedConnection?.password, values.password) ? undefined : values.password,
              token: isEqual(selectedConnection?.token, values.token) ? undefined : values.token,
              accessToken: isEqual((selectedConnection as any)?.accessToken, values.accessToken)
                ? undefined
                : values.accessToken,
              refreshToken: isEqual((selectedConnection as any)?.refreshToken, values.refreshToken)
                ? undefined
                : values.refreshToken,
              appId: isEqual(selectedConnection?.appId, values.appId) ? undefined : values.appId,
              clientId: isEqual((selectedConnection as any)?.clientId, values.clientId) ? undefined : values.clientId,
              secretKey: isEqual(selectedConnection?.secretKey, values.secretKey) ? undefined : values.secretKey,
              clientSecret: isEqual((selectedConnection as any)?.clientSecret, values.clientSecret)
                ? undefined
                : values.clientSecret,
              proxy: isEqual(selectedConnection?.proxy, values.proxy) ? undefined : values.proxy,
              dbUrl: isEqual(selectedConnection?.dbUrl, values.dbUrl) ? undefined : values.dbUrl,
              companyId: isEqual(selectedConnection?.companyId, values.companyId) ? undefined : values.companyId,
              organization: isEqual(selectedConnection?.organization, values.organization)
                ? undefined
                : values.organization,
              organizationId: isEqual(selectedConnection?.organizationId, values.organizationId)
                ? undefined
                : values.organizationId,
              workspaceSlug: isEqual(selectedConnection?.workspaceSlug, values.workspaceSlug)
                ? undefined
                : values.workspaceSlug,
              loginUrl: isEqual((selectedConnection as any)?.loginUrl, values.loginUrl) ? undefined : values.loginUrl,
              instanceUrl: isEqual((selectedConnection as any)?.instanceUrl, values.instanceUrl)
                ? undefined
                : values.instanceUrl,
              apiVersion: isEqual((selectedConnection as any)?.apiVersion, values.apiVersion)
                ? undefined
                : values.apiVersion,
              enableWebhook: isEqual((selectedConnection as any)?.enableWebhook, values.enableWebhook)
                ? undefined
                : values.enableWebhook,
              webhookSharedKey: isEqual((selectedConnection as any)?.webhookSharedKey, values.webhookSharedKey)
                ? undefined
                : values.webhookSharedKey,
            } as any)
          : API.connection.testOld(
              plugin,
              pick({ ...initialValues, ...values }, [
                'name',
                'endpoint',
                'token',
                'accessToken',
                'refreshToken',
                'username',
                'password',
                'proxy',
                'authMethod',
                'authMode',
                'appId',
                'clientId',
                'secretKey',
                'clientSecret',
                'accessKeyId',
                'secretAccessKey',
                'region',
                'bucket',
                'identityStoreId',
                'identityStoreRegion',
                'rateLimitPerHour',
                'tenantId',
                'tenantType',
                'dbUrl',
                'companyId',
                'organization',
                'organizationId',
                'loginUrl',
                'instanceUrl',
                'apiVersion',
                'workspaceSlug',
                'enableWebhook',
                'webhookSharedKey',
              ]),
            ),
      {
        setOperating: setTesting,
        formatMessage: () => 'Test Connection Successfully.',
      },
    );
  };

  const handleSave = async () => {
    // Save and Test diverge on the update path: handleTest sends only fields
    // that changed vs `selectedConnection`, while handleSave sends the merged
    // (plugin defaults + form values), whitelisted payload. Don't unify these
    // without first confirming the save and test endpoints accept the same
    // shape.
    const payload = buildConnectionSavePayload(initialValues, values);
    const [success, res] = await operator(
      () =>
        !connectionId
          ? dispatch(addConnection({ plugin, ...payload })).unwrap()
          : dispatch(updateConnection({ plugin, connectionId, ...payload })).unwrap(),
      {
        setOperating: setSaving,
        formatMessage: () => (!connectionId ? 'Create a New Connection Successful.' : 'Update Connection Successful.'),
      },
    );

    if (success) {
      onSuccess?.(res.id);
    }
  };

  return (
    <Flex vertical gap="small">
      <Alert
        message={
          <>
            {' '}
            If you run into any problems while creating a new connection for {name},{' '}
            <ExternalLink link={docLink}>check out this doc</ExternalLink>.
          </>
        }
      />
      <Form
        type={type}
        name={name}
        fields={fields}
        initialValues={{ ...initialValues, ...(selectedConnection ?? {}) }}
        values={values}
        errors={errors}
        setValues={setValues}
        setErrors={setErrors}
      />
      <Flex justify="flex-end" gap="small">
        <Button htmlType="button" loading={testing} disabled={disabled || saving} onClick={handleTest}>
          Test Connection
        </Button>
        <Button htmlType="button" type="primary" loading={saving} disabled={disabled || testing} onClick={handleSave}>
          Save Connection
        </Button>
      </Flex>
    </Flex>
  );
};
