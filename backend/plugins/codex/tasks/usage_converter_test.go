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
	"github.com/apache/incubator-devlake/plugins/codex/models"
	"github.com/stretchr/testify/assert"
)

type mockCodexPlugin struct{}

func (m mockCodexPlugin) Description() string { return "" }
func (m mockCodexPlugin) Name() string        { return "codex" }
func (m mockCodexPlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/codex"
}

func init() {
	plugin.RegisterPlugin("codex", mockCodexPlugin{})
}

func testCodexIdGen() *didgen.DomainIdGenerator {
	return didgen.NewDomainIdGenerator(&models.CodexUsage{})
}

func TestBuildCodexActivity_AllFields(t *testing.T) {
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId:  connectionId,
		ScopeId:       "ws-xyz",
		Date:          date,
		ClientSurface: "cli",
		UserEmail:     "dev@example.com",
		Threads:       10,
		Turns:         50,
		Credits:       25.5,
	}

	idGen := testCodexIdGen()
	activity := buildCodexActivity(idGen, connectionId, "acct-42", u)

	assert.NotEmpty(t, activity.Id)
	assert.Equal(t, "codex", activity.Provider)
	assert.Equal(t, "acct-42", activity.AccountId)
	assert.Equal(t, date, activity.Date)
	assert.Equal(t, "dev@example.com", activity.UserEmail)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "cli", activity.InterfaceType)
	assert.Equal(t, 10, activity.NumSessions)
	assert.Equal(t, 50, activity.SuggestionsCount)
	// Credits are a billing unit, not mapped to EstimatedCostUsd.
	assert.Equal(t, float64(0), activity.EstimatedCostUsd)
}

func TestBuildCodexActivity_ClientSurfaceMapping(t *testing.T) {
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		surface  string
		expected string
	}{
		{"cli", "cli"},
		{"ide", "ide_plugin"},
		{"cloud", "web_ui"},
		{"code_review", "code_review"},
		{"unknown_surface", "unknown_surface"}, // pass-through for unknown values
	}

	for _, tc := range cases {
		t.Run(tc.surface, func(t *testing.T) {
			u := &models.CodexUsage{
				ConnectionId:  connectionId,
				ScopeId:       "ws-xyz",
				Date:          date,
				ClientSurface: tc.surface,
			}
			idGen := testCodexIdGen()
			activity := buildCodexActivity(idGen, connectionId, "", u)
			assert.Equal(t, tc.expected, activity.InterfaceType)
		})
	}
}

func TestBuildCodexActivity_ZeroMetrics(t *testing.T) {
	connectionId := uint64(22)
	date := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId:  connectionId,
		ScopeId:       "ws-zero",
		Date:          date,
		ClientSurface: "cloud",
	}

	idGen := testCodexIdGen()
	activity := buildCodexActivity(idGen, connectionId, "", u)

	assert.Equal(t, int64(0), int64(activity.NumSessions))
	assert.Equal(t, int64(0), int64(activity.SuggestionsCount))
	assert.Equal(t, float64(0), activity.EstimatedCostUsd)
	assert.Equal(t, "codex", activity.Provider)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "web_ui", activity.InterfaceType)
}

func TestBuildCodexActivity_IdDeterminism(t *testing.T) {
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId:  connectionId,
		ScopeId:       "ws-xyz",
		Date:          date,
		ClientSurface: "cli",
		UserEmail:     "user@example.com",
	}

	idGen := testCodexIdGen()
	a1 := buildCodexActivity(idGen, connectionId, "", u)
	a2 := buildCodexActivity(idGen, connectionId, "", u)

	assert.Equal(t, a1.Id, a2.Id, "ID must be deterministic for same input")
}

func TestBuildCodexActivity_DifferentSurfacesGetDifferentIds(t *testing.T) {
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	u1 := &models.CodexUsage{ConnectionId: connectionId, ScopeId: "ws-xyz", Date: date, ClientSurface: "cli"}
	u2 := &models.CodexUsage{ConnectionId: connectionId, ScopeId: "ws-xyz", Date: date, ClientSurface: "ide"}

	idGen := testCodexIdGen()
	a1 := buildCodexActivity(idGen, connectionId, "", u1)
	a2 := buildCodexActivity(idGen, connectionId, "", u2)

	assert.NotEqual(t, a1.Id, a2.Id, "Different surfaces on the same date must produce distinct IDs")
}
