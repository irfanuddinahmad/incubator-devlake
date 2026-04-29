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
	"gorm.io/gorm"
)

type PlaneConnection20260415 struct {
	ID               uint64         `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Name             string         `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Endpoint         string         `json:"endpoint"`
	ApiKey           string         `gorm:"serializer:encdec" json:"apiKey"`
	WorkspaceSlug    string         `json:"workspaceSlug"`
	Proxy            string         `json:"proxy"`
	RateLimitPerHour int            `json:"rateLimitPerHour"`
}

func (PlaneConnection20260415) TableName() string {
	return "_tool_plane_connections"
}

type addInitTables20260415 struct{}

func (*addInitTables20260415) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&PlaneConnection20260415{},
	)
}

func (*addInitTables20260415) Version() uint64 {
	return 20260415000001
}

func (*addInitTables20260415) Name() string {
	return "Plane init schemas"
}
