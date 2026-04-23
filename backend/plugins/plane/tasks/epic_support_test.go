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

	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlaneEpicResultsFromArray(t *testing.T) {
	response := &http.Response{
		Body: io.NopCloser(strings.NewReader(`[{"id":"epic-1"},{"id":"epic-2"}]`)),
	}

	results, err := parsePlaneEpicResults(response)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"epic-1"}`, string(results[0]))
}

func TestParsePlaneEpicResultsFromObject(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"count":   2,
		"results": []map[string]any{{"id": "epic-1"}, {"id": "epic-2"}},
	})

	results, err := parsePlaneEpicResults(response)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"epic-2"}`, string(results[1]))
}

func TestParsePlaneEpicResultsFromDataFallback(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"count": 2,
		"data":  []map[string]any{{"id": "epic-1"}, {"id": "epic-2"}},
	})

	results, err := parsePlaneEpicResults(response)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"epic-1"}`, string(results[0]))
}

func TestParsePlaneEpicResultsFromEmptyObject(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{})

	results, err := parsePlaneEpicResults(response)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParsePlaneEpicResultsForCollectorIncremental(t *testing.T) {
	since := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	response := planePaginatedResponse(t, map[string]any{
		"results": []map[string]any{
			{"id": "epic-new", "updated_at": "2024-01-10T12:05:00Z"},
			{"id": "epic-equal", "updated_at": "2024-01-10T12:00:00Z"},
			{"id": "epic-old", "updated_at": "2024-01-10T11:59:59Z"},
		},
	})

	results, err := parsePlaneEpicResultsForCollector(response, since)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.JSONEq(t, `{"id":"epic-new","updated_at":"2024-01-10T12:05:00Z"}`, string(results[0]))
	assert.JSONEq(t, `{"id":"epic-equal","updated_at":"2024-01-10T12:00:00Z"}`, string(results[1]))
}

func TestParsePlaneEpicNextOffset(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"total_count": 250,
		"results":     []map[string]any{{"id": "epic-1"}},
	})
	nextOffset, err := parsePlaneEpicNextOffset(response, 100, 100)
	require.NoError(t, err)
	assert.Equal(t, 200, nextOffset)

	response = planePaginatedResponse(t, map[string]any{
		"next_offset": 300,
		"results":     []map[string]any{{"id": "epic-1"}},
	})
	nextOffset, err = parsePlaneEpicNextOffset(response, 200, 100)
	require.NoError(t, err)
	assert.Equal(t, 300, nextOffset)

	response = planePaginatedResponse(t, map[string]any{
		"results": []map[string]any{{"id": "epic-1"}},
	})
	nextOffset, err = parsePlaneEpicNextOffset(response, 0, 100)
	require.NoError(t, err)
	assert.Nil(t, nextOffset)
}

func TestMapPlaneEpic_AssigneeResolvedFieldsAndIsClosed(t *testing.T) {
	createdAt := mustParsePlaneTime(t, "2024-01-10T12:00:00Z")
	updatedAt := mustParsePlaneTime(t, "2024-01-11T12:00:00Z")
	completedAt := mustParsePlaneTime(t, "2024-01-12T12:30:00Z")

	epic, err := mapPlaneEpic(
		&planeApiEpic{
			Id:                  "epic-1",
			SequenceId:          12,
			Name:                "Release v1",
			DescriptionStripped: "Launch milestone",
			Type:                "type-feature",
			State:               "state-done",
			Priority:            "high",
			Assignees: []planeApiAssignee{
				{Id: "user-1", Name: "Alice"},
				{Id: "user-2", Name: "Bob"},
			},
			EstimatePoint: planeTestApiFloat64(8),
			Point:         planeTestIntPtr(13),
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
			CompletedAt:   completedAt,
			StartDate:     "2024-01-09",
			TargetDate:    "2024-01-15",
			Parent:        planeTestStringPtr("epic-parent"),
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
			"type-feature": {
				TypeId: "type-feature",
				Name:   "Feature",
			},
		},
		map[string]*float64{},
	)
	require.NoError(t, err)
	require.NotNil(t, epic)

	assert.Equal(t, uint64(7), epic.ConnectionId)
	assert.Equal(t, "project-1", epic.ProjectId)
	assert.Equal(t, "epic-1", epic.EpicId)
	assert.Equal(t, "Launch milestone", epic.Description)
	assert.Equal(t, "user-1", epic.AssigneeId)
	assert.Equal(t, "Alice", epic.AssigneeName)
	assert.Equal(t, "Done", epic.StateName)
	assert.Equal(t, "completed", epic.StateGroup)
	assert.Equal(t, "Feature", epic.TypeName)
	assert.True(t, epic.IsClosed)
	require.NotNil(t, epic.EstimatePoint)
	assert.Equal(t, 8.0, *epic.EstimatePoint)
	require.NotNil(t, epic.Point)
	assert.Equal(t, 13, *epic.Point)
}

func TestPlaneApiEpicEstimatePointAcceptsString(t *testing.T) {
	var epic planeApiEpic

	err := json.Unmarshal([]byte(`{
		"id":"epic-1",
		"estimate_point":"3.5"
	}`), &epic)

	require.NoError(t, err)
	require.NotNil(t, epic.EstimatePoint.Float64Ptr())
	assert.Equal(t, 3.5, *epic.EstimatePoint.Float64Ptr())
}

func TestMapPlaneEpicResolvesEstimateUUID(t *testing.T) {
	epic, err := mapPlaneEpic(
		&planeApiEpic{
			Id:            "epic-1",
			EstimatePoint: planeApiFloat64{rawString: "point-1"},
		},
		7,
		"project-1",
		map[string]models.PlaneState{},
		map[string]models.PlaneWorkItemType{},
		map[string]*float64{
			"point-1": planeTestFloat64Ptr(13),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, epic)
	require.NotNil(t, epic.EstimatePoint)
	assert.Equal(t, 13.0, *epic.EstimatePoint)
}

func TestMapPlaneEpicFallsBackToLegacyPointWhenEstimateUUIDUnknown(t *testing.T) {
	epic, err := mapPlaneEpic(
		&planeApiEpic{
			Id:            "epic-1",
			EstimatePoint: planeApiFloat64{rawString: "missing"},
			Point:         planeTestIntPtr(21),
		},
		7,
		"project-1",
		map[string]models.PlaneState{},
		map[string]models.PlaneWorkItemType{},
		map[string]*float64{},
	)
	require.NoError(t, err)
	require.NotNil(t, epic)
	assert.Nil(t, epic.EstimatePoint)
	require.NotNil(t, planeEpicStoryPoint(epic))
	assert.Equal(t, 21.0, *planeEpicStoryPoint(epic))
}

func TestPlaneEpicStoryPointFallback(t *testing.T) {
	estimate := 5.0
	epic := &models.PlaneEpic{
		EstimatePoint: &estimate,
		Point:         planeTestIntPtr(8),
	}
	require.NotNil(t, planeEpicStoryPoint(epic))
	assert.Equal(t, 5.0, *planeEpicStoryPoint(epic))

	epic.EstimatePoint = nil
	require.NotNil(t, planeEpicStoryPoint(epic))
	assert.Equal(t, 8.0, *planeEpicStoryPoint(epic))

	epic.Point = nil
	assert.Nil(t, planeEpicStoryPoint(epic))
}

func TestBuildPlaneEpicURL(t *testing.T) {
	assert.Equal(
		t,
		"https://app.plane.so/workspace-a/epics/PROJ-42",
		buildPlaneEpicURL("https://api.plane.so/", "workspace-a", "PROJ", 42),
	)

	assert.Equal(
		t,
		"https://plane.example.com/workspace%2Fa/epics/PROJ-42",
		buildPlaneEpicURL("https://plane.example.com/", "workspace/a", "PROJ", 42),
	)
}

func TestResolvePlaneParentIssueId(t *testing.T) {
	workItemIdGen := didgen.NewDomainIdGenerator(&models.PlaneWorkItem{})
	epicIdGen := didgen.NewDomainIdGenerator(&models.PlaneEpic{})
	parentId := planeTestStringPtr("parent-1")
	epicIDSet := map[string]struct{}{
		"epic-1": {},
	}

	assert.Equal(
		t,
		workItemIdGen.Generate(uint64(1), "project-1", "parent-1"),
		resolvePlaneParentIssueId(1, "project-1", parentId, epicIDSet, workItemIdGen, epicIdGen),
	)

	parentId = planeTestStringPtr("epic-1")
	assert.Equal(
		t,
		epicIdGen.Generate(uint64(1), "project-1", "epic-1"),
		resolvePlaneParentIssueId(1, "project-1", parentId, epicIDSet, workItemIdGen, epicIdGen),
	)

	assert.Empty(t, resolvePlaneParentIssueId(1, "project-1", nil, epicIDSet, workItemIdGen, epicIdGen))
}

func TestResolvePlaneParentIssueIdEmptyString(t *testing.T) {
	workItemIdGen := didgen.NewDomainIdGenerator(&models.PlaneWorkItem{})
	epicIdGen := didgen.NewDomainIdGenerator(&models.PlaneEpic{})
	epicIDSet := map[string]struct{}{
		"epic-1": {},
	}
	assert.Empty(t, resolvePlaneParentIssueId(1, "project-1", planeTestStringPtr(""), epicIDSet, workItemIdGen, epicIdGen))
}

func planeTestIntPtr(value int) *int {
	return &value
}
