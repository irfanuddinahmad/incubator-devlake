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

import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';
import { Auth } from './connection-fields';

export const TaigaConfig: IPluginConfig = {
  plugin: 'taiga',
  name: 'Taiga',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 30,
  connection: {
    docLink: 'https://devlake.apache.org/docs/plugins/taiga',
    fields: [
      'name',
      ({ type, initialValues, values, errors, setValues, setErrors }: any) => (
        <Auth
          key="auth"
          type={type}
          initialValues={initialValues}
          values={values}
          errors={errors}
          setValues={setValues}
          setErrors={setErrors}
        />
      ),
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses dynamic rate limit for optimized data collection for Taiga. But you can adjust the collection speed by setting up your desirable rate limit.',
        defaultValue: 10000,
      },
    ],
  },
  dataScope: {
    title: 'Projects',
  },
  scopeConfig: {
    entities: ['TICKET'],
    transformation: {},
  },
};
