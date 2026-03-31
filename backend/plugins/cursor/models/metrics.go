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

// CursorDailyUsage stores per-user daily usage metrics from GET /teams/daily-usage-data.
type CursorDailyUsage struct {
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Day          time.Time `gorm:"primaryKey;type:date" json:"day"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`

	TotalTabsShown     int `json:"totalTabsShown"`
	TotalTabsAccepted  int `json:"totalTabsAccepted"`
	TotalLinesAdded    int `json:"totalLinesAdded"`
	AcceptedLinesAdded int `json:"acceptedLinesAdded"`
	TotalLinesDeleted  int `json:"totalLinesDeleted"`

	common.NoPKModel
}

func (CursorDailyUsage) TableName() string {
	return "_tool_cursor_daily_usage"
}

// CursorUsageEvent stores raw usage events from GET /teams/filtered-usage-events.
type CursorUsageEvent struct {
	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	EventId      string `gorm:"primaryKey;type:varchar(255)" json:"eventId"`

	Timestamp    time.Time `json:"timestamp"`
	UserEmail    string    `gorm:"type:varchar(255)" json:"userEmail"`
	Model        string    `gorm:"type:varchar(100)" json:"model"`
	InputTokens  int64     `json:"inputTokens"`
	OutputTokens int64     `json:"outputTokens"`
	RequestCost  float64   `json:"requestCost"`

	common.NoPKModel
}

func (CursorUsageEvent) TableName() string {
	return "_tool_cursor_usage_events"
}

// CursorCommitAiShare stores per-commit AI code attribution from GET /analytics/ai-code/commits.
// This endpoint is only available on Enterprise plans.
type CursorCommitAiShare struct {
	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	RepoName     string `gorm:"primaryKey;type:varchar(255)" json:"repoName"`
	CommitSha    string `gorm:"primaryKey;type:varchar(64)" json:"commitSha"`

	CommitDate         time.Time `gorm:"type:date" json:"commitDate"`
	TabLinesAdded      int       `json:"tabLinesAdded"`
	ComposerLinesAdded int       `json:"composerLinesAdded"`
	ManualLinesAdded   int       `json:"manualLinesAdded"`

	common.NoPKModel
}

func (CursorCommitAiShare) TableName() string {
	return "_tool_cursor_commit_ai_share"
}
