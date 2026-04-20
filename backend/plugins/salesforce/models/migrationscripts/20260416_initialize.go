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

type salesforceConnection20260416 struct {
	archived.Model
	Name             string `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Endpoint         string `gorm:"type:varchar(255)" json:"endpoint"`
	Proxy            string `gorm:"type:varchar(255)" json:"proxy"`
	AuthMode         string `gorm:"column:auth_mode;type:varchar(32)" json:"authMode"`
	AccessToken      string `gorm:"column:access_token" json:"accessToken"`
	RefreshToken     string `gorm:"column:refresh_token" json:"refreshToken"`
	ClientId         string `gorm:"column:client_id;type:varchar(255)" json:"clientId"`
	ClientSecret     string `gorm:"column:client_secret" json:"clientSecret"`
	LoginUrl         string `gorm:"column:login_url;type:varchar(255)" json:"loginUrl"`
	InstanceUrl      string `gorm:"column:instance_url;type:varchar(255)" json:"instanceUrl"`
	ApiVersion       string `gorm:"column:api_version;type:varchar(32)" json:"apiVersion"`
	RateLimitPerHour int    `json:"rateLimitPerHour"`
}

func (salesforceConnection20260416) TableName() string { return "_tool_salesforce_connections" }

type salesforceScope20260416 struct {
	archived.NoPKModel
	ConnectionId  uint64 `json:"connectionId" gorm:"primaryKey"`
	ScopeConfigId uint64 `json:"scopeConfigId,omitempty"`
	Id            string `json:"id" gorm:"primaryKey;type:varchar(255)"`
	Name          string `json:"name" gorm:"type:varchar(255)"`
}

func (salesforceScope20260416) TableName() string { return "_tool_salesforce_scopes" }

type salesforceScopeConfig20260416 struct {
	archived.Model
	ConnectionId uint64   `json:"connectionId" gorm:"primaryKey"`
	Name         string   `gorm:"type:varchar(255)" json:"name"`
	ObjectTypes  []string `gorm:"serializer:json" json:"objectTypes"`
	UseCdc       bool     `gorm:"column:use_cdc" json:"useCdc"`
}

func (salesforceScopeConfig20260416) TableName() string { return "_tool_salesforce_scope_configs" }

type salesforceActivityEvent20260416 struct {
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

func (salesforceActivityEvent20260416) TableName() string { return "_tool_salesforce_activity_events" }

type salesforceUser20260416 struct {
	archived.NoPKModel
	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	UserId       string `gorm:"primaryKey;type:varchar(255)" json:"userId"`
	Name         string `gorm:"type:varchar(255)" json:"name"`
	Username     string `gorm:"type:varchar(255)" json:"username"`
	Email        string `gorm:"type:varchar(255)" json:"email"`
	IsActive     bool   `json:"isActive"`
}

func (salesforceUser20260416) TableName() string { return "_tool_salesforce_users" }

type salesforceCdcCheckpoint20260416 struct {
	archived.NoPKModel
	ConnectionId  uint64     `gorm:"primaryKey" json:"connectionId"`
	ScopeId       string     `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Channel       string     `gorm:"primaryKey;type:varchar(255)" json:"channel"`
	ReplayId      int64      `json:"replayId"`
	LastEventTime *time.Time `gorm:"type:timestamp" json:"lastEventTime"`
}

func (salesforceCdcCheckpoint20260416) TableName() string { return "_tool_salesforce_cdc_checkpoints" }

type addSalesforceInitialTables struct{}

func (script *addSalesforceInitialTables) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&salesforceConnection20260416{},
		&salesforceScope20260416{},
		&salesforceScopeConfig20260416{},
		&salesforceActivityEvent20260416{},
		&salesforceUser20260416{},
		&salesforceCdcCheckpoint20260416{},
	)
}

func (script *addSalesforceInitialTables) Version() uint64 { return 20260416000001 }
func (script *addSalesforceInitialTables) Name() string    { return "add Salesforce initial tables" }
