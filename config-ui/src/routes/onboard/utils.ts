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

import dayjs from 'dayjs';

import { getPluginScopeId } from '../../plugins/scope-id';
import type { BlueprintConnectionPayload } from '../../types/blueprint';
import { formatTime } from '../../utils/time';

const formatTimeAfter = (now: Date) => {
  const timeAfter = dayjs.utc(now).subtract(14, 'day').startOf('day').toDate();

  return formatTime(timeAfter, 'YYYY-MM-DD[T]HH:mm:ssZ', { utc: true });
};

type ScopePayloadInput = {
  data: Record<string, unknown>;
};

type OnboardBlueprintUpdatePayload = {
  connections: BlueprintConnectionPayload[];
  timeAfter?: string;
};

export const buildOnboardBlueprintUpdatePayload = (
  plugin: string,
  connectionId: string | number,
  scopes: ScopePayloadInput[],
  now = new Date(),
): OnboardBlueprintUpdatePayload => {
  const payload: OnboardBlueprintUpdatePayload = {
    connections: [
      {
        pluginName: plugin,
        connectionId,
        scopes: scopes.map((it) => ({
          scopeId: getPluginScopeId(plugin, it.data),
        })),
      },
    ],
  };

  if (plugin !== 'github') {
    payload.timeAfter = formatTimeAfter(now);
  }

  return payload;
};
