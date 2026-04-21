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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/migrationscripts/archived"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type cursorConnection20260327 struct {
	archived.Model
	Name             string `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Endpoint         string `gorm:"type:varchar(255)" json:"endpoint"`
	Proxy            string `gorm:"type:varchar(255)" json:"proxy"`
	RateLimitPerHour int    `json:"rateLimitPerHour"`
	ApiKey           string `json:"apiKey"`
	TeamId           string `gorm:"type:varchar(255)" json:"teamId"`
}

func (cursorConnection20260327) TableName() string { return "_tool_cursor_connections" }

type cursorScope20260327 struct {
	archived.NoPKModel
	ConnectionId  uint64 `json:"connectionId" gorm:"primaryKey"`
	ScopeConfigId uint64 `json:"scopeConfigId,omitempty"`
	Id            string `json:"id" gorm:"primaryKey;type:varchar(255)"`
	Name          string `json:"name" gorm:"type:varchar(255)"`
	TeamId        string `json:"teamId" gorm:"type:varchar(255)"`
}

func (cursorScope20260327) TableName() string { return "_tool_cursor_scopes" }

type cursorScopeConfig20260327 struct {
	archived.Model
	ConnectionId uint64 `json:"connectionId" gorm:"primaryKey"`
	Name         string `gorm:"type:varchar(255)" json:"name"`
}

func (cursorScopeConfig20260327) TableName() string { return "_tool_cursor_scope_configs" }

type cursorDailyUsage20260327 struct {
	ConnectionId       uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId            string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Day                time.Time `gorm:"primaryKey;type:date" json:"day"`
	UserEmail          string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`
	TotalTabsShown     int       `json:"totalTabsShown"`
	TotalTabsAccepted  int       `json:"totalTabsAccepted"`
	TotalLinesAdded    int       `json:"totalLinesAdded"`
	AcceptedLinesAdded int       `json:"acceptedLinesAdded"`
	TotalLinesDeleted  int       `json:"totalLinesDeleted"`
}

func (cursorDailyUsage20260327) TableName() string { return "_tool_cursor_daily_usage" }

type cursorUsageEvent20260327 struct {
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	EventId      string    `gorm:"primaryKey;type:varchar(255)" json:"eventId"`
	Timestamp    time.Time `json:"timestamp"`
	UserEmail    string    `gorm:"type:varchar(255)" json:"userEmail"`
	Model        string    `gorm:"type:varchar(100)" json:"model"`
	InputTokens  int64     `json:"inputTokens"`
	OutputTokens int64     `json:"outputTokens"`
	RequestCost  float64   `json:"requestCost"`
}

func (cursorUsageEvent20260327) TableName() string { return "_tool_cursor_usage_events" }

type cursorCommitAiShare20260327 struct {
	ConnectionId       uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId            string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	RepoName           string    `gorm:"primaryKey;type:varchar(255)" json:"repoName"`
	CommitSha          string    `gorm:"primaryKey;type:varchar(64)" json:"commitSha"`
	CommitDate         time.Time `gorm:"type:date" json:"commitDate"`
	TabLinesAdded      int       `json:"tabLinesAdded"`
	ComposerLinesAdded int       `json:"composerLinesAdded"`
	ManualLinesAdded   int       `json:"manualLinesAdded"`
}

func (cursorCommitAiShare20260327) TableName() string { return "_tool_cursor_commit_ai_share" }

type addCursorInitialTables struct{}

func (script *addCursorInitialTables) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&cursorConnection20260327{},
		&cursorScope20260327{},
		&cursorScopeConfig20260327{},
		&cursorDailyUsage20260327{},
		&cursorUsageEvent20260327{},
		&cursorCommitAiShare20260327{},
	)
}

func (script *addCursorInitialTables) Version() uint64 { return 20260327000001 }
func (script *addCursorInitialTables) Name() string    { return "add Cursor initial tables" }
