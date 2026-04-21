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

type hubspotConnection20260406 struct {
	archived.Model
	Name             string `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Endpoint         string `gorm:"type:varchar(255)" json:"endpoint"`
	Proxy            string `gorm:"type:varchar(255)" json:"proxy"`
	RateLimitPerHour int    `json:"rateLimitPerHour"`
	ApiToken         string `json:"apiToken"`
	PortalId         string `gorm:"type:varchar(255)" json:"portalId"`
}

func (hubspotConnection20260406) TableName() string { return "_tool_hubspot_connections" }

type hubspotScope20260406 struct {
	archived.NoPKModel
	ConnectionId  uint64 `json:"connectionId" gorm:"primaryKey"`
	ScopeConfigId uint64 `json:"scopeConfigId,omitempty"`
	Id            string `json:"id" gorm:"primaryKey;type:varchar(255)"`
	Name          string `json:"name" gorm:"type:varchar(255)"`
}

func (hubspotScope20260406) TableName() string { return "_tool_hubspot_scopes" }

type hubspotScopeConfig20260406 struct {
	archived.Model
	ConnectionId uint64 `json:"connectionId" gorm:"primaryKey"`
	Name         string `gorm:"type:varchar(255)" json:"name"`
}

func (hubspotScopeConfig20260406) TableName() string { return "_tool_hubspot_scope_configs" }

type hubspotActivityEvent20260406 struct {
	archived.NoPKModel
	ConnectionId     uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId          string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	EventId          string    `gorm:"primaryKey;type:varchar(255)" json:"eventId"`
	OccurredAt       time.Time `gorm:"type:timestamp" json:"occurredAt"`
	ActingUserId     string    `gorm:"type:varchar(255)" json:"actingUserId"`
	ActingUserEmail  string    `gorm:"type:varchar(255)" json:"actingUserEmail"`
	ActionType       string    `gorm:"type:varchar(255)" json:"actionType"`
	ObjectType       string    `gorm:"type:varchar(255)" json:"objectType"`
	ObjectId         string    `gorm:"type:varchar(255)" json:"objectId"`
	SourceObjectType string    `gorm:"type:varchar(100)" json:"sourceObjectType"`
	RawData          string    `gorm:"type:longtext" json:"rawData"`
}

func (hubspotActivityEvent20260406) TableName() string { return "_tool_hubspot_activity_events" }

type addHubspotInitialTables struct{}

func (script *addHubspotInitialTables) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&hubspotConnection20260406{},
		&hubspotScope20260406{},
		&hubspotScopeConfig20260406{},
		&hubspotActivityEvent20260406{},
	)
}

func (script *addHubspotInitialTables) Version() uint64 { return 20260406000001 }
func (script *addHubspotInitialTables) Name() string    { return "add HubSpot initial tables" }
