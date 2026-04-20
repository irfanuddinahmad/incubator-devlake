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

package tasks

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlanePaginatedResultsAndCursor(t *testing.T) {
	response := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{
			"next_cursor": "100:1:0",
			"results": [
				{"id":"item-1"},
				{"id":"item-2"}
			]
		}`)),
	}

	results, err := parsePlanePaginatedResults(response)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"item-1"}`, string(results[0]))

	response.Body = io.NopCloser(strings.NewReader(`{
		"next_cursor": "100:1:0",
		"results": [{"id":"item-1"}]
	}`))
	cursor, err := parsePlaneNextCursor(response)
	require.NoError(t, err)
	assert.Equal(t, "100:1:0", cursor)

	response.Body = io.NopCloser(strings.NewReader(`{
		"next_cursor": "",
		"results": [{"id":"item-1"}]
	}`))
	cursor, err = parsePlaneNextCursor(response)
	require.NoError(t, err)
	assert.Nil(t, cursor)
}

func TestExtractPlaneWorkItem_AssigneeAndResolvedFields(t *testing.T) {
	createdAt := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	updatedAt := mustParsePlaneTime(t, "2024-01-11T12:00:00Z")
	completedAt := mustParsePlaneTime(t, "2024-01-12T12:30:00Z")

	workItem, err := extractPlaneWorkItem(
		[]byte(`{
			"id": "work-item-1",
			"sequence_id": 42,
			"name": "Ship Phase 4",
			"description_stripped": "Implement work item sync",
			"type": "type-bug",
			"state": "state-done",
			"priority": "high",
			"assignees": ["user-1", "user-2"],
			"estimate_point": 5,
			"created_at": "2024-01-10T12:00:00Z",
			"updated_at": "2024-01-11T12:00:00Z",
			"completed_at": "2024-01-12T12:30:00Z",
			"start_date": "2024-01-09",
			"target_date": "2024-01-15",
			"parent": "parent-1"
		}`),
		7,
		"project-1",
		map[string]models.PlaneState{
			"state-done": {
				StateId: "state-done",
				Name:    "Done",
				Group:   "completed",
			},
		},
		map[string]models.PlaneWorkItemType{
			"type-bug": {
				TypeId: "type-bug",
				Name:   "Bug",
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, workItem)

	assert.Equal(t, uint64(7), workItem.ConnectionId)
	assert.Equal(t, "project-1", workItem.ProjectId)
	assert.Equal(t, "work-item-1", workItem.WorkItemId)
	assert.Equal(t, 42, workItem.SequenceId)
	assert.Equal(t, "Ship Phase 4", workItem.Title)
	assert.Equal(t, "Implement work item sync", workItem.Description)
	assert.Equal(t, "user-1", workItem.AssigneeId)
	assert.Equal(t, "", workItem.AssigneeName)
	assert.Equal(t, "Done", workItem.StateName)
	assert.Equal(t, "completed", workItem.StateGroup)
	assert.Equal(t, "Bug", workItem.TypeName)
	assert.Equal(t, createdAt, workItem.CreatedDate)
	assert.Equal(t, updatedAt, workItem.UpdatedDate)
	assert.Equal(t, completedAt, workItem.CompletedAt)
	require.NotNil(t, workItem.ParentId)
	assert.Equal(t, "parent-1", *workItem.ParentId)
	require.NotNil(t, workItem.EstimatePoint)
	assert.Equal(t, 5.0, *workItem.EstimatePoint)
	assert.True(t, workItem.IsClosed)
}

func TestPlaneWorkItemMappingHelpers(t *testing.T) {
	assert.Equal(t, "BUG", planeWorkItemTypeToStandardType("Bug"))
	assert.Equal(t, "REQUIREMENT", planeWorkItemTypeToStandardType("Feature"))
	assert.Equal(t, "TASK", planeWorkItemTypeToStandardType("Unknown"))

	assert.Equal(t, "TODO", planeStateGroupToStandardStatus("backlog"))
	assert.Equal(t, "IN_PROGRESS", planeStateGroupToStandardStatus("started"))
	assert.Equal(t, "TODO", planeStateGroupToStandardStatus("something-unexpected"))
}

func TestComputePlaneLeadTimeMinutes(t *testing.T) {
	createdAt := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	completedAt := mustParsePlaneTime(t, "2024-01-10T13:45:00Z")

	leadTime := computePlaneLeadTimeMinutes(createdAt, completedAt)
	require.NotNil(t, leadTime)
	assert.Equal(t, uint(105), *leadTime)

	assert.Nil(t, computePlaneLeadTimeMinutes(createdAt, nil))
	assert.Nil(t, computePlaneLeadTimeMinutes(completedAt, createdAt))
}

func TestBuildPlaneWorkItemURL(t *testing.T) {
	assert.Equal(
		t,
		"https://app.plane.so/workspace-a/work-items/PROJ-42",
		buildPlaneWorkItemURL("https://api.plane.so/", "workspace-a", "PROJ", 42),
	)

	assert.Equal(
		t,
		"https://plane.example.com/workspace%2Fa/work-items/PROJ-42",
		buildPlaneWorkItemURL("https://plane.example.com/", "workspace/a", "PROJ", 42),
	)
}

func mustParsePlaneTime(t *testing.T, value string) *time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return &parsed
}
