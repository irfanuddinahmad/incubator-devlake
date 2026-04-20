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
	"encoding/json"
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

	workItem, err := mapPlaneWorkItem(
		&planeApiWorkItem{
			Id:                  "work-item-1",
			SequenceId:          42,
			Name:                "Ship Phase 4",
			DescriptionStripped: "Implement work item sync",
			Type:                "type-bug",
			State:               "state-done",
			Priority:            "high",
			Assignees:           []string{"user-1", "user-2"},
			EstimatePoint:       planeTestFloat64Ptr(5),
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
			CompletedAt:         completedAt,
			StartDate:           "2024-01-09",
			TargetDate:          "2024-01-15",
			Parent:              planeTestStringPtr("parent-1"),
		},
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

func TestParsePlaneWorkItemResultsForCollectorFullRefresh(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results": []map[string]any{
			{"id": "item-1", "updated_at": "2024-01-10T12:00:00Z"},
			{"id": "item-2", "updated_at": "2024-01-09T12:00:00Z"},
		},
	})

	results, err := parsePlaneWorkItemResultsForCollector(response, nil)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"item-1","updated_at":"2024-01-10T12:00:00Z"}`, string(results[0]))
}

func TestParsePlaneWorkItemResultsForCollectorIncremental(t *testing.T) {
	since := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	response := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results": []map[string]any{
			{"id": "item-new", "updated_at": "2024-01-10T12:05:00Z"},
			{"id": "item-equal", "updated_at": "2024-01-10T12:00:00Z"},
			{"id": "item-old", "updated_at": "2024-01-10T11:59:59Z"},
		},
	})

	results, err := parsePlaneWorkItemResultsForCollector(response, since)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"item-new","updated_at":"2024-01-10T12:05:00Z"}`, string(results[0]))
	assert.JSONEq(t, `{"id":"item-equal","updated_at":"2024-01-10T12:00:00Z"}`, string(results[1]))
}

func TestParsePlaneWorkItemResultsForCollectorNoGraceWindow(t *testing.T) {
	since := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	response := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results": []map[string]any{
			{"id": "item-equal", "updated_at": "2024-01-10T12:00:00Z"},
			{"id": "item-too-old", "updated_at": "2024-01-10T11:59:59Z"},
		},
	})

	results, err := parsePlaneWorkItemResultsForCollector(response, since)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.JSONEq(t, `{"id":"item-equal","updated_at":"2024-01-10T12:00:00Z"}`, string(results[0]))
}

func TestParsePlaneWorkItemResultsForCollectorFallbackAndNilUpdatedAt(t *testing.T) {
	since := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	response := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results": []map[string]any{
			{"id": "item-old", "updated_at": "2024-01-09T12:00:00Z"},
			{"id": "item-missing"},
			{"id": "item-new", "updated_at": "2024-01-11T12:00:00Z"},
		},
	})

	results, err := parsePlaneWorkItemResultsForCollector(response, since)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"item-missing"}`, string(results[0]))
	assert.JSONEq(t, `{"id":"item-new","updated_at":"2024-01-11T12:00:00Z"}`, string(results[1]))
}

func TestParsePlaneWorkItemResultsForCollectorEmptyAndAllOlder(t *testing.T) {
	since := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")

	emptyResponse := planePaginatedResponse(t, map[string]any{
		"next_cursor": "",
		"results":     []map[string]any{},
	})
	results, err := parsePlaneWorkItemResultsForCollector(emptyResponse, since)
	require.NoError(t, err)
	assert.Empty(t, results)

	olderResponse := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results": []map[string]any{
			{"id": "item-old-1", "updated_at": "2024-01-10T11:00:00Z"},
			{"id": "item-old-2", "updated_at": "2024-01-10T10:00:00Z"},
		},
	})
	results, err = parsePlaneWorkItemResultsForCollector(olderResponse, since)
	require.NoError(t, err)
	assert.Empty(t, results)
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

func planePaginatedResponse(t *testing.T, payload map[string]any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return &http.Response{
		Body: io.NopCloser(strings.NewReader(string(body))),
	}
}

func planeTestFloat64Ptr(value float64) *float64 {
	return &value
}

func planeTestStringPtr(value string) *string {
	return &value
}
