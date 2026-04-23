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
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

const planeEpicPageSize = 100

type planeApiEpic struct {
	Id                  string             `json:"id"`
	SequenceId          int                `json:"sequence_id"`
	Name                string             `json:"name"`
	DescriptionStripped string             `json:"description_stripped"`
	Type                string             `json:"type"`
	State               string             `json:"state"`
	Priority            string             `json:"priority"`
	Assignees           []planeApiAssignee `json:"assignees"`
	EstimatePoint       planeApiFloat64    `json:"estimate_point"`
	// Plane may expose both a floating estimate and a legacy integer point value.
	Point       *int       `json:"point"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
	StartDate   string     `json:"start_date"`
	TargetDate  string     `json:"target_date"`
	Parent      *string    `json:"parent"`
}

type planeApiEpicUpdateMarker struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

type planeEpicListResponse struct {
	Count      *int              `json:"count"`
	TotalCount *int              `json:"total_count"`
	NextOffset *int              `json:"next_offset"`
	Results    []json.RawMessage `json:"results"`
	Data       []json.RawMessage `json:"data"`
}

func parsePlaneEpicResults(response *http.Response) ([]json.RawMessage, errors.Error) {
	page, err := parsePlaneEpicPage(response)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func parsePlaneEpicResultsForCollector(response *http.Response, since *time.Time) ([]json.RawMessage, errors.Error) {
	results, err := parsePlaneEpicResults(response)
	if err != nil {
		return nil, err
	}
	if since == nil {
		return results, nil
	}
	return filterPlaneEpicsByUpdatedAt(results, since)
}

func filterPlaneEpicsByUpdatedAt(results []json.RawMessage, since *time.Time) ([]json.RawMessage, errors.Error) {
	if since == nil {
		return results, nil
	}

	filtered := make([]json.RawMessage, 0, len(results))
	for _, result := range results {
		var marker planeApiEpicUpdateMarker
		if err := json.Unmarshal(result, &marker); err != nil {
			return nil, errors.Default.Wrap(err, "error unmarshalling Plane epic updated_at marker")
		}
		if marker.UpdatedAt == nil || !marker.UpdatedAt.Before(*since) {
			filtered = append(filtered, result)
		}
	}
	return filtered, nil
}

func parsePlaneEpicNextOffset(response *http.Response, currentOffset, pageSize int) (interface{}, errors.Error) {
	page, err := parsePlaneEpicPage(response)
	if err != nil {
		return nil, err
	}
	if page.NextOffset != nil {
		return *page.NextOffset, nil
	}
	if page.TotalCount != nil && currentOffset+len(page.Results) < *page.TotalCount {
		return currentOffset + pageSize, nil
	}
	if page.Count != nil && currentOffset+len(page.Results) < *page.Count {
		return currentOffset + pageSize, nil
	}
	if len(page.Results) < pageSize {
		return nil, nil
	}
	return currentOffset + pageSize, nil
}

func parsePlaneEpicPage(response *http.Response) (*planeEpicListResponse, errors.Error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error reading Plane epic response body")
	}

	var paged planeEpicListResponse
	if err := json.Unmarshal(body, &paged); err == nil && (paged.Results != nil || paged.Data != nil || paged.TotalCount != nil || paged.Count != nil || paged.NextOffset != nil) {
		// Some Plane epic responses place list items under `data` instead of `results`.
		if paged.Results == nil && paged.Data != nil {
			paged.Results = paged.Data
		}
		return &paged, nil
	}

	var rawResults []json.RawMessage
	if err := json.Unmarshal(body, &rawResults); err != nil {
		if strings.TrimSpace(string(body)) == "{}" {
			return &planeEpicListResponse{Results: []json.RawMessage{}}, nil
		}
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane epic list response")
	}
	return &planeEpicListResponse{
		Results: rawResults,
	}, nil
}

func mapPlaneEpic(
	apiEpic *planeApiEpic,
	connectionId uint64,
	projectId string,
	states map[string]models.PlaneState,
	workItemTypes map[string]models.PlaneWorkItemType,
	estimateMap map[string]*float64,
) (*models.PlaneEpic, errors.Error) {
	epic := &models.PlaneEpic{
		ConnectionId:  connectionId,
		ProjectId:     projectId,
		EpicId:        apiEpic.Id,
		SequenceId:    apiEpic.SequenceId,
		Name:          apiEpic.Name,
		Description:   apiEpic.DescriptionStripped,
		TypeId:        apiEpic.Type,
		StateId:       apiEpic.State,
		Priority:      apiEpic.Priority,
		EstimatePoint: resolvePlaneEstimatePoint(apiEpic.EstimatePoint, estimateMap),
		Point:         apiEpic.Point,
		CreatedDate:   apiEpic.CreatedAt,
		UpdatedDate:   apiEpic.UpdatedAt,
		CompletedAt:   apiEpic.CompletedAt,
		ParentId:      apiEpic.Parent,
	}
	startDate, dueDate, err := applyPlaneDates(apiEpic.StartDate, apiEpic.TargetDate)
	if err != nil {
		return nil, err
	}
	epic.StartDate = startDate
	epic.DueDate = dueDate
	if len(apiEpic.Assignees) > 0 {
		epic.AssigneeId = apiEpic.Assignees[0].Id
		epic.AssigneeName = apiEpic.Assignees[0].Name
	}
	if state, ok := states[apiEpic.State]; ok {
		epic.StateName = state.Name
		epic.StateGroup = state.Group
		epic.IsClosed = state.Group == "completed" || state.Group == "cancelled"
	}
	if workItemType, ok := workItemTypes[apiEpic.Type]; ok {
		epic.TypeName = workItemType.Name
	}
	return epic, nil
}

func buildPlaneEpicURL(endpoint, workspaceSlug, projectIdentifier string, sequenceId int) string {
	base := strings.TrimRight(endpoint, "/")
	if parsed, err := neturl.Parse(base); err == nil {
		if parsed.Host == planeHostAPI {
			parsed.Host = planeHostApp
			base = strings.TrimRight(parsed.String(), "/")
		}
	}
	identifier := fmt.Sprintf("%s-%d", projectIdentifier, sequenceId)
	return base + "/" + neturl.PathEscape(workspaceSlug) + "/epics/" + neturl.PathEscape(identifier)
}

func planeEpicStoryPoint(epic *models.PlaneEpic) *float64 {
	if epic.EstimatePoint != nil {
		return epic.EstimatePoint
	}
	if epic.Point == nil {
		return nil
	}
	value := float64(*epic.Point)
	return &value
}

func loadPlaneEpicIDSet(db dal.Dal, connectionId uint64, projectId string) (map[string]struct{}, errors.Error) {
	type planeEpicIdentity struct {
		EpicId string
	}

	var epics []planeEpicIdentity
	if err := db.All(
		&epics,
		dal.Select("epic_id"),
		dal.From(&models.PlaneEpic{}),
		dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId),
	); err != nil {
		return nil, err
	}
	epicIDSet := make(map[string]struct{}, len(epics))
	for _, epic := range epics {
		epicIDSet[epic.EpicId] = struct{}{}
	}
	return epicIDSet, nil
}
