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
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
	"github.com/stretchr/testify/assert"
)

type mockHubspotPlugin struct{}

func (m mockHubspotPlugin) Description() string { return "" }
func (m mockHubspotPlugin) Name() string        { return "hubspot" }
func (m mockHubspotPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/hubspot"
}

func init() {
	plugin.RegisterPlugin("hubspot", mockHubspotPlugin{})
}

func testHubspotIdGen() *didgen.DomainIdGenerator {
	return didgen.NewDomainIdGenerator(&models.HubspotActivityEvent{})
}

func TestBuildHubspotActivitiesFromEvents_DebounceAndSummary(t *testing.T) {
	base := time.Date(2026, 4, 6, 10, 1, 0, 0, time.UTC)

	events := []models.HubspotActivityEvent{
		{
			NoPKModel:        common.NoPKModel{RawDataOrigin: common.RawDataOrigin{RawDataTable: "_raw_hubspot_emails"}},
			ConnectionId:     1,
			ScopeId:          "scope-a",
			EventId:          "e1",
			OccurredAt:       base,
			ActingUserEmail:  "user@example.com",
			ActingUserId:     "u-1",
			ActionType:       "UPDATED",
			ObjectType:       "email",
			ObjectId:         "obj-1",
			SourceObjectType: "email",
		},
		{
			NoPKModel:        common.NoPKModel{RawDataOrigin: common.RawDataOrigin{RawDataTable: "_raw_hubspot_emails"}},
			ConnectionId:     1,
			ScopeId:          "scope-a",
			EventId:          "e2",
			OccurredAt:       base.Add(2 * time.Minute),
			ActingUserEmail:  "user@example.com",
			ActingUserId:     "u-1",
			ActionType:       "UPDATED",
			ObjectType:       "email",
			ObjectId:         "obj-1",
			SourceObjectType: "email",
		},
	}

	calledWith := ""
	activities := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), func(email string) string {
		calledWith = email
		return "acc-123"
	})

	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "acc-123", a.AccountId)
		assert.Equal(t, "hubspot", a.SourceSystem)
		assert.Equal(t, "e2", a.SourceEventId)
		assert.Equal(t, "updated", a.ActionType)
		assert.Equal(t, "email", a.ObjectType)
		assert.Equal(t, "obj-1", a.ObjectId)
		assert.Equal(t, "email:obj-1", a.ObjectRef)
		assert.Equal(t, base.Add(2*time.Minute), a.ActionTime)
		assert.Equal(t, time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC), a.ActionDay)
		assert.Equal(t, "HubSpot email updated x2", a.Summary)
		assert.Equal(t, "_raw_hubspot_emails", a.RawDataOrigin.RawDataTable)
	}
	assert.Equal(t, "user@example.com", calledWith)
}

func TestBuildHubspotActivitiesFromEvents_FallbackIdentityAndDefaults(t *testing.T) {
	event := models.HubspotActivityEvent{
		ConnectionId:     2,
		ScopeId:          "scope-b",
		EventId:          "e3",
		OccurredAt:       time.Date(2026, 4, 6, 11, 30, 0, 0, time.UTC),
		ActingUserId:     "native-77",
		ActionType:       "",
		ObjectType:       "",
		ObjectId:         "obj-9",
		SourceObjectType: "note",
	}

	activities := buildHubspotActivitiesFromEvents([]models.HubspotActivityEvent{event}, testHubspotIdGen(), nil)

	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "", a.AccountId)
		assert.Equal(t, "", a.UserEmail)
		assert.Equal(t, "native-77", a.UserDisplay)
		assert.Equal(t, "native-77", a.NativeUserId)
		assert.Equal(t, "updated", a.ActionType)
		assert.Equal(t, "note", a.ObjectType)
		assert.Equal(t, "note:obj-9", a.ObjectRef)
		assert.Equal(t, "HubSpot note updated", a.Summary)
	}
}

func TestBuildHubspotActivitiesFromEvents_IdDeterminism(t *testing.T) {
	events := []models.HubspotActivityEvent{{
		ConnectionId:     3,
		ScopeId:          "scope-c",
		EventId:          "e4",
		OccurredAt:       time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
		ActingUserEmail:  "id@example.com",
		ActionType:       "updated",
		ObjectType:       "email",
		ObjectId:         "obj-100",
		SourceObjectType: "email",
	}}

	a1 := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), nil)
	a2 := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), nil)

	if assert.Len(t, a1, 1) && assert.Len(t, a2, 1) {
		assert.Equal(t, a1[0].Id, a2[0].Id)
	}
}

