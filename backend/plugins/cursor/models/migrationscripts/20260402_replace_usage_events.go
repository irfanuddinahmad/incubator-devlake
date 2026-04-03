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

// cursorModelUsage20260402 is the snapshot of CursorModelUsage used by this migration.
type cursorModelUsage20260402 struct {
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`
	Model        string    `gorm:"primaryKey;type:varchar(100)" json:"model"`
	Messages     int       `json:"messages"`
	archived.NoPKModel
}

func (cursorModelUsage20260402) TableName() string { return "_tool_cursor_model_usage" }

type replaceUsageEventsWithModelUsage struct{}

func (script *replaceUsageEventsWithModelUsage) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	// Drop the old event-based table (endpoint no longer exists).
	// Ignore the error in case the table doesn't exist yet.
	_ = db.DropTables("_tool_cursor_usage_events")
	return migrationhelper.AutoMigrateTables(basicRes, &cursorModelUsage20260402{})
}

func (script *replaceUsageEventsWithModelUsage) Version() uint64 { return 20260402000001 }
func (script *replaceUsageEventsWithModelUsage) Name() string {
	return "replace _tool_cursor_usage_events with _tool_cursor_model_usage"
}
