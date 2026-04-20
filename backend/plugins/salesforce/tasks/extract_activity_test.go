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

func TestBuildSalesforceActivityEvent_MapsCreatedRecord(t *testing.T) {
	row := []byte(`{
		"Id":"001xx000003DGbYAAW",
		"CreatedDate":"2026-04-16T10:00:00.000+0000",
		"CreatedById":"005xx000001Sv6aAAC",
		"LastModifiedDate":"2026-04-16T10:00:00.000+0000",
		"LastModifiedById":"005xx000001Sv6aAAC",
		"SystemModstamp":"2026-04-16T10:00:00.000+0000",
		"attributes":{"type":"Account"}
	}`)

	event, err := buildSalesforceActivityEvent(row, 7, "org-1", "Account")
	expectedTime := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	if assert.NoError(t, err) && assert.NotNil(t, event) {
		assert.Equal(t, uint64(7), event.ConnectionId)
		assert.Equal(t, "org-1", event.ScopeId)
		assert.Equal(t, "Account:001xx000003DGbYAAW:1776333600000", event.EventId)
		assert.Equal(t, expectedTime, event.OccurredAt)
		assert.Equal(t, "005xx000001Sv6aAAC", event.ActingUserId)
		assert.Equal(t, "created", event.ActionType)
		assert.Equal(t, "account", event.ObjectType)
		assert.Equal(t, "001xx000003DGbYAAW", event.ObjectId)
		assert.Equal(t, "Account", event.SourceObjectType)
	}
}

func TestBuildSalesforceActivityEvent_MapsUpdatedRecord(t *testing.T) {
	row := []byte(`{
		"Id":"500xx0000025QWEAA2",
		"CreatedDate":"2026-04-16T10:00:00.000+0000",
		"CreatedById":"005xx000001Create",
		"LastModifiedDate":"2026-04-16T11:15:00.000+0000",
		"LastModifiedById":"005xx000001Editor",
		"SystemModstamp":"2026-04-16T11:15:00.000+0000",
		"attributes":{"type":"Case"}
	}`)

	event, err := buildSalesforceActivityEvent(row, 7, "org-1", "Case")
	expectedTime := time.Date(2026, 4, 16, 11, 15, 0, 0, time.UTC)

	if assert.NoError(t, err) && assert.NotNil(t, event) {
		assert.Equal(t, expectedTime, event.OccurredAt)
		assert.Equal(t, "005xx000001Editor", event.ActingUserId)
		assert.Equal(t, "updated", event.ActionType)
		assert.Equal(t, "case", event.ObjectType)
	}
}
