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

type PlaneWorkItem struct {
	ConnectionId  uint64 `gorm:"primaryKey"`
	ProjectId     string `gorm:"primaryKey;type:varchar(255);index"`
	WorkItemId    string `gorm:"primaryKey;type:varchar(255)"`
	SequenceId    int
	Title         string `gorm:"type:varchar(255)"`
	Description   string `gorm:"type:text"`
	TypeId        string `gorm:"type:varchar(255);index"`
	TypeName      string `gorm:"type:varchar(255)"`
	StateId       string `gorm:"type:varchar(255);index"`
	StateName     string `gorm:"type:varchar(255)"`
	StateGroup    string `gorm:"type:varchar(100)"`
	Priority      string `gorm:"type:varchar(100)"`
	AssigneeId    string `gorm:"type:varchar(255)"`
	AssigneeName  string `gorm:"type:varchar(255)"`
	EstimatePoint *float64
	CreatedDate   *time.Time
	UpdatedDate   *time.Time `gorm:"index"`
	CompletedAt   *time.Time
	StartDate     *time.Time `gorm:"type:date"`
	DueDate       *time.Time `gorm:"type:date"`
	ParentId      *string    `gorm:"type:varchar(255);index"`
	IsClosed      bool
	common.NoPKModel
}

func (PlaneWorkItem) TableName() string {
	return "_tool_plane_work_items"
}
