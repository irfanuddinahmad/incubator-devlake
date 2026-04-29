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

type PlaneEpic20260420 struct {
	ConnectionId  uint64 `gorm:"primaryKey"`
	ProjectId     string `gorm:"primaryKey;type:varchar(255);index"`
	EpicId        string `gorm:"primaryKey;type:varchar(255)"`
	SequenceId    int
	Name          string `gorm:"type:varchar(255)"`
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
	Point         *int `gorm:"type:int"`
	CreatedDate   *time.Time
	UpdatedDate   *time.Time `gorm:"index"`
	CompletedAt   *time.Time
	StartDate     *time.Time `gorm:"type:date"`
	DueDate       *time.Time `gorm:"type:date"`
	ParentId      *string    `gorm:"type:varchar(255);index"`
	IsClosed      bool
	archived.NoPKModel
}

func (PlaneEpic20260420) TableName() string {
	return "_tool_plane_epics"
}

type addEpicTable20260420 struct{}

func (*addEpicTable20260420) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&PlaneEpic20260420{},
	)
}

func (*addEpicTable20260420) Version() uint64 {
	return 20260420000001
}

func (*addEpicTable20260420) Name() string {
	return "add plane epic table"
}
