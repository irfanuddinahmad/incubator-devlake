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

	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
)

func TestResolveHubspotSince_Priority(t *testing.T) {
	collected := time.Date(2026, 4, 6, 11, 0, 0, 0, time.UTC)
	occurredAfter := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	since := resolveHubspotSince(&collected, &occurredAfter)
	if assert.NotNil(t, since) {
		assert.Equal(t, collected, *since)
	}

	since = resolveHubspotSince(nil, &occurredAfter)
	if assert.NotNil(t, since) {
		assert.Equal(t, occurredAfter, *since)
	}

	since = resolveHubspotSince(nil, nil)
	assert.Nil(t, since)
}

func TestBuildHubspotSearchRequestBody_WithFiltersAndAfter(t *testing.T) {
	since := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	body := buildHubspotSearchRequestBody("emails", &since, &until, 100, "cursor-1")

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
	body := buildHubspotSearchRequestBody("deals", nil, nil, 50, nil)
	_, hasFilterGroups := body["filterGroups"]
	_, hasAfter := body["after"]

	assert.False(t, hasFilterGroups)
	assert.False(t, hasAfter)
	assert.Equal(t, 50, body["limit"])
}

func TestBuildHubspotSearchRequestBody_ContactsUseLastModifiedDate(t *testing.T) {
	since := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	body := buildHubspotSearchRequestBody("contacts", &since, nil, 25, nil)

	sorts, ok := body["sorts"].([]map[string]string)
	if assert.True(t, ok) && assert.Len(t, sorts, 1) {
		assert.Equal(t, "lastmodifieddate", sorts[0]["propertyName"])
	}

	filterGroups, ok := body["filterGroups"].([]map[string]interface{})
	if assert.True(t, ok) && assert.Len(t, filterGroups, 1) {
		filters, ok := filterGroups[0]["filters"].([]map[string]interface{})
		if assert.True(t, ok) && assert.Len(t, filters, 1) {
			assert.Equal(t, "lastmodifieddate", filters[0]["propertyName"])
		}
	}

	properties, ok := body["properties"].([]string)
	if assert.True(t, ok) {
		assert.Contains(t, properties, "createdate")
		assert.Contains(t, properties, "lastmodifieddate")
	}
}

func TestParseHubspotNextAfter(t *testing.T) {
	after, err := parseHubspotNextAfter([]byte(`{"paging":{"next":{"after":"abc"}}}`))
	assert.NoError(t, err)
	assert.Equal(t, "abc", after)

	after, err = parseHubspotNextAfter([]byte(`{"paging":{"next":{"after":""}}}`))
	assert.NoError(t, err)
	assert.Equal(t, "", after)
}

func TestResolveHubspotNextCustomData_Flow(t *testing.T) {
	next, err := resolveHubspotNextCustomData([]byte(`{"paging":{"next":{"after":"cursor-2"}}}`))
	if assert.NoError(t, err) {
		cursor, ok := next.(string)
		if assert.True(t, ok) {
			assert.Equal(t, "cursor-2", cursor)
			body := buildHubspotSearchRequestBody("deals", nil, nil, 100, cursor)
			assert.Equal(t, "cursor-2", body["after"])
		}
	}

	_, err = resolveHubspotNextCustomData([]byte(`{"paging":{"next":{"after":""}}}`))
	assert.ErrorIs(t, err, helper.ErrFinishCollect)
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

func TestResolveHubspotCollectionTargets_Defaults(t *testing.T) {
	targets := resolveHubspotCollectionTargets(nil)
	if assert.NotEmpty(t, targets) {
		assert.Equal(t, "leads", targets[0].ObjectType)
		assert.Equal(t, "lead", targets[0].DomainObjectType)
		assert.Equal(t, "_raw_hubspot_leads", targets[0].RawTable)
	}
}

func TestResolveHubspotCollectionTargets_FilterAndDeduplicate(t *testing.T) {
	targets := resolveHubspotCollectionTargets([]string{"deals", "", "deals", "custom", "contacts"})
	if assert.Len(t, targets, 2) {
		assert.Equal(t, "deals", targets[0].ObjectType)
		assert.Equal(t, "contacts", targets[1].ObjectType)
	}
}
