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

import "github.com/apache/incubator-devlake/plugins/cursor/models"

// CursorOptions holds the runtime options passed by the pipeline configuration.
type CursorOptions struct {
	ConnectionId uint64 `json:"connectionId" mapstructure:"connectionId"`
	ScopeId      string `json:"scopeId" mapstructure:"scopeId"`
	TeamId       string `json:"teamId" mapstructure:"teamId"`
}

// CursorTaskData is passed to every subtask via the task context.
type CursorTaskData struct {
	Options    *CursorOptions
	Connection *models.CursorConnection
}

// cursorRawParams identifies the raw data scope for Cursor records.
type cursorRawParams struct {
	ConnectionId uint64 `json:"connectionId"`
	ScopeId      string `json:"scopeId"`
	TeamId       string `json:"teamId"`
}

// GetParams implements helper.TaskOptions.
func (p cursorRawParams) GetParams() any { return p }
