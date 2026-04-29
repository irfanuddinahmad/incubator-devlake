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
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	helperapi "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

type planeApiEstimatePoint struct {
	Id          string `json:"id"`
	Key         int    `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type planeApiEstimate struct {
	Id     string                  `json:"id"`
	Points []planeApiEstimatePoint `json:"points"`
}

type planeEstimateListResponse struct {
	Count      *int              `json:"count"`
	TotalCount *int              `json:"total_count"`
	NextOffset *int              `json:"next_offset"`
	NextCursor string            `json:"next_cursor"`
	Results    []json.RawMessage `json:"results"`
	Data       []json.RawMessage `json:"data"`
}

const planeEstimateEnrichmentConcurrency = 5

func parsePlaneEstimatePointValue(s string) (*float64, string) {
	label := strings.TrimSpace(s)
	if label == "" {
		return nil, ""
	}
	parsed, err := strconv.ParseFloat(label, 64)
	if err != nil {
		return nil, label
	}
	return &parsed, label
}

func loadPlaneEstimatePointMap(db dal.Dal, connectionId uint64, projectId string) (map[string]*float64, errors.Error) {
	var points []models.PlaneEstimatePoint
	if err := db.All(&points, dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId)); err != nil {
		return nil, err
	}
	estimateMap := make(map[string]*float64, len(points))
	for _, point := range points {
		estimateMap[point.PointId] = point.Value
	}
	return estimateMap, nil
}

func ignoreHTTPStatus404(res *http.Response) errors.Error {
	if res.StatusCode == http.StatusUnauthorized {
		return errors.Unauthorized.New("authentication failed, please check your AccessToken")
	}
	if res.StatusCode == http.StatusNotFound {
		return helperapi.ErrIgnoreAndContinue
	}
	return nil
}

func parsePlaneEstimateResults(response *http.Response) ([]json.RawMessage, errors.Error) {
	page, err := parsePlaneEstimatePage(response)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func enrichPlaneEstimateResults(
	results []json.RawMessage,
	apiClient *helperapi.ApiAsyncClient,
	workspaceSlug string,
	projectId string,
) ([]json.RawMessage, errors.Error) {
	enriched := make([]json.RawMessage, len(results))
	semaphore := make(chan struct{}, planeEstimateEnrichmentConcurrency)
	var wg sync.WaitGroup
	var firstErr errors.Error
	var errMu sync.Mutex

	for i, raw := range results {
		enriched[i] = raw
		var apiEstimate planeApiEstimate
		if err := json.Unmarshal(raw, &apiEstimate); err != nil {
			return nil, errors.Default.Wrap(err, "error unmarshalling Plane estimate")
		}
		if len(apiEstimate.Points) == 0 && apiEstimate.Id != "" {
			wg.Add(1)
			semaphore <- struct{}{}
			go func(index int, estimate planeApiEstimate) {
				defer wg.Done()
				defer func() { <-semaphore }()
				marshalled, err := enrichSingleEstimate(apiClient, workspaceSlug, projectId, estimate)
				if err != nil {
					recordPlaneEstimateEnrichmentError(&errMu, &firstErr, err)
					return
				}
				enriched[index] = marshalled
			}(i, apiEstimate)
		}
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return enriched, nil
}

func enrichSingleEstimate(
	apiClient *helperapi.ApiAsyncClient,
	workspaceSlug string,
	projectId string,
	estimate planeApiEstimate,
) (json.RawMessage, errors.Error) {
	points, err := fetchPlaneEstimatePoints(apiClient, workspaceSlug, projectId, estimate.Id)
	if err != nil {
		return nil, err
	}
	estimate.Points = points
	marshalled, marshalErr := json.Marshal(&estimate)
	if marshalErr != nil {
		return nil, errors.Default.Wrap(marshalErr, "error marshalling Plane estimate with points")
	}
	return marshalled, nil
}

func recordPlaneEstimateEnrichmentError(errMu *sync.Mutex, firstErr *errors.Error, err errors.Error) {
	errMu.Lock()
	defer errMu.Unlock()
	if *firstErr == nil {
		*firstErr = err
	}
}

func parsePlaneEstimateNextPage(response *http.Response, currentOffset, pageSize int) (interface{}, errors.Error) {
	page, err := parsePlaneEstimatePage(response)
	if err != nil {
		return nil, err
	}
	if page.NextCursor != "" {
		return page.NextCursor, nil
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

func parsePlaneEstimatePage(response *http.Response) (*planeEstimateListResponse, errors.Error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error reading Plane estimate response body")
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return &planeEstimateListResponse{Results: []json.RawMessage{}}, nil
	}

	var paged planeEstimateListResponse
	// Detect paginated envelope; fall through to array or single-object shapes otherwise.
	if err := json.Unmarshal(body, &paged); err == nil &&
		(paged.Results != nil || paged.Data != nil || paged.TotalCount != nil || paged.Count != nil || paged.NextOffset != nil || paged.NextCursor != "") {
		if paged.Results == nil && paged.Data != nil {
			paged.Results = paged.Data
		}
		if paged.Results == nil {
			paged.Results = []json.RawMessage{}
		}
		return &paged, nil
	}

	var rawResults []json.RawMessage
	// Handle plain array responses.
	if err := json.Unmarshal(body, &rawResults); err == nil {
		return &planeEstimateListResponse{Results: rawResults}, nil
	}

	var lastErr error
	var rawObject json.RawMessage
	// Handle single estimate object responses.
	if err := json.Unmarshal(body, &rawObject); err == nil {
		return &planeEstimateListResponse{Results: []json.RawMessage{rawObject}}, nil
	} else {
		lastErr = err
	}

	if trimmed == "{}" || trimmed == "null" {
		return &planeEstimateListResponse{Results: []json.RawMessage{}}, nil
	}
	return nil, errors.Default.Wrap(lastErr, "error unmarshalling Plane estimate list response")
}

func extractPlaneEstimatePoints(data []byte, connectionId uint64, projectId string) ([]interface{}, errors.Error) {
	var apiEstimate planeApiEstimate
	if err := json.Unmarshal(data, &apiEstimate); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane estimate")
	}
	return buildPlaneEstimatePointInterfaces(apiEstimate.Id, apiEstimate.Points, connectionId, projectId), nil
}

func fetchPlaneEstimatePoints(apiClient *helperapi.ApiAsyncClient, workspaceSlug string, projectId string, estimateId string) ([]planeApiEstimatePoint, errors.Error) {
	response, err := apiClient.Get(
		"api/v1/workspaces/"+url.PathEscape(workspaceSlug)+"/projects/"+url.PathEscape(projectId)+"/estimates/"+url.PathEscape(estimateId)+"/estimate-points/",
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return []planeApiEstimatePoint{}, nil
	}
	if response.StatusCode == http.StatusUnauthorized {
		return nil, errors.Unauthorized.New("authentication failed, please check your Plane API key")
	}
	if response.StatusCode >= http.StatusBadRequest {
		return nil, errors.HttpStatus(response.StatusCode).New("error fetching Plane estimate points")
	}

	return parsePlaneEstimatePointsResponse(response)
}

func parsePlaneEstimatePointsResponse(response *http.Response) ([]planeApiEstimatePoint, errors.Error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error reading Plane estimate point response body")
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "[]" || trimmed == "{}" || trimmed == "null" {
		return []planeApiEstimatePoint{}, nil
	}

	var points []planeApiEstimatePoint
	if err := json.Unmarshal(body, &points); err == nil {
		return points, nil
	}

	var paged planeEstimateListResponse
	if err := json.Unmarshal(body, &paged); err == nil && (paged.Results != nil || paged.Data != nil) {
		rawResults := paged.Results
		if rawResults == nil {
			rawResults = paged.Data
		}
		points = make([]planeApiEstimatePoint, 0, len(rawResults))
		for _, raw := range rawResults {
			var point planeApiEstimatePoint
			if err := json.Unmarshal(raw, &point); err != nil {
				return nil, errors.Default.Wrap(err, "error unmarshalling Plane estimate point")
			}
			points = append(points, point)
		}
		return points, nil
	}

	return nil, errors.Default.New("error unmarshalling Plane estimate point response")
}

func buildPlaneEstimatePointInterfaces(
	estimateId string,
	points []planeApiEstimatePoint,
	connectionId uint64,
	projectId string,
) []interface{} {
	rows := make([]interface{}, 0, len(points))
	for _, point := range points {
		value, label := parsePlaneEstimatePointValue(point.Value)
		rows = append(rows, &models.PlaneEstimatePoint{
			ConnectionId: connectionId,
			ProjectId:    projectId,
			PointId:      point.Id,
			EstimateId:   estimateId,
			Key:          point.Key,
			Value:        value,
			ValueLabel:   label,
			Description:  point.Description,
		})
	}
	return rows
}

func extractPlaneRawEstimatePointValue(data []byte) string {
	var payload struct {
		EstimatePoint json.RawMessage `json:"estimate_point"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.EstimatePoint) == 0 {
		return ""
	}

	var stringValue string
	if err := json.Unmarshal(payload.EstimatePoint, &stringValue); err == nil {
		return strings.TrimSpace(stringValue)
	}

	var numberValue float64
	if err := json.Unmarshal(payload.EstimatePoint, &numberValue); err == nil {
		return strconv.FormatFloat(numberValue, 'f', -1, 64)
	}

	return ""
}
