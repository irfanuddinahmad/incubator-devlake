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

// CodexUsage stores one daily Codex usage record from the Analytics API
// endpoint GET /analytics/codex/workspaces/{workspace_id}/usage.
//
// A record is uniquely identified by (connection, scope, date, client_surface, user_email).
// When the API is queried without a per-user breakdown, user_email is empty and the
// record represents workspace-level aggregate data for that surface and day.
type CodexUsage struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	// ClientSurface is the Codex client used: "cli", "ide", "cloud", or "code_review".
	ClientSurface string `gorm:"primaryKey;type:varchar(50);default:''" json:"clientSurface"`
	// UserEmail is populated when the API is queried with per-user breakdown.
	// Empty for workspace-level aggregate records.
	UserEmail string `gorm:"primaryKey;type:varchar(255);default:''" json:"userEmail"`

	// Threads is the number of Codex sessions/tasks started.
	Threads int64 `json:"threads"`
	// Turns is the total number of conversation turns (user→model interactions).
	Turns int64 `json:"turns"`
	// Credits is the internal usage-credit quantity consumed. Credits are
	// OpenAI's billing unit for Codex; they do not directly map to USD.
	Credits float64 `json:"credits"`
}

func (CodexUsage) TableName() string {
	return "_tool_codex_usage"
}
