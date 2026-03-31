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

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

// CodexUsage stores daily aggregate Codex (OpenAI Codex / ChatGPT API) token usage
// ingested from the OpenAI Usage API (GET /v1/usage?date=<date>&project_id=<id>).
// The API aggregates across all users in a project; per-user breakdown is not available
// on the public endpoint.
type CodexUsage struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	Model        string    `gorm:"primaryKey;type:varchar(100)" json:"model"`

	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	EstimatedCostUsd float64 `json:"estimatedCostUsd"`
}

func (CodexUsage) TableName() string {
	return "_tool_codex_usage"
}
