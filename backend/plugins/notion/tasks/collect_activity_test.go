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

func TestResolveNotionSince_Priority(t *testing.T) {
	now := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	collected := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	occurredAfter := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

	since := resolveNotionSince(&collected, &occurredAfter, now)
	if assert.NotNil(t, since) {
		assert.Equal(t, collected, *since)
	}

	since = resolveNotionSince(nil, &occurredAfter, now)
	if assert.NotNil(t, since) {
		assert.Equal(t, occurredAfter, *since)
	}

	since = resolveNotionSince(nil, nil, now)
	if assert.NotNil(t, since) {
		assert.Equal(t, now.AddDate(0, 0, -30), *since)
	}
}

func TestBuildNotionQueryRequestBody_WithUntilAndCursor(t *testing.T) {
	since := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	body := buildNotionQueryRequestBody(&since, &until, 100, "cursor-1")

	assert.Equal(t, 100, body["page_size"])
	assert.Equal(t, "cursor-1", body["start_cursor"])

	filter, ok := body["filter"].(map[string]interface{})
	if assert.True(t, ok) {
		andItems, ok := filter["and"].([]map[string]interface{})
		if assert.True(t, ok) && assert.Len(t, andItems, 2) {
			first := andItems[0]["last_edited_time"].(map[string]interface{})
			second := andItems[1]["last_edited_time"].(map[string]interface{})
			assert.Equal(t, "2026-04-06T09:00:00Z", first["on_or_after"])
			assert.Equal(t, "2026-04-06T10:00:00Z", second["on_or_before"])
		}
	}
}

func TestBuildNotionQueryRequestBody_WithoutUntil(t *testing.T) {
	since := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	body := buildNotionQueryRequestBody(&since, nil, 50, nil)

	_, hasCursor := body["start_cursor"]
	assert.False(t, hasCursor)

	filter, ok := body["filter"].(map[string]interface{})
	if assert.True(t, ok) {
		lastEdited := filter["last_edited_time"].(map[string]interface{})
		assert.Equal(t, "2026-04-06T09:00:00Z", lastEdited["on_or_after"])
	}
}

func TestParseNotionCollectorHelpers(t *testing.T) {
	next, hasMore, err := parseNotionNextCursor([]byte(`{"has_more":true,"next_cursor":"next-1"}`))
	assert.NoError(t, err)
	assert.Equal(t, "next-1", next)
	assert.True(t, hasMore)

	results, err := parseNotionQueryResponse([]byte(`{"results":[{"id":"1"}]}`))
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestParseNotionCollectorHelpers_InvalidJson(t *testing.T) {
	_, _, err := parseNotionNextCursor([]byte("{"))
	assert.Error(t, err)

	_, err = parseNotionQueryResponse([]byte("{"))
	assert.Error(t, err)
}
