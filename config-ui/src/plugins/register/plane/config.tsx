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

import { DOC_URL } from '@/release';
import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';
import { WorkspaceSlug } from './connection-fields';

export const PlaneConfig: IPluginConfig = {
  plugin: 'plane',
  name: 'Plane',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 16,
  isBeta: true,
  connection: {
    docLink: DOC_URL.PLUGIN.PLANE.BASIS,
    initialValues: {
      endpoint: 'https://api.plane.so',
    },
    fields: [
      'name',
      {
        key: 'endpoint',
        label: 'Endpoint',
        subLabel: 'Plane API base URL. Use the cloud default or your self-hosted Plane endpoint.',
      },
      {
        key: 'token',
        label: 'API Key',
        subLabel: 'Workspace-scoped Plane API key used for authentication.',
      },
      WorkspaceSlug,
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel: 'Maximum number of API requests per hour. Leave blank for default.',
      },
    ],
  },
  dataScope: {
    title: 'Projects',
    searchPlaceholder: 'Search projects...',
  },
};
