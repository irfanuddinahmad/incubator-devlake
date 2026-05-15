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

import { equal } from 'node:assert/strict';
import { test } from 'node:test';

import { isMaskedGithubToken } from './token-utils';

test('detects masked GitHub tokens returned by the API', () => {
  equal(isMaskedGithubToken('ghp_example********************Suffix12'), true);
  equal(
    isMaskedGithubToken('github_pat_example******************************************************************Suffix12'),
    true,
  );
});

test('does not treat real GitHub tokens as masked placeholders', () => {
  equal(isMaskedGithubToken('ghp_exampleTokenWithoutMaskCharacters12345'), false);
  equal(isMaskedGithubToken(''), false);
  equal(isMaskedGithubToken(undefined), false);
});
