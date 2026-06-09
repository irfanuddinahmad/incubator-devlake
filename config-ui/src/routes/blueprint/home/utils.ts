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

import type { BlueprintConnectionPayload } from '../../../types/blueprint';
import { IBPMode } from '../../../types';

type BlueprintCreatePayload = {
  name: string;
  mode: IBPMode;
  enable: boolean;
  cronConfig: string;
  isManual: boolean;
  skipOnFail: boolean;
  connections?: BlueprintConnectionPayload[];
  plan?: unknown[][];
};

export const buildBlueprintCreatePayload = (
  name: string,
  mode: IBPMode,
  cronConfig: string,
): BlueprintCreatePayload => {
  const payload: BlueprintCreatePayload = {
    name,
    mode,
    enable: true,
    cronConfig,
    isManual: false,
    skipOnFail: true,
  };

  if (mode === IBPMode.NORMAL) {
    payload.connections = [];
  }

  if (mode === IBPMode.ADVANCED) {
    payload.plan = [[]];
  }

  return payload;
};
