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

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlaneEstimatePointValue(t *testing.T) {
	value, label := parsePlaneEstimatePointValue(" 5 ")
	require.NotNil(t, value)
	assert.Equal(t, 5.0, *value)
	assert.Equal(t, "5", label)

	value, label = parsePlaneEstimatePointValue("M")
	assert.Nil(t, value)
	assert.Equal(t, "M", label)

	value, label = parsePlaneEstimatePointValue("   ")
	assert.Nil(t, value)
	assert.Equal(t, "", label)
}

func TestParsePlaneEstimateResults(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"results": []map[string]any{{"id": "estimate-1"}},
	})
	results, err := parsePlaneEstimateResults(response)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.JSONEq(t, `{"id":"estimate-1"}`, string(results[0]))

	response = planePaginatedResponse(t, map[string]any{
		"data": []map[string]any{{"id": "estimate-2"}},
	})
	results, err = parsePlaneEstimateResults(response)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.JSONEq(t, `{"id":"estimate-2"}`, string(results[0]))

	response = &http.Response{Body: io.NopCloser(strings.NewReader(`[{"id":"estimate-3"}]`))}
	results, err = parsePlaneEstimateResults(response)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.JSONEq(t, `{"id":"estimate-3"}`, string(results[0]))

	response = &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":"estimate-4"}`))}
	results, err = parsePlaneEstimateResults(response)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.JSONEq(t, `{"id":"estimate-4"}`, string(results[0]))

	response = &http.Response{Body: io.NopCloser(strings.NewReader(``))}
	results, err = parsePlaneEstimateResults(response)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestParsePlaneEstimatePageReturnsUnmarshalError(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader(`{"id":`))}

	page, err := parsePlaneEstimatePage(response)
	assert.Nil(t, page)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshalling Plane estimate list response")
}

func TestParsePlaneEstimateNextPage(t *testing.T) {
	response := planePaginatedResponse(t, map[string]any{
		"next_cursor": "cursor-1",
		"results":     []map[string]any{{"id": "estimate-1"}},
	})
	nextPage, err := parsePlaneEstimateNextPage(response, 0, planeEstimatePageSize)
	require.NoError(t, err)
	assert.Equal(t, "cursor-1", nextPage)

	response = planePaginatedResponse(t, map[string]any{
		"next_offset": 200,
		"results":     []map[string]any{{"id": "estimate-1"}},
	})
	nextPage, err = parsePlaneEstimateNextPage(response, 100, planeEstimatePageSize)
	require.NoError(t, err)
	assert.Equal(t, 200, nextPage)

	response = planePaginatedResponse(t, map[string]any{
		"total_count": 201,
		"results":     make([]map[string]any, planeEstimatePageSize),
	})
	nextPage, err = parsePlaneEstimateNextPage(response, 100, planeEstimatePageSize)
	require.NoError(t, err)
	assert.Equal(t, 200, nextPage)

	response = planePaginatedResponse(t, map[string]any{
		"results": []map[string]any{{"id": "estimate-1"}},
	})
	nextPage, err = parsePlaneEstimateNextPage(response, 0, planeEstimatePageSize)
	require.NoError(t, err)
	assert.Nil(t, nextPage)
}

func TestIgnoreHTTPStatus404(t *testing.T) {
	assert.Equal(t, ignoreHTTPStatus404(&http.Response{StatusCode: http.StatusNotFound}).Error(), "ignore and continue")

	err := ignoreHTTPStatus404(&http.Response{StatusCode: http.StatusUnauthorized})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")

	assert.NoError(t, ignoreHTTPStatus404(&http.Response{StatusCode: http.StatusOK}))
}

func TestExtractPlaneEstimatePoints(t *testing.T) {
	rows, err := extractPlaneEstimatePoints([]byte(`{
		"id":"estimate-1",
		"points":[
			{"id":"point-1","key":0,"value":"5","description":"Five"},
			{"id":"point-2","key":1,"value":"M","description":"Medium"}
		]
	}`), 7, "project-1")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	point1 := rows[0].(*models.PlaneEstimatePoint)
	require.NotNil(t, point1.Value)
	assert.Equal(t, 5.0, *point1.Value)
	assert.Equal(t, "5", point1.ValueLabel)
	assert.Equal(t, "estimate-1", point1.EstimateId)

	point2 := rows[1].(*models.PlaneEstimatePoint)
	assert.Nil(t, point2.Value)
	assert.Equal(t, "M", point2.ValueLabel)
}

func TestExtractPlaneEstimatePointsHandlesMissingPoints(t *testing.T) {
	rows, err := extractPlaneEstimatePoints([]byte(`{"id":"estimate-1","points":null}`), 7, "project-1")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParsePlaneEstimatePointsResponse(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader(`[
		{"id":"point-1","key":1,"value":"3","description":""},
		{"id":"point-2","key":2,"value":"M","description":"medium"}
	]`))}
	points, err := parsePlaneEstimatePointsResponse(response)
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "point-1", points[0].Id)
	assert.Equal(t, "3", points[0].Value)
	assert.Equal(t, "point-2", points[1].Id)
	assert.Equal(t, "M", points[1].Value)
}

func TestBuildPlaneEstimatePointInterfaces(t *testing.T) {
	rows, err := extractPlaneEstimatePoints([]byte(`{
		"id":"estimate-1",
		"points":[
			{"id":"point-1","key":1,"value":"5","description":""},
			{"id":"point-2","key":2,"value":"L","description":"large"}
		]
	}`), 7, "project-1")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	point1 := rows[0].(*models.PlaneEstimatePoint)
	require.NotNil(t, point1.Value)
	assert.Equal(t, 5.0, *point1.Value)
	assert.Equal(t, "5", point1.ValueLabel)

	point2 := rows[1].(*models.PlaneEstimatePoint)
	assert.Nil(t, point2.Value)
	assert.Equal(t, "L", point2.ValueLabel)
	assert.Equal(t, "estimate-1", point2.EstimateId)
}

func TestLoadPlaneEstimatePointMap(t *testing.T) {
	expected := []models.PlaneEstimatePoint{
		{PointId: "point-1", Value: planeTestFloat64Ptr(3)},
		{PointId: "point-2", Value: nil},
	}
	spy := &loadEstimatePointSpyDal{returnPoints: expected}

	estimateMap, err := loadPlaneEstimatePointMap(spy, 7, "project-1")
	require.NoError(t, err)
	require.Len(t, estimateMap, 2)
	require.NotNil(t, estimateMap["point-1"])
	assert.Equal(t, 3.0, *estimateMap["point-1"])
	assert.Nil(t, estimateMap["point-2"])
}

type loadEstimatePointSpyDal struct {
	dal.Dal
	returnPoints []models.PlaneEstimatePoint
}

func (d *loadEstimatePointSpyDal) All(out interface{}, _ ...dal.Clause) errors.Error {
	*out.(*[]models.PlaneEstimatePoint) = d.returnPoints
	return nil
}
