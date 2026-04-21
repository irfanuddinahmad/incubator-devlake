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

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/notion/models"
	"github.com/stretchr/testify/assert"
)

type mockNotionPlugin struct{}

func (m mockNotionPlugin) Description() string { return "" }
func (m mockNotionPlugin) Name() string        { return "notion" }
func (m mockNotionPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/notion"
}

func init() {
	plugin.RegisterPlugin("notion", mockNotionPlugin{})
}

func testNotionIdGen() *didgen.DomainIdGenerator {
	return didgen.NewDomainIdGenerator(&models.NotionActivityEvent{})
}

func TestBuildNotionActivitiesFromEvents_DebounceAndSummary(t *testing.T) {
	base := time.Date(2026, 4, 6, 9, 2, 0, 0, time.UTC)

	events := []models.NotionActivityEvent{
		{
			NoPKModel:        common.NoPKModel{RawDataOrigin: common.RawDataOrigin{RawDataTable: "_raw_notion_data_source_pages"}},
			ConnectionId:     11,
			ScopeId:          "ds-1",
			EventId:          "n1",
			OccurredAt:       base,
			EditorUserEmail:  "editor@example.com",
			EditorUserId:     "nu-1",
			ActionType:       "Edited",
			ObjectType:       "page",
			ObjectId:         "p-1",
			SourceObjectType: "notion_data_source_page",
		},
		{
			NoPKModel:        common.NoPKModel{RawDataOrigin: common.RawDataOrigin{RawDataTable: "_raw_notion_data_source_pages"}},
			ConnectionId:     11,
			ScopeId:          "ds-1",
			EventId:          "n2",
			OccurredAt:       base.Add(time.Nanosecond),
			EditorUserEmail:  "editor@example.com",
			EditorUserId:     "nu-1",
			ActionType:       "Edited",
			ObjectType:       "page",
			ObjectId:         "p-1",
			SourceObjectType: "notion_data_source_page",
		},
	}

	calledWith := ""
	activities := buildNotionActivitiesFromEvents(events, testNotionIdGen(), func(email string) string {
		calledWith = email
		return "acc-ntn"
	}, nil)

	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "acc-ntn", a.AccountId)
		assert.Equal(t, "notion", a.SourceSystem)
		assert.Equal(t, "n2", a.SourceEventId)
		assert.Equal(t, "edited", a.ActionType)
		assert.Equal(t, "page", a.ObjectType)
		assert.Equal(t, "p-1", a.ObjectId)
		assert.Equal(t, "page:p-1", a.ObjectRef)
		assert.Equal(t, base.Add(time.Nanosecond), a.ActionTime)
		assert.Equal(t, time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC), a.ActionDay)
		assert.Equal(t, "Notion page edited x2", a.Summary)
		assert.Equal(t, "_raw_notion_data_source_pages", a.RawDataOrigin.RawDataTable)
	}
	assert.Equal(t, "editor@example.com", calledWith)
}

func TestBuildNotionActivitiesFromEvents_FallbackIdentityAndDefaults(t *testing.T) {
	event := models.NotionActivityEvent{
		ConnectionId:     12,
		ScopeId:          "ds-2",
		EventId:          "n3",
		OccurredAt:       time.Date(2026, 4, 6, 12, 40, 0, 0, time.UTC),
		EditorUserId:     "native-notion",
		ActionType:       "",
		ObjectType:       "",
		ObjectId:         "p-9",
		SourceObjectType: "page",
	}

	activities := buildNotionActivitiesFromEvents([]models.NotionActivityEvent{event}, testNotionIdGen(), nil, nil)

	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "", a.AccountId)
		assert.Equal(t, "", a.UserEmail)
		assert.Equal(t, "native-notion", a.UserDisplay)
		assert.Equal(t, "native-notion", a.NativeUserId)
		assert.Equal(t, "edited", a.ActionType)
		assert.Equal(t, "page", a.ObjectType)
		assert.Equal(t, "page:p-9", a.ObjectRef)
		assert.Equal(t, "Notion page edited", a.Summary)
	}
}

