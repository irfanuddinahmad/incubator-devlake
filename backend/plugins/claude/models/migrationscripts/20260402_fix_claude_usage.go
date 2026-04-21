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
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

// fixClaudeUsageSchema drops and recreates _tool_claude_usage with the correct
// primary key: (connection_id, scope_id, date, user_email, model).
type fixClaudeUsageSchema struct{}

func (script *fixClaudeUsageSchema) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	// Drop the old table; ignore any error if it doesn't yet exist.
	_ = db.DropTables("_tool_claude_usage")
	return migrationhelper.AutoMigrateTables(basicRes, &claudeUsage20260402{})
}

func (script *fixClaudeUsageSchema) Version() uint64 {
	return 20260402000001
}

func (script *fixClaudeUsageSchema) Name() string {
	return "fix _tool_claude_usage schema: add scope_id and model to primary key"
}

// claudeUsage20260402 mirrors models.ClaudeUsage at this migration version.
type claudeUsage20260402 struct {
	ConnectionId     uint64    `gorm:"primaryKey"`
	ScopeId          string    `gorm:"primaryKey;type:varchar(255)"`
	Date             time.Time `gorm:"primaryKey;type:date"`
	UserEmail        string    `gorm:"primaryKey;type:varchar(255)"`
	Model            string    `gorm:"primaryKey;type:varchar(100)"`
	NumSessions      int
	LinesAdded       int
	LinesRemoved     int
	CommitsByClaude  int
	PrsByClaude      int
	InputTokens      int64
	OutputTokens     int64
	EstimatedCostUsd float64
}

func (claudeUsage20260402) TableName() string {
	return "_tool_claude_usage"
}
