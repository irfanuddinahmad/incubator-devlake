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

	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
	"github.com/stretchr/testify/assert"
)

type mockCursorPlugin struct{}

func (m mockCursorPlugin) Description() string { return "" }
func (m mockCursorPlugin) Name() string        { return "cursor" }
func (m mockCursorPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/cursor"
}

func init() {
	plugin.RegisterPlugin("cursor", mockCursorPlugin{})
}

func testCursorIdGen() *didgen.DomainIdGenerator {
	return didgen.NewDomainIdGenerator(&models.CursorDailyUsage{})
}

func TestBuildCursorDailyActivity_AllFields(t *testing.T) {
	connectionId := uint64(10)
	day := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	u := &models.CursorDailyUsage{
		ConnectionId:       connectionId,
		ScopeId:            "scope-abc",
		Day:                day,
		UserEmail:          "dev@cursor.ai",
		TotalTabsShown:     200,
		TotalTabsAccepted:  80,
		AcceptedLinesAdded: 60,
		TotalLinesDeleted:  15,
	}

	idGen := testCursorIdGen()
	activity := buildCursorDailyActivity(idGen, connectionId, "account-dev", u)

	assert.NotEmpty(t, activity.Id)
	assert.Equal(t, "cursor", activity.Provider)
	assert.Equal(t, "account-dev", activity.AccountId)
	assert.Equal(t, "dev@cursor.ai", activity.UserEmail)
	assert.Equal(t, day, activity.Date)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "ide_plugin", activity.InterfaceType)
	assert.Equal(t, 200, activity.SuggestionsCount)
	assert.Equal(t, 80, activity.AcceptanceCount)
	assert.Equal(t, 60, activity.LinesAdded)
	assert.Equal(t, 15, activity.LinesRemoved)
}

func TestBuildCursorDailyActivity_FieldMapping(t *testing.T) {
	// Explicitly verify each field maps to the correct AiActivity field.
	connectionId := uint64(11)
	day := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)

	u := &models.CursorDailyUsage{
		ConnectionId:       connectionId,
		Day:                day,
		UserEmail:          "mapper@cursor.ai",
		TotalTabsShown:     500, // → SuggestionsCount
		TotalTabsAccepted:  150, // → AcceptanceCount
		TotalLinesAdded:    999, // intentionally NOT mapped
		AcceptedLinesAdded: 120, // → LinesAdded
		TotalLinesDeleted:  40,  // → LinesRemoved
	}

	idGen := testCursorIdGen()
	activity := buildCursorDailyActivity(idGen, connectionId, "", u)

	assert.Equal(t, 500, activity.SuggestionsCount, "TotalTabsShown → SuggestionsCount")
	assert.Equal(t, 150, activity.AcceptanceCount, "TotalTabsAccepted → AcceptanceCount")
	assert.Equal(t, 120, activity.LinesAdded, "AcceptedLinesAdded → LinesAdded")
	assert.Equal(t, 40, activity.LinesRemoved, "TotalLinesDeleted → LinesRemoved")
	// tokens / commits / prs are not available from Cursor daily usage
	assert.Equal(t, int64(0), activity.InputTokens)
	assert.Equal(t, int64(0), activity.OutputTokens)
	assert.Equal(t, 0, activity.CommitsCreated)
	assert.Equal(t, 0, activity.PrsCreated)
}

func TestBuildCursorDailyActivity_EmptyEmail(t *testing.T) {
	connectionId := uint64(12)
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	u := &models.CursorDailyUsage{
		ConnectionId: connectionId,
		Day:          day,
		UserEmail:    "",
	}

	idGen := testCursorIdGen()
	activity := buildCursorDailyActivity(idGen, connectionId, "", u)

	assert.NotEmpty(t, activity.Id)
	assert.Equal(t, "", activity.AccountId)
	assert.Equal(t, "", activity.UserEmail)
	assert.Equal(t, "cursor", activity.Provider)
	assert.Equal(t, "ide_plugin", activity.InterfaceType)
}

func TestBuildCursorDailyActivity_IdDeterminism(t *testing.T) {
	connectionId := uint64(10)
	day := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	u := &models.CursorDailyUsage{
		ConnectionId: connectionId,
		Day:          day,
		UserEmail:    "det@cursor.ai",
	}

	idGen := testCursorIdGen()
	a1 := buildCursorDailyActivity(idGen, connectionId, "", u)
	a2 := buildCursorDailyActivity(idGen, connectionId, "", u)

	assert.Equal(t, a1.Id, a2.Id, "ID must be deterministic for same input")
}
