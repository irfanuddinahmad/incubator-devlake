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
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addAiActivitiesUnifiedSchema)(nil)

type aiActivity20260331 struct {
	archived.DomainEntity
	Provider         string    `gorm:"type:varchar(100);index"`
	AccountId        string    `gorm:"type:varchar(255);index"`
	UserEmail        string    `gorm:"type:varchar(255)"`
	Date             time.Time `gorm:"type:date;index"`
	Type             string    `gorm:"type:varchar(100)"`
	Model            string    `gorm:"type:varchar(100)"`
	InterfaceType    string    `gorm:"type:varchar(50)"`
	NumSessions      int
	SuggestionsCount int
	AcceptanceCount  int
	LinesAdded       int
	LinesRemoved     int
	CommitsCreated   int
	PrsCreated       int
	InputTokens      int64
	OutputTokens     int64
	EstimatedCostUsd float64
}

func (aiActivity20260331) TableName() string {
	return "ai_activities"
}

type addAiActivitiesUnifiedSchema struct{}

// Up runs the migration.
//
// Changes to ai_activities:
//   - Rename commits_by_claude  → commits_created
//   - Rename prs_by_claude      → prs_created
//   - Add interface_type   varchar(50)
//   - Add suggestions_count int (default 0)
//   - Add acceptance_count  int (default 0)
//
// The RENAME is implemented as ADD+COPY+DROP so it works across MySQL, PostgreSQL,
// and SQLite without dialect-specific RENAME COLUMN support issues.
// AutoMigrate (called after all migration scripts) will then reconcile any remaining
// column differences against the updated Go struct.
func (script *addAiActivitiesUnifiedSchema) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// If the table doesn't exist (fresh install or e2e tests), completely create it
	// using AutoMigrate, bypassing the need to rename missing legacy columns.
	if !db.HasTable("ai_activities") {
		return db.AutoMigrate(&aiActivity20260331{})
	}

	// 1. Rename commits_by_claude → commits_created
	if err := db.AddColumn("ai_activities", "commits_created", "bigint"); err != nil {
		// column may already exist if migration was partially applied — ignore duplicate
		_ = err
	}
	_ = db.Exec("UPDATE ai_activities SET commits_created = commits_by_claude WHERE commits_by_claude IS NOT NULL")
	_ = db.DropColumns("ai_activities", "commits_by_claude")

	// 2. Rename prs_by_claude → prs_created
	if err := db.AddColumn("ai_activities", "prs_created", "bigint"); err != nil {
		_ = err
	}
	_ = db.Exec("UPDATE ai_activities SET prs_created = prs_by_claude WHERE prs_by_claude IS NOT NULL")
	_ = db.DropColumns("ai_activities", "prs_by_claude")

	// 3. Add new columns (AddColumn is idempotent-friendly — errors on duplicate are silently ignored)
	_ = db.AddColumn("ai_activities", "interface_type", "varchar(50)")
	_ = db.AddColumn("ai_activities", "suggestions_count", "bigint")
	_ = db.AddColumn("ai_activities", "acceptance_count", "bigint")

	return nil
}

func (*addAiActivitiesUnifiedSchema) Version() uint64 {
	return 20260331000001
}

func (*addAiActivitiesUnifiedSchema) Name() string {
	return "add unified schema fields to ai_activities (rename commits/prs, add interface_type, suggestions, acceptance)"
}
