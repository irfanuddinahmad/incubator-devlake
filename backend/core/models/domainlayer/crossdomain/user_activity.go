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

package crossdomain

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/domainlayer"
)

// UserActivity is a generic cross-tool activity table for non-code, non-ticket
// user actions collected from external collaboration platforms such as HubSpot
// and Notion. It intentionally mirrors the shared-domain pattern used by
// ai_activities while preserving activity-specific fields.
type UserActivity struct {
	domainlayer.DomainEntity

	ConnectionId uint64 `gorm:"index" json:"connectionId"`
	ScopeId      string `gorm:"type:varchar(255);index" json:"scopeId"`

	SourceSystem  string `gorm:"type:varchar(100);index" json:"sourceSystem"`
	SourceEventId string `gorm:"type:varchar(255)" json:"sourceEventId"`

	AccountId    string `gorm:"type:varchar(255);index" json:"accountId"`
	UserEmail    string `gorm:"type:varchar(255);index" json:"userEmail"`
	UserDisplay  string `gorm:"type:varchar(255)" json:"userDisplay"`
	NativeUserId string `gorm:"type:varchar(255)" json:"nativeUserId"`

	ActionType string `gorm:"type:varchar(100);index" json:"actionType"`
	ObjectType string `gorm:"type:varchar(100)" json:"objectType"`
	ObjectId   string `gorm:"type:varchar(255)" json:"objectId"`
	ObjectRef  string `gorm:"type:varchar(512)" json:"objectRef"`

	ActionTime time.Time `gorm:"type:timestamp;index" json:"actionTime"`
	ActionDay  time.Time `gorm:"type:date;index" json:"actionDay"`
	Summary    string    `gorm:"type:text" json:"summary"`
}

func (UserActivity) TableName() string {
	return "user_activities"
}
