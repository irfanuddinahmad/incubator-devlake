/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may obtain a copy of the License at

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
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

const (
	planeWorkItemPageSize                  = 100
	planeUpdatedAtOrderingVerificationNote = "Fallback mode stays enabled until a multi-page Plane dataset verifies order_by=-updated_at across page boundaries."

	planeStatusCancelled = "CANCELLED"

	planeHostAPI = "api.plane.so"
	planeHostApp = "app.plane.so"
)

type planePaginatedResults struct {
	NextCursor string            `json:"next_cursor"`
	Results    []json.RawMessage `json:"results"`
}

type planeApiAssignee struct {
	Id   string
	Name string
}

func (a *planeApiAssignee) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		a.Id = id
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	decodeStringField := func(keys ...string) string {
		for _, key := range keys {
			value, ok := raw[key]
			if !ok {
				continue
			}
			var decoded string
			if err := json.Unmarshal(value, &decoded); err == nil {
				return decoded
			}
		}
		return ""
	}

	a.Id = decodeStringField("id")
	a.Name = decodeStringField("display_name", "displayName", "name")
	return nil
}

type planeApiWorkItem struct {
	Id                  string             `json:"id"`
	SequenceId          int                `json:"sequence_id"`
	Name                string             `json:"name"`
	DescriptionStripped string             `json:"description_stripped"`
	Type                string             `json:"type"`
	State               string             `json:"state"`
	Priority            string             `json:"priority"`
	Assignees           []planeApiAssignee `json:"assignees"`
	EstimatePoint       planeApiFloat64    `json:"estimate_point"`
	CreatedAt           *time.Time         `json:"created_at"`
	UpdatedAt           *time.Time         `json:"updated_at"`
	CompletedAt         *time.Time         `json:"completed_at"`
	StartDate           string             `json:"start_date"`
	TargetDate          string             `json:"target_date"`
	Parent              *string            `json:"parent"`
}

type planeApiFloat64 struct {
	value     *float64
	rawString string
}

func (f planeApiFloat64) MarshalJSON() ([]byte, error) {
	if f.value == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(*f.value)
}

func (f *planeApiFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		f.value = nil
		f.rawString = ""
		return nil
	}

	var floatValue float64
	if err := json.Unmarshal(data, &floatValue); err == nil {
		f.value = &floatValue
		f.rawString = ""
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}
	if strings.TrimSpace(stringValue) == "" {
		f.value = nil
		f.rawString = ""
		return nil
	}

	trimmed := strings.TrimSpace(stringValue)
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		// Plane may return a non-numeric estimate identifier instead of a score.
		// Treat that as "no numeric estimate" instead of failing the whole extractor.
		f.value = nil
		f.rawString = trimmed
		return nil
	}
	f.value = &parsed
	f.rawString = ""
	return nil
}

func (f planeApiFloat64) Float64Ptr() *float64 {
	return f.value
}

func (f planeApiFloat64) RawString() string {
	return f.rawString
}

type planeApiState struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Group    string  `json:"group"`
	Color    string  `json:"color"`
	Sequence float64 `json:"sequence"`
}

type planeApiWorkItemType struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type planeApiWorkItemUpdateMarker struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

func parsePlanePaginatedResults(response *http.Response) ([]json.RawMessage, errors.Error) {
	var page planePaginatedResults
	if err := api.UnmarshalResponse(response, &page); err != nil {
		return nil, err
	}
	return page.Results, nil
}

func parsePlaneNextCursor(response *http.Response) (interface{}, errors.Error) {
	var page planePaginatedResults
	if err := api.UnmarshalResponse(response, &page); err != nil {
		return nil, err
	}
	if page.NextCursor == "" {
		return nil, nil
	}
	return page.NextCursor, nil
}

func parsePlaneWorkItemResultsForCollector(
	response *http.Response,
	since *time.Time,
) ([]json.RawMessage, errors.Error) {
	var page planePaginatedResults
	if err := api.UnmarshalResponse(response, &page); err != nil {
		return nil, err
	}
	if since == nil {
		return page.Results, nil
	}
	return filterPlaneWorkItemsByUpdatedAt(page.Results, since)
}

func filterPlaneWorkItemsByUpdatedAt(
	results []json.RawMessage,
	since *time.Time,
) ([]json.RawMessage, errors.Error) {
	if since == nil {
		return results, nil
	}

	filtered := make([]json.RawMessage, 0, len(results))
	for _, result := range results {
		var marker planeApiWorkItemUpdateMarker
		if err := json.Unmarshal(result, &marker); err != nil {
			return nil, errors.Default.Wrap(err, "error unmarshalling Plane work item updated_at marker")
		}
		if marker.UpdatedAt == nil || !marker.UpdatedAt.Before(*since) {
			filtered = append(filtered, result)
		}
	}
	return filtered, nil
}

func extractPlaneState(data []byte, connectionId uint64, projectId string) (*models.PlaneState, errors.Error) {
	var apiState planeApiState
	if err := json.Unmarshal(data, &apiState); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane state")
	}
	return &models.PlaneState{
		ConnectionId: connectionId,
		ProjectId:    projectId,
		StateId:      apiState.Id,
		Name:         apiState.Name,
		Group:        apiState.Group,
		Color:        apiState.Color,
		Sequence:     apiState.Sequence,
	}, nil
}

