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
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/helpers/e2ehelper"
	"github.com/apache/incubator-devlake/plugins/asana/impl"
	"github.com/apache/incubator-devlake/plugins/asana/models"
	"github.com/apache/incubator-devlake/plugins/asana/tasks"
)

func TestAsanaTaskConversionDataFlow(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId:  1,
			ProjectId:     "1234567890",
			ScopeConfigId: 1,
		},
	}

	// Import raw data tables
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_tasks.csv", "_raw_asana_tasks")

	// Import tool layer data needed for conversion
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_projects.csv", &models.AsanaProject{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_scope_configs.csv", &models.AsanaScopeConfig{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_tags.csv", &models.AsanaTag{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_task_tags.csv", &models.AsanaTaskTag{})

	// Verify task extraction
	dataflowTester.FlushTabler(&models.AsanaTask{})
	dataflowTester.Subtask(tasks.ExtractTaskMeta, taskData)
	dataflowTester.VerifyTableWithOptions(models.AsanaTask{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/_tool_asana_tasks.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})

	// Verify task conversion to domain layer
	dataflowTester.FlushTabler(&ticket.Issue{})
	dataflowTester.FlushTabler(&ticket.BoardIssue{})
	dataflowTester.FlushTabler(&ticket.IssueAssignee{})
	dataflowTester.Subtask(tasks.ConvertTaskMeta, taskData)

	dataflowTester.VerifyTable(
		ticket.Issue{},
		"./snapshot_tables/issues.csv",
		[]string{
			"id",
			"url",
			"issue_key",
			"title",
			"description",
			"type",
			"original_type",
			"status",
			"original_status",
			"story_point",
			"resolution_date",
			"created_date",
			"updated_date",
			"lead_time_minutes",
			"parent_issue_id",
			"creator_id",
			"creator_name",
			"assignee_id",
			"assignee_name",
			"due_date",
		},
	)

	dataflowTester.VerifyTable(
		ticket.BoardIssue{},
		"./snapshot_tables/board_issues.csv",
		[]string{"board_id", "issue_id"},
	)

	dataflowTester.VerifyTableWithOptions(ticket.IssueAssignee{}, e2ehelper.TableOptions{
		CSVRelPath:  "./snapshot_tables/issue_assignees.csv",
		IgnoreTypes: []interface{}{common.NoPKModel{}},
	})
}

func TestAsanaTaskWithTypeMapping(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	// Test with scope config that has issue type mappings
	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId:  1,
			ProjectId:     "1234567890",
			ScopeConfigId: 1,
		},
	}

	// Import raw data and tool layer data
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_tasks_with_tags.csv", "_raw_asana_tasks")
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_projects.csv", &models.AsanaProject{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_scope_configs_with_mappings.csv", &models.AsanaScopeConfig{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_tags.csv", &models.AsanaTag{})
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_task_tags_with_types.csv", &models.AsanaTaskTag{})

	// Extract and convert
	dataflowTester.FlushTabler(&models.AsanaTask{})
	dataflowTester.Subtask(tasks.ExtractTaskMeta, taskData)

	dataflowTester.FlushTabler(&ticket.Issue{})
	dataflowTester.FlushTabler(&ticket.BoardIssue{})
	dataflowTester.Subtask(tasks.ConvertTaskMeta, taskData)

	// Verify issues have correct type based on tag matching
	dataflowTester.VerifyTable(
		ticket.Issue{},
		"./snapshot_tables/issues_with_type_mapping.csv",
		[]string{
			"id",
			"type",
			"original_type",
			"status",
		},
	)
}

func TestAsanaSubtaskConversion(t *testing.T) {
	var asana impl.Asana
	dataflowTester := e2ehelper.NewDataFlowTester(t, "asana", asana)

	taskData := &tasks.AsanaTaskData{
		Options: &tasks.AsanaOptions{
			ConnectionId:  1,
			ProjectId:     "1234567890",
			ScopeConfigId: 0, // No scope config
		},
	}

	// Import subtask data
	dataflowTester.ImportCsvIntoRawTable("./raw_tables/_raw_asana_subtasks.csv", "_raw_asana_tasks")
	dataflowTester.ImportCsvIntoTabler("./snapshot_tables/_tool_asana_projects.csv", &models.AsanaProject{})

	// Extract
	dataflowTester.FlushTabler(&models.AsanaTask{})
	dataflowTester.Subtask(tasks.ExtractTaskMeta, taskData)

	// Convert
	dataflowTester.FlushTabler(&ticket.Issue{})
	dataflowTester.FlushTabler(&ticket.BoardIssue{})
	dataflowTester.Subtask(tasks.ConvertTaskMeta, taskData)

	// Verify subtasks have correct parent_issue_id and type=SUBTASK
	dataflowTester.VerifyTable(
		ticket.Issue{},
		"./snapshot_tables/issues_subtasks.csv",
		[]string{
			"id",
			"type",
			"parent_issue_id",
		},
	)
}
