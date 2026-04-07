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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildHubspotSearchRequestBody_WithFiltersAndAfter(t *testing.T) {
	since := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	body := buildHubspotSearchRequestBody(&since, &until, 100, "cursor-1")

	assert.Equal(t, 100, body["limit"])
	assert.Equal(t, "cursor-1", body["after"])

	filterGroups, ok := body["filterGroups"].([]map[string]interface{})
	if assert.True(t, ok) && assert.Len(t, filterGroups, 1) {
		filters, ok := filterGroups[0]["filters"].([]map[string]interface{})
		if assert.True(t, ok) && assert.Len(t, filters, 2) {
			assert.Equal(t, "GTE", filters[0]["operator"])
			assert.Equal(t, "LTE", filters[1]["operator"])
		}
	}

	properties, ok := body["properties"].([]string)
	if assert.True(t, ok) {
		assert.Contains(t, properties, "hubspot_owner_email")
		assert.Contains(t, properties, "hs_email_sender_email")
	}
}

func TestBuildHubspotSearchRequestBody_WithoutFilters(t *testing.T) {
	body := buildHubspotSearchRequestBody(nil, nil, 50, nil)
	_, hasFilterGroups := body["filterGroups"]
	_, hasAfter := body["after"]

	assert.False(t, hasFilterGroups)
	assert.False(t, hasAfter)
	assert.Equal(t, 50, body["limit"])
}

func TestParseHubspotNextAfter(t *testing.T) {
	after, err := parseHubspotNextAfter([]byte(`{"paging":{"next":{"after":"abc"}}}`))
	assert.NoError(t, err)
	assert.Equal(t, "abc", after)

	after, err = parseHubspotNextAfter([]byte(`{"paging":{"next":{"after":""}}}`))
	assert.NoError(t, err)
	assert.Equal(t, "", after)
}

func TestParseHubspotSearchResponse(t *testing.T) {
	results, err := parseHubspotSearchResponse([]byte(`{"results":[{"id":"1"},{"id":"2"}]}`))
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestParseHubspotCollectorHelpers_InvalidJson(t *testing.T) {
	_, err := parseHubspotNextAfter([]byte("{"))
	assert.Error(t, err)

	_, err = parseHubspotSearchResponse([]byte("{"))
	assert.Error(t, err)
}
