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

import { pick } from 'lodash';

const SAVE_CONNECTION_FIELDS = [
  'name',
  'endpoint',
  'authMethod',
  'authMode',
  'authType',
  'username',
  'password',
  'token',
  'accessToken',
  'refreshToken',
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
  'installationId',
  'proxy',
  'dbUrl',
  'companyId',
  'organization',
  'organizationId',
  'enterprise',
  'workspaceId',
  'workspaceSlug',
  'projectId',
  'portalId',
  'loginUrl',
  'instanceUrl',
  'apiVersion',
  'rateLimitPerHour',
  'tenantId',
  'tenantType',
  'usesApiToken',
  'enableWebhook',
  'webhookSharedKey',
];

type ConnectionFormValues = Record<string, unknown>;

export const buildConnectionSavePayload = (
  initialValues: ConnectionFormValues | undefined,
  values: ConnectionFormValues,
) => pick({ ...(initialValues ?? {}), ...values }, SAVE_CONNECTION_FIELDS);
