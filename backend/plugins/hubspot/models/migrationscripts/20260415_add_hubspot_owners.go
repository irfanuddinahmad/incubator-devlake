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

type hubspotOwner20260415 struct {
	archived.NoPKModel
	ConnectionId uint64 `gorm:"primaryKey"`
	OwnerId      string `gorm:"primaryKey;type:varchar(255)"`
	UserId       string `gorm:"type:varchar(255)"`
	Email        string `gorm:"type:varchar(255)"`
	FirstName    string `gorm:"type:varchar(255)"`
	LastName     string `gorm:"type:varchar(255)"`
	FullName     string `gorm:"type:varchar(255)"`
}

func (hubspotOwner20260415) TableName() string { return "_tool_hubspot_owners" }

type addHubspotOwners struct{}

func (script *addHubspotOwners) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(basicRes, &hubspotOwner20260415{})
}

func (script *addHubspotOwners) Version() uint64 { return 20260415000001 }
func (script *addHubspotOwners) Name() string    { return "add _tool_hubspot_owners table" }
