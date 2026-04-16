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
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type PlaneProject20260416 struct {
	common.RawDataOrigin
	ConnectionId  uint64 `gorm:"primaryKey"`
	ProjectId     string `gorm:"primaryKey;type:varchar(255)"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	WorkspaceSlug string `gorm:"type:varchar(255)"`
	ScopeConfigId uint64
	Name          string `gorm:"type:varchar(255)"`
	Identifier    string `gorm:"type:varchar(255)"`
	Description   string `gorm:"type:text"`
	Network       int
}

func (PlaneProject20260416) TableName() string {
	return "_tool_plane_projects"
}

type PlaneScopeConfig20260416 struct {
	ID            uint64    `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Entities      string `gorm:"type:json"`
	ConnectionId  uint64 `gorm:"index"`
	Name          string `gorm:"type:varchar(255);uniqueIndex"`
	TypeMappings  string `gorm:"type:json"`
	StateMappings string `gorm:"type:json"`
}

func (PlaneScopeConfig20260416) TableName() string {
	return "_tool_plane_scope_configs"
}

type addScopeTables20260416 struct{}

func (*addScopeTables20260416) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&PlaneProject20260416{},
		&PlaneScopeConfig20260416{},
	)
}

func (*addScopeTables20260416) Version() uint64 {
	return 20260416000001
}

func (*addScopeTables20260416) Name() string {
	return "add plane scope tables"
}
