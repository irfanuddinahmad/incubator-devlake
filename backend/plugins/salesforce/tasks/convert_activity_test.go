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
	"fmt"
	"testing"
	"time"

	"github.com/apache/incubator-devlake/plugins/salesforce/models"
	"github.com/stretchr/testify/assert"
)

func stubIdGen() salesforceActivityIdFunc {
	return func(pks ...interface{}) string {
		parts := make([]string, 0, len(pks))
		for _, pk := range pks {
			parts = append(parts, fmt.Sprintf("%v", pk))
		}
		return "id:" + fmt.Sprint(parts)
	}
}

func newTestEvent(occurredAt time.Time, userId, action, objectType, objectId string) models.SalesforceActivityEvent {
	return models.SalesforceActivityEvent{
		ConnectionId:     1,
		ScopeId:          "org-1",
		EventId:          objectType + ":" + objectId + ":" + occurredAt.UTC().Format(time.RFC3339Nano),
		OccurredAt:       occurredAt.UTC(),
		ActingUserId:     userId,
		ActionType:       action,
		ObjectType:       objectType,
		ObjectId:         objectId,
		SourceObjectType: objectType,
	}
}

func TestBuildSalesforceActivitiesFromEvents_DebouncesWithinWindow(t *testing.T) {
	idGen := stubIdGen()
	base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(base, "u1", "updated", "Account", "a1"),
		newTestEvent(base.Add(time.Minute), "u1", "updated", "Account", "a1"),
		newTestEvent(base.Add(2*time.Minute), "u1", "updated", "Account", "a1"),
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, nil, nil)

	if assert.Len(t, activities, 1) {
		assert.Equal(t, "Salesforce account updated x3", activities[0].Summary)
		assert.Equal(t, base.Add(2*time.Minute), activities[0].ActionTime)
		assert.Equal(t, "u1", activities[0].NativeUserId)
	}
}

func TestBuildSalesforceActivitiesFromEvents_SeparatesAcrossDebounceBoundary(t *testing.T) {
	idGen := stubIdGen()

	bucket1 := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	bucket2 := time.Date(2026, 4, 17, 10, 5, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(bucket1, "u1", "updated", "Account", "a1"),
		newTestEvent(bucket2, "u1", "updated", "Account", "a1"),
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, nil, nil)
	assert.Len(t, activities, 2, "events in different 5-minute buckets must not merge")
}

func TestBuildSalesforceActivitiesFromEvents_GroupsByUserActionAndObject(t *testing.T) {
	idGen := stubIdGen()
	base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(base, "u1", "created", "Account", "a1"),
		newTestEvent(base.Add(30*time.Second), "u1", "updated", "Account", "a1"),
		newTestEvent(base.Add(time.Minute), "u2", "updated", "Account", "a1"),
		newTestEvent(base.Add(2*time.Minute), "u1", "updated", "Contact", "c1"),
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, nil, nil)
	assert.Len(t, activities, 4, "different user/action/object triples produce separate activities")
}

func TestBuildSalesforceActivitiesFromEvents_FiltersIgnoredEvents(t *testing.T) {
	idGen := stubIdGen()
	base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(base, "u1", "ignored", "Account", "a1"),
		newTestEvent(base.Add(time.Minute), "u1", "updated", "Account", "a1"),
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, nil, nil)
	if assert.Len(t, activities, 1) {
		assert.Equal(t, "updated", activities[0].ActionType)
	}
}

func TestBuildSalesforceActivitiesFromEvents_EnrichesFromUserMap(t *testing.T) {
	idGen := stubIdGen()
	base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(base, "005U001", "updated", "Account", "a1"),
	}
	userMap := map[string]models.SalesforceUser{
		"005U001": {UserId: "005U001", Email: "alice@example.com", Name: "Alice"},
	}
	resolveAccountId := func(email string) string {
		if email == "alice@example.com" {
			return "acct-42"
		}
		return ""
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, resolveAccountId, userMap)
	if assert.Len(t, activities, 1) {
		assert.Equal(t, "alice@example.com", activities[0].UserEmail)
		assert.Equal(t, "Alice", activities[0].UserDisplay)
		assert.Equal(t, "acct-42", activities[0].AccountId)
	}
}

func TestBuildSalesforceActivitiesFromEvents_FallsBackToUnknownUser(t *testing.T) {
	idGen := stubIdGen()
	base := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	events := []models.SalesforceActivityEvent{
		newTestEvent(base, "", "updated", "Account", "a1"),
	}

	activities := buildSalesforceActivitiesFromEvents(events, idGen, nil, nil)
	if assert.Len(t, activities, 1) {
		assert.Equal(t, "Salesforce user", activities[0].UserDisplay)
		assert.Equal(t, "", activities[0].NativeUserId)
	}
}

func TestBuildSalesforceActivitiesFromEvents_EmptyInput(t *testing.T) {
	idGen := stubIdGen()
	activities := buildSalesforceActivitiesFromEvents(nil, idGen, nil, nil)
	assert.Empty(t, activities)
}

func TestBuildSalesforceUsersQuery_ShapeIsStable(t *testing.T) {
	q := buildSalesforceUsersQuery()
	assert.Contains(t, q, "FROM User")
	assert.Contains(t, q, "WHERE IsActive = true")
	assert.Contains(t, q, "ORDER BY Id ASC")
	// no user-controlled interpolation: literal must not contain placeholders
	assert.NotContains(t, q, "%")
	assert.NotContains(t, q, "?")
}
