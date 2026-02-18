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
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/asana/impl"
	"github.com/apache/incubator-devlake/plugins/asana/models"
	"github.com/apache/incubator-devlake/plugins/asana/tasks"
)

func TestAsanaTagDataFlow(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId: 1,
			ProjectId:    "1234567890",
		},
	}

	// Import raw data for tags
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_tags.csv", "_raw_asana_tags")

	// Import tasks needed for tag collection (tags are collected per task)
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_tasks_for_tags.csv", &models.AsanaTask{})

	// Verify tag extraction
	dataflowTester.FlushTabler(&models.AsanaTag{})
	dataflowTester.FlushTabler(&models.AsanaTaskTag{})
	dataflowTester.Subtask(tasks.ExtractTagMeta, taskData)

	dataflowTester.VerifyTableWithOptions(models.AsanaTag{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_tags.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})

	dataflowTester.VerifyTableWithOptions(models.AsanaTaskTag{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_task_tags.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})
}

func TestAsanaTagWithMultipleTasks(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId: 1,
			ProjectId:    "1234567890",
		},
	}

	// Import raw data with multiple tasks having tags
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_tags_multiple.csv", "_raw_asana_tags")
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_tasks_multiple.csv", &models.AsanaTask{})

	// Extract tags
	dataflowTester.FlushTabler(&models.AsanaTag{})
	dataflowTester.FlushTabler(&models.AsanaTaskTag{})
	dataflowTester.Subtask(tasks.ExtractTagMeta, taskData)

	// Verify multiple task-tag relationships are created
	dataflowTester.VerifyTableWithOptions(models.AsanaTaskTag{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_task_tags_multiple.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})
}
