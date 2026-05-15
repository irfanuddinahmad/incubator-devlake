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

export interface GithubInstallation {
  id: number;
  account: {
    login: string;
  };
}

export interface GithubAppSettings {
  appId?: string;
  secretKey?: string;
  installationId?: number;

  status: 'idle' | 'valid' | 'invalid';
  from?: string;
  installations?: GithubInstallation[];
}

export const isMaskedGithubAppSecret = (secretKey?: string) => !!secretKey && secretKey.includes('***');

export const shouldValidateGithubAppConfig = (endpoint?: string, appId?: string, secretKey?: string) =>
  !!endpoint && !!appId && !!secretKey && !isMaskedGithubAppSecret(secretKey);

export const buildGithubInstallationOptions = (installations?: GithubInstallation[], installationId?: number) => {
  const options = installations
    ? installations.map((it) => ({
        value: it.id,
        label: it.account.login,
      }))
    : [];

  if (installationId && !options.some((it) => it.value === installationId)) {
    return [...options, { value: installationId, label: `Saved installation (${installationId})` }];
  }

  return options;
};

export const invalidateGithubAppConfig = (
  settings: GithubAppSettings,
  changedValues: Partial<Pick<GithubAppSettings, 'appId' | 'secretKey'>>,
): GithubAppSettings => {
  return {
    ...settings,
    ...changedValues,
    installationId: undefined,
    status: 'idle',
    from: undefined,
    installations: undefined,
  };
};