func extractPlaneWorkItemType(data []byte, connectionId uint64, projectId string) (*models.PlaneWorkItemType, errors.Error) {
	var apiType planeApiWorkItemType
	if err := json.Unmarshal(data, &apiType); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane work item type")
	}
	return &models.PlaneWorkItemType{
		ConnectionId: connectionId,
		ProjectId:    projectId,
		TypeId:       apiType.Id,
		Name:         apiType.Name,
		IsDefault:    apiType.IsDefault,
	}, nil
}

func mapPlaneWorkItem(
	apiWorkItem *planeApiWorkItem,
	connectionId uint64,
	projectId string,
	states map[string]models.PlaneState,
	workItemTypes map[string]models.PlaneWorkItemType,
	estimateMap map[string]*float64,
) (*models.PlaneWorkItem, errors.Error) {
	workItem := &models.PlaneWorkItem{
		ConnectionId:  connectionId,
		ProjectId:     projectId,
		WorkItemId:    apiWorkItem.Id,
		SequenceId:    apiWorkItem.SequenceId,
		Title:         apiWorkItem.Name,
		Description:   apiWorkItem.DescriptionStripped,
		TypeId:        apiWorkItem.Type,
		StateId:       apiWorkItem.State,
		Priority:      apiWorkItem.Priority,
		EstimatePoint: resolvePlaneEstimatePoint(apiWorkItem.EstimatePoint, estimateMap),
		CreatedDate:   apiWorkItem.CreatedAt,
		UpdatedDate:   apiWorkItem.UpdatedAt,
		CompletedAt:   apiWorkItem.CompletedAt,
		ParentId:      apiWorkItem.Parent,
	}
	startDate, dueDate, err := applyPlaneDates(apiWorkItem.StartDate, apiWorkItem.TargetDate)
	if err != nil {
		return nil, err
	}
	workItem.StartDate = startDate
	workItem.DueDate = dueDate
	if len(apiWorkItem.Assignees) > 0 {
		workItem.AssigneeId = apiWorkItem.Assignees[0].Id
		workItem.AssigneeName = apiWorkItem.Assignees[0].Name
	}
	if state, ok := states[apiWorkItem.State]; ok {
		workItem.StateName = state.Name
		workItem.StateGroup = state.Group
		workItem.IsClosed = state.Group == "completed" || state.Group == "cancelled"
	}
	if workItemType, ok := workItemTypes[apiWorkItem.Type]; ok {
		workItem.TypeName = workItemType.Name
	}
	return workItem, nil
}

func resolvePlaneEstimatePoint(estimatePoint planeApiFloat64, estimateMap map[string]*float64) *float64 {
	resolved := estimatePoint.Float64Ptr()
	if resolved == nil && estimatePoint.RawString() != "" {
		resolved = estimateMap[estimatePoint.RawString()]
	}
	return resolved
}

func applyPlaneDates(startDate string, targetDate string) (*time.Time, *time.Time, errors.Error) {
	parsedStartDate, err := parsePlaneDate(startDate)
	if err != nil {
		return nil, nil, errors.Default.Wrap(err, "error parsing Plane item start_date")
	}
	parsedTargetDate, err := parsePlaneDate(targetDate)
	if err != nil {
		return nil, nil, errors.Default.Wrap(err, "error parsing Plane item target_date")
	}
	return parsedStartDate, parsedTargetDate, nil
}

func parsePlaneDate(value string) (*time.Time, errors.Error) {
	if value == "" {
		return nil, nil
	}

	layouts := []string{
		"2006-01-02",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, errors.Default.Wrap(&time.ParseError{Layout: "2006-01-02|RFC3339Nano|RFC3339", Value: value}, "error parsing Plane date")
}

func planeStateGroupToStandardStatus(group string) string {
	switch strings.ToLower(group) {
	case "backlog", "unstarted":
		return ticket.TODO
	case "started":
		return ticket.IN_PROGRESS
	case "completed":
		return ticket.DONE
	case "cancelled":
		return planeStatusCancelled
	default:
		return ticket.TODO
	}
}

func planeWorkItemTypeToStandardType(typeName string) string {
	switch strings.ToLower(typeName) {
	case "bug":
		return ticket.BUG
	case "feature", "story", "enhancement":
		return ticket.REQUIREMENT
	case "task":
		return ticket.TASK
	default:
		return ticket.TASK
	}
}

func computePlaneLeadTimeMinutes(createdAt, completedAt *time.Time) *uint {
	if createdAt == nil || completedAt == nil || completedAt.Before(*createdAt) {
		return nil
	}
	minutes := uint(completedAt.Sub(*createdAt).Minutes())
	return &minutes
}

func buildPlaneWorkItemURL(endpoint, workspaceSlug, projectIdentifier string, sequenceId int) string {
	base := strings.TrimRight(endpoint, "/")
	if parsed, err := neturl.Parse(base); err == nil {
		if parsed.Host == planeHostAPI {
			parsed.Host = planeHostApp
			base = strings.TrimRight(parsed.String(), "/")
		}
	}
	identifier := fmt.Sprintf("%s-%d", projectIdentifier, sequenceId)
	return base + "/" + neturl.PathEscape(workspaceSlug) + "/work-items/" + neturl.PathEscape(identifier)
}
