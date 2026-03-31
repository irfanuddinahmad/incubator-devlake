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
		ConnectionId:     connectionId,
		ScopeId:          "proj-xyz",
		Date:             date,
		Model:            "gpt-4o",
		InputTokens:      50000,
		OutputTokens:     12000,
		EstimatedCostUsd: 0.18,
	}

	idGen := testCodexIdGen()
	activity := buildCodexActivity(idGen, connectionId, u)

	assert.NotEmpty(t, activity.Id)
	assert.Equal(t, "codex", activity.Provider)
	assert.Equal(t, date, activity.Date)
	assert.Equal(t, "gpt-4o", activity.Model)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "cli", activity.InterfaceType)
	assert.Equal(t, int64(50000), activity.InputTokens)
	assert.Equal(t, int64(12000), activity.OutputTokens)
	assert.InDelta(t, 0.18, activity.EstimatedCostUsd, 1e-9)
}

func TestBuildCodexActivity_NoUserContext(t *testing.T) {
	// Codex records have no per-user data; UserEmail and AccountId must always be empty.
	connectionId := uint64(21)
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId: connectionId,
		Date:         date,
		Model:        "gpt-4-turbo",
		InputTokens:  1000,
		OutputTokens: 200,
	}

	idGen := testCodexIdGen()
	activity := buildCodexActivity(idGen, connectionId, u)

	assert.Equal(t, "", activity.UserEmail, "Codex activities must have no user email")
	assert.Equal(t, "", activity.AccountId, "Codex activities must have no account ID")
	// No autocomplete metrics
	assert.Equal(t, 0, activity.SuggestionsCount)
	assert.Equal(t, 0, activity.AcceptanceCount)
	// No commit/PR agentic metrics
	assert.Equal(t, 0, activity.CommitsCreated)
	assert.Equal(t, 0, activity.PrsCreated)
}

func TestBuildCodexActivity_ZeroTokens(t *testing.T) {
	connectionId := uint64(22)
	date := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId: connectionId,
		Date:         date,
		Model:        "gpt-3.5-turbo",
	}

	idGen := testCodexIdGen()
	activity := buildCodexActivity(idGen, connectionId, u)

	assert.Equal(t, int64(0), activity.InputTokens)
	assert.Equal(t, int64(0), activity.OutputTokens)
	assert.Equal(t, float64(0), activity.EstimatedCostUsd)
	assert.Equal(t, "codex", activity.Provider)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "cli", activity.InterfaceType)
}

func TestBuildCodexActivity_IdDeterminism(t *testing.T) {
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	u := &models.CodexUsage{
		ConnectionId: connectionId,
		Date:         date,
		Model:        "gpt-4o",
	}

	idGen := testCodexIdGen()
	a1 := buildCodexActivity(idGen, connectionId, u)
	a2 := buildCodexActivity(idGen, connectionId, u)

	assert.Equal(t, a1.Id, a2.Id, "ID must be deterministic for same input")
}

func TestBuildCodexActivity_DifferentModelsGetDifferentIds(t *testing.T) {
	// Two records for the same date but different models should produce distinct IDs.
	connectionId := uint64(20)
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	u1 := &models.CodexUsage{ConnectionId: connectionId, Date: date, Model: "gpt-4o"}
	u2 := &models.CodexUsage{ConnectionId: connectionId, Date: date, Model: "gpt-3.5-turbo"}

	idGen := testCodexIdGen()
	a1 := buildCodexActivity(idGen, connectionId, u1)
	a2 := buildCodexActivity(idGen, connectionId, u2)

	assert.NotEqual(t, a1.Id, a2.Id, "Different models on the same date must produce distinct IDs")
}
