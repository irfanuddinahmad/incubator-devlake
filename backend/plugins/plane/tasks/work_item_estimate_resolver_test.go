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
	"testing"

	helperapi "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEstimateForRow(t *testing.T) {
	estimateMap := map[string]*float64{
		"point-1": planeTestFloat64Ptr(8),
	}

	mapped := resolveEstimateForRow([]byte(`{"estimate_point":"point-1"}`), estimateMap)
	require.NotNil(t, mapped)
	assert.Equal(t, 8.0, *(mapped.(*float64)))

	numeric := resolveEstimateForRow([]byte(`{"estimate_point":"13"}`), estimateMap)
	require.NotNil(t, numeric)
	assert.Equal(t, 13.0, *(numeric.(*float64)))

	assert.Nil(t, resolveEstimateForRow([]byte(`{"estimate_point":"unknown"}`), estimateMap))
	assert.Nil(t, resolveEstimateForRow([]byte(`{"estimate_point":null}`), estimateMap))
}

func TestCollectResolvedWorkItemEstimatesGroupsByResolvedValue(t *testing.T) {
	estimateMap := map[string]*float64{
		"point-1": planeTestFloat64Ptr(5),
	}
	rawRows := []helperapi.RawData{
		{Data: []byte(`{"id":"work-item-1","estimate_point":"point-1"}`)},
		{Data: []byte(`{"id":"work-item-2","estimate_point":"5"}`)},
		{Data: []byte(`{"id":"work-item-3","estimate_point":null}`)},
	}

	groupedIds, groupedValues, err := collectResolvedWorkItemEstimates(rawRows, estimateMap)
	require.NoError(t, err)

	require.Len(t, groupedIds["5"], 2)
	assert.ElementsMatch(t, []string{"work-item-1", "work-item-2"}, groupedIds["5"])
	require.NotNil(t, groupedValues["5"])
	assert.Equal(t, 5.0, *(groupedValues["5"].(*float64)))

	require.Len(t, groupedIds[resolvedEstimateNilKey], 1)
	assert.Equal(t, []string{"work-item-3"}, groupedIds[resolvedEstimateNilKey])
	assert.Nil(t, groupedValues[resolvedEstimateNilKey])
}

func TestCollectResolvedWorkItemEstimatesUsesLatestRawRowPerWorkItem(t *testing.T) {
	estimateMap := map[string]*float64{
		"point-new": planeTestFloat64Ptr(21),
		"point-old": planeTestFloat64Ptr(5),
	}
	rawRows := []helperapi.RawData{
		{ID: 10, Data: []byte(`{"id":"work-item-1","estimate_point":null}`)},
		{ID: 20, Data: []byte(`{"id":"work-item-1","estimate_point":"point-new"}`)},
		{ID: 30, Data: []byte(`{"id":"work-item-2","estimate_point":"point-old"}`)},
		{ID: 40, Data: []byte(`{"id":"work-item-2","estimate_point":null}`)},
	}

	groupedIds, groupedValues, err := collectResolvedWorkItemEstimates(rawRows, estimateMap)
	require.NoError(t, err)

	require.Len(t, groupedIds["21"], 1)
	assert.Equal(t, []string{"work-item-1"}, groupedIds["21"])
	assert.Equal(t, 21.0, *(groupedValues["21"].(*float64)))

	require.Len(t, groupedIds[resolvedEstimateNilKey], 1)
	assert.Equal(t, []string{"work-item-2"}, groupedIds[resolvedEstimateNilKey])
	assert.Nil(t, groupedValues[resolvedEstimateNilKey])
}
