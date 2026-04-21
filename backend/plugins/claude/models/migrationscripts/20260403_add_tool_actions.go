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

// claudeUsage20260403 mirrors models.ClaudeUsage at this migration version,
// adding tool_actions and cache token columns.
type claudeUsage20260403 struct {
	ConnectionId uint64    `gorm:"primaryKey"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)"`
	Date         time.Time `gorm:"primaryKey;type:date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)"`
	Model        string    `gorm:"primaryKey;type:varchar(100)"`

	NumSessions     int
	LinesAdded      int
	LinesRemoved    int
	CommitsByClaude int
	PrsByClaude     int

	EditToolAccepted         int
	EditToolRejected         int
	MultiEditToolAccepted    int
	MultiEditToolRejected    int
	WriteToolAccepted        int
	WriteToolRejected        int
	NotebookEditToolAccepted int
	NotebookEditToolRejected int

	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	EstimatedCostUsd    float64
}

func (claudeUsage20260403) TableName() string { return "_tool_claude_usage" }

type addToolActionsAndCacheTokens struct{}

func (script *addToolActionsAndCacheTokens) Up(basicRes context.BasicRes) errors.Error {
	// AutoMigrate adds the new columns without touching existing data.
	return migrationhelper.AutoMigrateTables(basicRes, &claudeUsage20260403{})
}

func (script *addToolActionsAndCacheTokens) Version() uint64 { return 20260403000001 }
func (script *addToolActionsAndCacheTokens) Name() string {
	return "add tool_actions and cache token columns to _tool_claude_usage"
}
