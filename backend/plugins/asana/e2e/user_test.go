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

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/asana/impl"
	"github.com/apache/incubator-devlake/plugins/asana/models"
	"github.com/apache/incubator-devlake/plugins/asana/tasks"
)

func TestAsanaUserDataFlow(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId: 1,
			ProjectId:    "1234567890",
		},
	}

	// Import raw data for users
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_users.csv", "_raw_asana_users")

	// Verify user extraction
	dataflowTester.FlushTabler(&models.AsanaUser{})
	dataflowTester.Subtask(tasks.ExtractUserMeta, taskData)

	dataflowTester.VerifyTableWithOptions(models.AsanaUser{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_users.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})

	// Verify user conversion to domain layer accounts
	dataflowTester.FlushTabler(&crossdomain.Account{})
	dataflowTester.Subtask(tasks.ConvertUserMeta, taskData)

	dataflowTester.VerifyTable(
		crossdomain.Account{},
		"./snapshot_tables/accounts.csv",
		[]string{
			"id",
			"email",
			"full_name",
			"user_name",
			"avatar_url",
		},
	)
}

func TestAsanaUserWithPhotoUrl(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId: 1,
			ProjectId:    "1234567890",
		},
	}

	// Import users with photo URLs
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_users_with_photos.csv", "_raw_asana_users")

	// Extract users
	dataflowTester.FlushTabler(&models.AsanaUser{})
	dataflowTester.Subtask(tasks.ExtractUserMeta, taskData)

	// Verify photo_url is extracted
	dataflowTester.VerifyTableWithOptions(models.AsanaUser{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_users_with_photos.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})
}
