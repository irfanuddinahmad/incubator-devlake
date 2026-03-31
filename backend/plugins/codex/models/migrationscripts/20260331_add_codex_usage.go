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

type addCodexUsageTable struct{}

func (script *addCodexUsageTable) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&codexUsage20260331{},
	)
}

func (*addCodexUsageTable) Version() uint64 { return 20260331000001 }
func (*addCodexUsageTable) Name() string    { return "add Codex usage table" }

type codexUsage20260331 struct {
	archived.NoPKModel
	ConnectionId     uint64    `gorm:"primaryKey"`
	ScopeId          string    `gorm:"primaryKey;type:varchar(255)"`
	Date             time.Time `gorm:"primaryKey;type:date"`
	Model            string    `gorm:"primaryKey;type:varchar(100)"`
	InputTokens      int64
	OutputTokens     int64
	EstimatedCostUsd float64
}

func (codexUsage20260331) TableName() string { return "_tool_codex_usage" }
