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

type ScopeIdInput = {
  id?: unknown;
  githubId?: unknown;
  boardId?: unknown;
  gitlabId?: unknown;
  fullName?: unknown;
  bitbucketId?: unknown;
  projectKey?: unknown;
  planKey?: unknown;
  name?: unknown;
  gid?: unknown;
  projectId?: unknown;
};

export const getPluginScopeId = (plugin: string, scope: ScopeIdInput): string => {
  let id: unknown;

  switch (plugin) {
    case 'github':
      id = scope.githubId;
      break;
    case 'jira':
      id = scope.boardId;
      break;
    case 'gitlab':
      id = scope.gitlabId;
      break;
    case 'jenkins':
      id = scope.fullName;
      break;
    case 'bitbucket':
      id = scope.bitbucketId;
      break;
    case 'bitbucket_server':
      id = scope.bitbucketId;
      break;
    case 'sonarqube':
      id = scope.projectKey;
      break;
    case 'bamboo':
      id = scope.planKey;
      break;
    case 'argocd':
      id = scope.name;
      break;
    case 'asana':
      id = scope.gid;
      break;
    case 'plane':
      id = scope.projectId;
      break;
    default:
      id = scope.id;
  }

  if (id == null) {
    throw new Error(`Missing scope ID field for plugin "${plugin}"`);
  }

  return String(id);
};
