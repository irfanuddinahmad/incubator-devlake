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

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

type NotionActivityEvent struct {
	common.NoPKModel
	ConnectionId     uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId          string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	EventId          string    `gorm:"primaryKey;type:varchar(255)" json:"eventId"`
	OccurredAt       time.Time `gorm:"type:timestamp" json:"occurredAt"`
	EditorUserId     string    `gorm:"type:varchar(255)" json:"editorUserId"`
	EditorUserEmail  string    `gorm:"type:varchar(255)" json:"editorUserEmail"`
	ActionType       string    `gorm:"type:varchar(255)" json:"actionType"`
	ObjectType       string    `gorm:"type:varchar(255)" json:"objectType"`
	ObjectId         string    `gorm:"type:varchar(255)" json:"objectId"`
	SourceObjectType string    `gorm:"type:varchar(100)" json:"sourceObjectType"`
	RawData          string    `gorm:"type:longtext" json:"rawData"`
}

func (NotionActivityEvent) TableName() string {
	return "_tool_notion_activity_events"
}
