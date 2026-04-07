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

func TestMapNotionType(t *testing.T) {
	action, object := mapNotionType("page.content_updated")
	assert.Equal(t, "updated", action)
	assert.Equal(t, "page", object)

	action, object = mapNotionType("comment.created")
	assert.Equal(t, "created", action)
	assert.Equal(t, "comment", object)

	action, object = mapNotionType("unknown.event")
	assert.Equal(t, "ignored", action)
	assert.Equal(t, "unknown", object)
}

func TestDecodeNotionWebhookEvent(t *testing.T) {
	e, err := decodeNotionWebhookEvent([]byte(`{
		"id":"evt-1",
		"timestamp":"2026-04-06T09:02:00Z",
		"type":"page.created",
		"authors":[{"id":"u-1","type":"person"}],
		"entity":{"id":"p-1","type":"page"}
	}`))
	if assert.NoError(t, err) {
		assert.Equal(t, "evt-1", e.Id)
		assert.Equal(t, "page.created", e.Type)
	}
}

func TestMapNotionWebhookEvent(t *testing.T) {
	e := &notionWebhookEvent{
		Id:        "evt-2",
		Timestamp: "2026-04-06T09:02:00Z",
		Type:      "data_source.schema_updated",
		Authors: []notionWebhookAuthor{
			{Id: "bot-1", Type: "bot"},
			{Id: "user-9", Type: "person"},
		},
		Entity: notionWebhookEntity{Id: "ds-1", Type: "data_source"},
	}
	mapped := mapNotionWebhookEvent(44, "scope-x", e)
	assert.Equal(t, uint64(44), mapped.ConnectionId)
	assert.Equal(t, "scope-x", mapped.ScopeId)
	assert.Equal(t, "evt-2", mapped.EventId)
	assert.Equal(t, "updated", mapped.ActionType)
	assert.Equal(t, "data_source", mapped.ObjectType)
	assert.Equal(t, "ds-1", mapped.ObjectId)
	assert.Equal(t, "user-9", mapped.EditorUserId)
}
