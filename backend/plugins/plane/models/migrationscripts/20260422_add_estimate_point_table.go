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
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type PlaneEstimatePoint20260422 struct {
	ConnectionId uint64 `gorm:"primaryKey"`
	ProjectId    string `gorm:"primaryKey;type:varchar(255);index"`
	PointId      string `gorm:"primaryKey;type:varchar(100)"`
	EstimateId   string `gorm:"type:varchar(100);index"`
	Key          int
	Value        *float64
	ValueLabel   string `gorm:"type:varchar(100)"`
	Description  string `gorm:"type:text"`
	common.NoPKModel
}

func (PlaneEstimatePoint20260422) TableName() string {
	return "_tool_plane_estimate_points"
}

type addEstimatePointTable20260422 struct{}

func (*addEstimatePointTable20260422) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&PlaneEstimatePoint20260422{},
	)
}

func (*addEstimatePointTable20260422) Version() uint64 {
	return 20260422000001
}

func (*addEstimatePointTable20260422) Name() string {
	return "add plane estimate point table"
}
