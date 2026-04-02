/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import "time"

// CodexOptions holds the pipeline-level task options for the Codex plugin.
type CodexOptions struct {
	ConnectionId uint64 `json:"connectionId" mapstructure:"connectionId"`
	ScopeId      string `json:"scopeId" mapstructure:"scopeId"`

	// Date range for collection. Defaults to the last 30 days if not set.
	StartDate *time.Time `json:"startDate" mapstructure:"startDate"`
	EndDate   *time.Time `json:"endDate" mapstructure:"endDate"`
}

// codexRawParams identifies the collection scope for raw data storage.
// WorkspaceId is the ChatGPT Enterprise workspace ID from the connection and is
// included here so that raw records can be scoped when re-collecting after a
// workspace change.
type codexRawParams struct {
	ConnectionId uint64 `json:"connectionId"`
	ScopeId      string `json:"scopeId"`
	WorkspaceId  string `json:"workspaceId"`
}

// GetParams implements helper.TaskOptions.
func (p codexRawParams) GetParams() any { return p }
