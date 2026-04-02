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
type cursorDailyUsage20260403 struct {
	ConnectionId uint64    `gorm:"primaryKey"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)"`
	Day          time.Time `gorm:"primaryKey;type:date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)"`

	TotalTabsShown    int
	TotalTabsAccepted int

	TotalLinesAdded      int
	TotalLinesDeleted    int
	AcceptedLinesAdded   int
	AcceptedLinesDeleted int

	TotalApplies int
	TotalAccepts int
	TotalRejects int

	ComposerRequests int
	ChatRequests     int
	AgentRequests    int
	CmdkUsages       int

	SubscriptionIncludedReqs int
	ApiKeyReqs               int
	UsageBasedReqs           int

	BugbotUsages  int
	MostUsedModel string `gorm:"type:varchar(100)"`
	ClientVersion string `gorm:"type:varchar(50)"`

	archived.NoPKModel
}

func (cursorDailyUsage20260403) TableName() string { return "_tool_cursor_daily_usage" }

// cursorUsageEvent20260403 is the snapshot of CursorUsageEvent used by this migration.
type cursorUsageEvent20260403 struct {
	ConnectionId uint64    `gorm:"primaryKey"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)"`
	Timestamp    time.Time `gorm:"primaryKey;type:datetime(3)"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)"`
	Model        string    `gorm:"primaryKey;type:varchar(100)"`

	Kind             string `gorm:"type:varchar(100)"`
	MaxMode          bool
	RequestsCosts    float64
	IsTokenBasedCall bool
	IsChargeable     bool
	IsHeadless       bool

	InputTokens      int64
	OutputTokens     int64
	CacheWriteTokens int64
	CacheReadTokens  int64
	TotalCents       float64
	ChargedCents     float64

	archived.NoPKModel
}

func (cursorUsageEvent20260403) TableName() string { return "_tool_cursor_usage_events" }

type adminApiSchema struct{}

func (script *adminApiSchema) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	// Drop the old model_usage table (replaced by usage_events from Admin API).
	// Ignore the error in case the table doesn't exist yet.
	_ = db.DropTables("_tool_cursor_model_usage")
	// Drop and recreate daily_usage to pick up the new schema columns.
	_ = db.DropTables("_tool_cursor_daily_usage")
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&cursorDailyUsage20260403{},
		&cursorUsageEvent20260403{},
	)
}

func (script *adminApiSchema) Version() uint64 { return 20260403000001 }
func (script *adminApiSchema) Name() string {
	return "migrate cursor data tables to admin api schema"
}
