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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/migrationscripts/archived"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type codexConnection20260330 struct {
	archived.Model
	Name             string `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Endpoint         string `gorm:"type:varchar(255)" json:"endpoint"`
	Proxy            string `gorm:"type:varchar(255)" json:"proxy"`
	RateLimitPerHour int    `json:"rateLimitPerHour"`
	ApiKey           string `json:"apiKey"`
	ProjectId        string `gorm:"type:varchar(255)" json:"projectId"`
}

func (codexConnection20260330) TableName() string { return "_tool_codex_connections" }

type codexScope20260330 struct {
	archived.NoPKModel
	ConnectionId  uint64 `json:"connectionId" gorm:"primaryKey"`
	ScopeConfigId uint64 `json:"scopeConfigId,omitempty"`
	Id            string `json:"id" gorm:"primaryKey;type:varchar(255)"`
	Name          string `json:"name" gorm:"type:varchar(255)"`
	ProjectId     string `json:"projectId" gorm:"type:varchar(255)"`
}

func (codexScope20260330) TableName() string { return "_tool_codex_scopes" }

type codexScopeConfig20260330 struct {
	archived.Model
	ConnectionId uint64 `json:"connectionId" gorm:"primaryKey"`
	Name         string `gorm:"type:varchar(255)" json:"name"`
}

func (codexScopeConfig20260330) TableName() string { return "_tool_codex_scope_configs" }

type addCodexInitialTables struct{}

func (script *addCodexInitialTables) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&codexConnection20260330{},
		&codexScope20260330{},
		&codexScopeConfig20260330{},
	)
}

func (script *addCodexInitialTables) Version() uint64 { return 20260330000001 }
func (script *addCodexInitialTables) Name() string    { return "add Codex initial tables" }