func TestBuildHubspotActivitiesFromEvents_DebounceBoundaryCreatesTwoGroups(t *testing.T) {
	base := time.Date(2026, 4, 6, 10, 4, 59, 0, time.UTC)
	events := []models.HubspotActivityEvent{
		{
			ConnectionId:     4,
			ScopeId:          "scope-d",
			EventId:          "b1",
			OccurredAt:       base,
			ActingUserEmail:  "bucket@example.com",
			ActionType:       "updated",
			ObjectType:       "email",
			ObjectId:         "obj-b",
			SourceObjectType: "email",
		},
		{
			ConnectionId:     4,
			ScopeId:          "scope-d",
			EventId:          "b2",
			OccurredAt:       base.Add(time.Second),
			ActingUserEmail:  "bucket@example.com",
			ActionType:       "updated",
			ObjectType:       "email",
			ObjectId:         "obj-b",
			SourceObjectType: "email",
		},
	}

	activities := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), nil)
	if assert.Len(t, activities, 2) {
		assert.Equal(t, "HubSpot email updated", activities[0].Summary)
		assert.Equal(t, "HubSpot email updated", activities[1].Summary)
		assert.NotEqual(t, activities[0].Id, activities[1].Id)
	}
}

func TestBuildHubspotActivitiesFromEvents_NormalizedKeyMergesDifferentCase(t *testing.T) {
	base := time.Date(2026, 4, 6, 10, 2, 0, 0, time.UTC)
	events := []models.HubspotActivityEvent{
		{
			ConnectionId:     5,
			ScopeId:          "scope-e",
			EventId:          "c1",
			OccurredAt:       base,
			ActingUserEmail:  "norm@example.com",
			ActionType:       "UPDATED",
			ObjectType:       "EMAIL",
			ObjectId:         "obj-norm",
			SourceObjectType: "email",
		},
		{
			ConnectionId:     5,
			ScopeId:          "scope-e",
			EventId:          "c2",
			OccurredAt:       base.Add(time.Minute),
			ActingUserEmail:  "norm@example.com",
			ActionType:       "updated",
			ObjectType:       "email",
			ObjectId:         "obj-norm",
			SourceObjectType: "email",
		},
	}

	activities := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), nil)
	if assert.Len(t, activities, 1) {
		assert.Equal(t, "HubSpot email updated x2", activities[0].Summary)
	}
}

func TestBuildHubspotActivitiesFromEvents_WebhookFlowAndIgnored(t *testing.T) {
	base := time.Date(2026, 4, 7, 9, 30, 0, 0, time.UTC)
	events := []models.HubspotActivityEvent{
		{
			ConnectionId:     21,
			ScopeId:          "scope-webhook",
			EventId:          "wh-1",
			OccurredAt:       base,
			ActingUserEmail:  "webhook@example.com",
			ActingUserId:     "hubspot-user-1",
			ActionType:       "ignored",
			ObjectType:       "contact",
			ObjectId:         "c-1",
			SourceObjectType: "contact.creation",
		},
		{
			ConnectionId:     21,
			ScopeId:          "scope-webhook",
			EventId:          "wh-2",
			OccurredAt:       base.Add(2 * time.Minute),
			ActingUserEmail:  "webhook@example.com",
			ActingUserId:     "hubspot-user-1",
			ActionType:       "updated",
			ObjectType:       "contact",
			ObjectId:         "c-1",
			SourceObjectType: "contact.propertyChange",
		},
	}

	activities := buildHubspotActivitiesFromEvents(events, testHubspotIdGen(), nil)
	if assert.Len(t, activities, 1) {
		a := activities[0]
		assert.Equal(t, "hubspot", a.SourceSystem)
		assert.Equal(t, "wh-2", a.SourceEventId)
		assert.Equal(t, "updated", a.ActionType)
		assert.Equal(t, "contact", a.ObjectType)
		assert.Equal(t, "c-1", a.ObjectId)
		assert.Equal(t, "contact:c-1", a.ObjectRef)
	}
}