func TestBuildNotionActivitiesFromEvents_IdDeterminism(t *testing.T) {
	events := []models.NotionActivityEvent{{
		ConnectionId:     13,
		ScopeId:          "ds-3",
		EventId:          "n4",
		OccurredAt:       time.Date(2026, 4, 6, 13, 0, 0, 0, time.UTC),
		EditorUserEmail:  "id@example.com",
		ActionType:       "edited",
		ObjectType:       "page",
		ObjectId:         "page-44",
		SourceObjectType: "page",
	}}

	a1 := buildNotionActivitiesFromEvents(events, testNotionIdGen(), nil, nil)
	a2 := buildNotionActivitiesFromEvents(events, testNotionIdGen(), nil, nil)

	if assert.Len(t, a1, 1) && assert.Len(t, a2, 1) {
		assert.Equal(t, a1[0].Id, a2[0].Id)
	}
}

func TestBuildNotionActivitiesFromEvents_DebounceBoundaryCreatesTwoGroups(t *testing.T) {
	base := time.Date(2026, 4, 6, 9, 4, 59, 0, time.UTC)
	events := []models.NotionActivityEvent{
		{
			ConnectionId:     14,
			ScopeId:          "ds-4",
			EventId:          "nb1",
			OccurredAt:       base,
			EditorUserEmail:  "bucket@example.com",
			ActionType:       "edited",
			ObjectType:       "page",
			ObjectId:         "page-b",
			SourceObjectType: "page",
		},
		{
			ConnectionId:     14,
			ScopeId:          "ds-4",
			EventId:          "nb2",
			OccurredAt:       base.Add(time.Second),
			EditorUserEmail:  "bucket@example.com",
			ActionType:       "edited",
			ObjectType:       "page",
			ObjectId:         "page-b",
			SourceObjectType: "page",
		},
	}

	activities := buildNotionActivitiesFromEvents(events, testNotionIdGen(), nil, nil)
	if assert.Len(t, activities, 2) {
		assert.Equal(t, "Notion page edited", activities[0].Summary)
		assert.Equal(t, "Notion page edited", activities[1].Summary)
		assert.NotEqual(t, activities[0].Id, activities[1].Id)
	}
}

func TestBuildNotionActivitiesFromEvents_NormalizedKeyMergesDifferentCase(t *testing.T) {
	base := time.Date(2026, 4, 6, 9, 2, 0, 0, time.UTC)
	events := []models.NotionActivityEvent{
		{
			ConnectionId:     15,
			ScopeId:          "ds-5",
			EventId:          "nc1",
			OccurredAt:       base,
			EditorUserEmail:  "norm@example.com",
			ActionType:       "Edited",
			ObjectType:       "PAGE",
			ObjectId:         "page-norm",
			SourceObjectType: "page",
		},
		{
			ConnectionId:     15,
			ScopeId:          "ds-5",
			EventId:          "nc2",
			OccurredAt:       base.Add(time.Minute),
			EditorUserEmail:  "norm@example.com",
			ActionType:       "edited",
			ObjectType:       "page",
			ObjectId:         "page-norm",
			SourceObjectType: "page",
		},
	}

	activities := buildNotionActivitiesFromEvents(events, testNotionIdGen(), nil, nil)
	if assert.Len(t, activities, 1) {
		assert.Equal(t, "Notion page edited x2", activities[0].Summary)
	}
}

func TestBuildNotionActivitiesFromEvents_WebhookFlowAndIgnored(t *testing.T) {
	base := time.Date(2026, 4, 7, 10, 45, 0, 0, time.UTC)
	events := []models.NotionActivityEvent{
		{
			ConnectionId:     31,
			ScopeId:          "scope-webhook",
			EventId:          "nwh-1",
			OccurredAt:       base,
			EditorUserEmail:  "notion-webhook@example.com",
			EditorUserId:     "notion-user-1",
			ActionType:       "ignored",
			ObjectType:       "page",
			ObjectId:         "p-1",
			SourceObjectType: "page.unknown_event",
		},
		{
			ConnectionId:     31,
			ScopeId:          "scope-webhook",
			EventId:          "nwh-2",
			OccurredAt:       base.Add(3 * time.Minute),
			EditorUserEmail:  "notion-webhook@example.com",
			EditorUserId:     "notion-user-1",
			ActionType:       "updated",
			ObjectType:       "page",
			ObjectId:         "p-1",
			SourceObjectType: "page.content_updated",
		},
	}

	activities := buildNotionActivitiesFromEvents(events, testNotionIdGen(), nil, nil)
	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "notion", a.SourceSystem)
		assert.Equal(t, "nwh-2", a.SourceEventId)
		assert.Equal(t, "updated", a.ActionType)
		assert.Equal(t, "page", a.ObjectType)
		assert.Equal(t, "p-1", a.ObjectId)
		assert.Equal(t, "page:p-1", a.ObjectRef)
	}
}
