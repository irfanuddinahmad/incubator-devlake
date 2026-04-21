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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeHubspotWebhookEvents_ArrayAndSingle(t *testing.T) {
	arr, err := decodeHubspotWebhookEvents([]byte(`[
		{"eventId":1,"occurredAt":1775466120000,"subscriptionType":"contact.creation","objectId":10}
	]`))
	if assert.NoError(t, err) {
		assert.Len(t, arr, 1)
	}

	single, err := decodeHubspotWebhookEvents([]byte(`{"eventId":2,"occurredAt":1775466120000,"subscriptionType":"contact.propertyChange","objectId":11}`))
	if assert.NoError(t, err) {
		assert.Len(t, single, 1)
		assert.Equal(t, int64(2), single[0].EventId)
	}
}

func TestMapHubspotActionType(t *testing.T) {
	assert.Equal(t, "created", mapHubspotActionType("contact.creation", ""))
	assert.Equal(t, "updated", mapHubspotActionType("contact.propertyChange", ""))
	assert.Equal(t, "deleted", mapHubspotActionType("contact.deletion", ""))
	assert.Equal(t, "created", mapHubspotActionType("", "CREATED"))
	assert.Equal(t, "ignored", mapHubspotActionType("workflow.status", ""))
}

func TestMapHubspotWebhookEvent(t *testing.T) {
	e := hubspotWebhookEvent{
		EventId:          99,
		OccurredAt:       1775466120000,
		SubscriptionType: "contact.propertyChange",
		ObjectId:         321,
		SourceId:         "userId:555",
	}
	mapped := mapHubspotWebhookEvent(12, "scope-a", e, 0)
	assert.Equal(t, uint64(12), mapped.ConnectionId)
	assert.Equal(t, "scope-a", mapped.ScopeId)
	assert.Equal(t, "99", mapped.EventId)
	assert.Equal(t, "updated", mapped.ActionType)
	assert.Equal(t, "contact", mapped.ObjectType)
	assert.Equal(t, "321", mapped.ObjectId)
	assert.Equal(t, "userId:555", mapped.ActingUserId)
}
