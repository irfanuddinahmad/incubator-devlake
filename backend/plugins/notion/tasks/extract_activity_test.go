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

func TestBuildNotionActivityEvent_MapsFieldsAndDefaults(t *testing.T) {
	row := []byte(`{
		"id": "page-1",
		"object": "",
		"last_edited_time": "2026-04-06T09:02:00Z",
		"last_edited_by": {
			"id": " user-1 ",
			"person": {
				"email": " editor@example.com "
			}
		}
	}`)

	event, err := buildNotionActivityEvent(row, 11, "scope-1")
	if assert.NoError(t, err) && assert.NotNil(t, event) {
		expectedTime := time.Date(2026, 4, 6, 9, 2, 0, 0, time.UTC)
		assert.Equal(t, uint64(11), event.ConnectionId)
		assert.Equal(t, "scope-1", event.ScopeId)
		assert.Equal(t, "page-1:1775466120000", event.EventId)
		assert.Equal(t, expectedTime, event.OccurredAt)
		assert.Equal(t, "user-1", event.EditorUserId)
		assert.Equal(t, "editor@example.com", event.EditorUserEmail)
		assert.Equal(t, "edited", event.ActionType)
		assert.Equal(t, "page", event.ObjectType)
		assert.Equal(t, "page-1", event.ObjectId)
		assert.Equal(t, "notion_data_source_page", event.SourceObjectType)
		assert.Equal(t, string(row), event.RawData)
	}
}

func TestBuildNotionActivityEvent_CreatedWhenTimestampsMatch(t *testing.T) {
	row := []byte(`{
		"id": "page-2",
		"object": "page",
		"created_time": "2026-04-06T09:02:00Z",
		"last_edited_time": "2026-04-06T09:02:00Z",
		"created_by": {
			"id": "creator-1",
			"person": {
				"email": "creator@example.com"
			}
		},
		"last_edited_by": {
			"id": "editor-1",
			"person": {
				"email": "editor@example.com"
			}
		}
	}`)

	event, err := buildNotionActivityEvent(row, 11, "scope-1")
	if assert.NoError(t, err) && assert.NotNil(t, event) {
		assert.Equal(t, "created", event.ActionType)
		assert.Equal(t, "creator-1", event.EditorUserId)
		assert.Equal(t, "creator@example.com", event.EditorUserEmail)
	}
}

func TestBuildNotionActivityEvent_EditedWhenTimestampsDiffer(t *testing.T) {
	row := []byte(`{
		"id": "page-3",
		"object": "page",
		"created_time": "2026-04-01T08:00:00Z",
		"last_edited_time": "2026-04-06T09:02:00Z",
		"created_by": {
			"id": "creator-1",
			"person": {"email": "creator@example.com"}
		},
		"last_edited_by": {
			"id": "editor-1",
			"person": {"email": "editor@example.com"}
		}
	}`)

	event, err := buildNotionActivityEvent(row, 11, "scope-1")
	if assert.NoError(t, err) && assert.NotNil(t, event) {
		assert.Equal(t, "edited", event.ActionType)
		assert.Equal(t, "editor-1", event.EditorUserId)
		assert.Equal(t, "editor@example.com", event.EditorUserEmail)
	}
}

func TestBuildNotionActivityEvent_SkipsEmptyId(t *testing.T) {
	row := []byte(`{
		"id": "   ",
		"object": "page",
		"last_edited_time": "2026-04-06T09:02:00Z"
	}`)

	event, err := buildNotionActivityEvent(row, 11, "scope-1")
	assert.NoError(t, err)
	assert.Nil(t, event)
}

func TestBuildNotionActivityEvent_InvalidTimeReturnsError(t *testing.T) {
	row := []byte(`{
		"id": "page-2",
		"object": "page",
		"last_edited_time": "not-a-time"
	}`)

	event, err := buildNotionActivityEvent(row, 11, "scope-1")
	assert.Nil(t, event)
	assert.Error(t, err)
}

func TestBuildNotionActivityEvent_InvalidJsonReturnsError(t *testing.T) {
	event, err := buildNotionActivityEvent([]byte("{"), 11, "scope-1")
	assert.Nil(t, event)
	assert.Error(t, err)
}
