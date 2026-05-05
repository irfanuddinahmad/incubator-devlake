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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveSalesforceSince_PrefersCollectedSince(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	collectedSince := time.Date(2026, 4, 16, 8, 30, 0, 0, time.FixedZone("offset", 2*60*60))
	occurredAfter := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	since := resolveSalesforceSince(&collectedSince, &occurredAfter, now)

	if assert.NotNil(t, since) {
		assert.Equal(t, collectedSince.UTC(), *since)
	}
}

func TestResolveSalesforceSince_FallsBackToOccurredAfter(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	occurredAfter := time.Date(2026, 4, 5, 15, 0, 0, 0, time.FixedZone("offset", -5*60*60))

	since := resolveSalesforceSince(nil, &occurredAfter, now)

	if assert.NotNil(t, since) {
		assert.Equal(t, occurredAfter.UTC(), *since)
	}
}

func TestResolveSalesforceSince_DefaultsToLast30Days(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	since := resolveSalesforceSince(nil, nil, now)

	if assert.NotNil(t, since) {
		assert.Equal(t, now.AddDate(0, 0, -30), *since)
	}
}

func TestBuildSalesforceActivityQuery_UsesUnquotedDateTimeLiterals(t *testing.T) {
	since := time.Date(2026, 4, 17, 10, 11, 12, 0, time.UTC)
	until := time.Date(2026, 4, 18, 13, 14, 15, 0, time.UTC)

	query, err := buildSalesforceActivityQuery("Account", &since, &until)
	assert.Nil(t, err)

	assert.Contains(t, query, "SystemModstamp >= 2026-04-17T10:11:12Z")
	assert.Contains(t, query, "SystemModstamp < 2026-04-18T13:14:15Z")
	assert.False(t, strings.Contains(query, "'2026-04-17T10:11:12Z'"))
	assert.False(t, strings.Contains(query, "'2026-04-18T13:14:15Z'"))
}

func TestBuildSalesforceActivityQuery_RejectsUnknownObjectType(t *testing.T) {
	_, err := buildSalesforceActivityQuery("Account; DROP TABLE foo--", nil, nil)
	assert.NotNil(t, err)

	_, err = buildSalesforceActivityQuery("UnknownObject", nil, nil)
	assert.NotNil(t, err)
}

func TestResolveSalesforceActivityCheckpoint_CapsAtOccurredBefore(t *testing.T) {
	since := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	runUntil := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)

	checkpoint := resolveSalesforceActivityCheckpoint(&since, &until, &runUntil)

	assert.Equal(t, until, checkpoint)
}

func TestResolveSalesforceActivityCheckpoint_DoesNotMoveBackBeforeSince(t *testing.T) {
	since := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	runUntil := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	checkpoint := resolveSalesforceActivityCheckpoint(&since, &until, &runUntil)

	assert.Equal(t, since, checkpoint)
}
