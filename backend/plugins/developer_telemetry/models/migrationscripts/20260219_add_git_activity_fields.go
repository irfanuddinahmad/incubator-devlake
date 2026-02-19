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
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addGitActivityFields)(nil)

type addGitActivityFields struct{}

type developerMetrics20260219 struct {
	GitActivity         string `gorm:"type:text"`
	DevelopmentActivity string `gorm:"type:text"`
}

func (developerMetrics20260219) TableName() string {
	return "_tool_developer_metrics"
}

func (*addGitActivityFields) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	
	// Add new columns for enhanced metrics
	err := db.AutoMigrate(&developerMetrics20260219{})
	if err != nil {
		return err
	}
	
	return nil
}

func (*addGitActivityFields) Version() uint64 {
	return 20260219000001
}

func (*addGitActivityFields) Name() string {
	return "Add git_activity and development_activity fields to developer_metrics"
}
