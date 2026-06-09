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

import { deepEqual, equal, throws } from 'node:assert/strict';
import { test } from 'node:test';

import { buildOnboardBlueprintUpdatePayload } from './utils';

test('github onboard blueprint update payload does not set timeAfter', () => {
  const payload = buildOnboardBlueprintUpdatePayload('github', 1, [{ data: { githubId: 1001 } }]);

  equal(Object.hasOwn(payload, 'timeAfter'), false);
  deepEqual(payload, {
    connections: [
      {
        pluginName: 'github',
        connectionId: 1,
        scopes: [{ scopeId: '1001' }],
      },
    ],
  });
});

test('non-github onboard blueprint update payload preserves the previous explicit 14-day timeAfter', () => {
  const payload = buildOnboardBlueprintUpdatePayload(
    'jira',
    1,
    [{ data: { boardId: 1001 } }],
    new Date('2026-06-08T12:34:56Z'),
  );

  equal(Object.hasOwn(payload, 'timeAfter'), true);
  // formatTime is called with { utc: true } so the offset is always +00:00.
  equal(payload.timeAfter, '2026-05-25T00:00:00+00:00');
  deepEqual(payload.connections, [
    {
      pluginName: 'jira',
      connectionId: 1,
      scopes: [{ scopeId: '1001' }],
    },
  ]);
});

test('onboard blueprint update payload rejects a scope without the plugin scope ID', () => {
  throws(() => buildOnboardBlueprintUpdatePayload('github', 1, [{ data: {} }]), /Missing scope ID field/);
});
