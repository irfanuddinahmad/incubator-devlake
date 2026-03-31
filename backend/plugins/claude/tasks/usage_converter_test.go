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
	"github.com/apache/incubator-devlake/plugins/claude/models"
	"github.com/stretchr/testify/assert"
)

type mockClaudePlugin struct{}

func (m mockClaudePlugin) Description() string { return "" }
func (m mockClaudePlugin) Name() string        { return "claude" }
func (m mockClaudePlugin) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/claude"
}

func init() {
	plugin.RegisterPlugin("claude", mockClaudePlugin{})
}

// testClaudeIdGen returns a DomainIdGenerator scoped to ClaudeUsage, same as production.
func testClaudeIdGen() *didgen.DomainIdGenerator {
	return didgen.NewDomainIdGenerator(&models.ClaudeUsage{})
}

func TestBuildClaudeActivity_AllFields(t *testing.T) {
	connectionId := uint64(1)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	usage := &models.ClaudeUsage{
		ConnectionId:     connectionId,
		Date:             date,
		UserEmail:        "alice@example.com",
		NumSessions:      5,
		LinesAdded:       120,
		LinesRemoved:     30,
		CommitsByClaude:  3,
		PrsByClaude:      1,
		Model:            "claude-sonnet-4-5",
		InputTokens:      10000,
		OutputTokens:     2500,
		EstimatedCostUsd: 0.045,
	}

	idGen := testClaudeIdGen()
	activity := buildClaudeActivity(idGen, connectionId, "global-account-42", usage)

	assert.NotEmpty(t, activity.Id, "Id should be generated")
	assert.Equal(t, "claude", activity.Provider)
	assert.Equal(t, "global-account-42", activity.AccountId)
	assert.Equal(t, "alice@example.com", activity.UserEmail)
	assert.Equal(t, date, activity.Date)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "cli", activity.InterfaceType)
	assert.Equal(t, "claude-sonnet-4-5", activity.Model)
	assert.Equal(t, 5, activity.NumSessions)
	assert.Equal(t, 120, activity.LinesAdded)
	assert.Equal(t, 30, activity.LinesRemoved)
	assert.Equal(t, 3, activity.CommitsCreated)
	assert.Equal(t, 1, activity.PrsCreated)
	assert.Equal(t, int64(10000), activity.InputTokens)
	assert.Equal(t, int64(2500), activity.OutputTokens)
	assert.InDelta(t, 0.045, activity.EstimatedCostUsd, 1e-9)
}

func TestBuildClaudeActivity_EmptyEmail(t *testing.T) {
	connectionId := uint64(2)
	date := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	usage := &models.ClaudeUsage{
		ConnectionId: connectionId,
		Date:         date,
		UserEmail:    "",
		Model:        "claude-opus-3",
		NumSessions:  1,
	}

	idGen := testClaudeIdGen()
	// When email lookup fails, resolveAccountId returns ""; test that the helper
	// faithfully passes the empty accountId through.
	activity := buildClaudeActivity(idGen, connectionId, "", usage)

	assert.NotEmpty(t, activity.Id)
	assert.Equal(t, "claude", activity.Provider)
	assert.Equal(t, "", activity.AccountId, "AccountId should be empty when email is missing")
	assert.Equal(t, "", activity.UserEmail)
	assert.Equal(t, "CODE_EDIT", activity.Type)
	assert.Equal(t, "cli", activity.InterfaceType)
}

func TestBuildClaudeActivity_ZeroMetrics(t *testing.T) {
	connectionId := uint64(3)
	date := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	usage := &models.ClaudeUsage{
		ConnectionId: connectionId,
		Date:         date,
		UserEmail:    "bob@example.com",
		Model:        "claude-haiku-3",
	}

	idGen := testClaudeIdGen()
	activity := buildClaudeActivity(idGen, connectionId, "acc-bob", usage)

	assert.Equal(t, 0, activity.NumSessions)
	assert.Equal(t, 0, activity.LinesAdded)
	assert.Equal(t, 0, activity.LinesRemoved)
	assert.Equal(t, 0, activity.CommitsCreated)
	assert.Equal(t, 0, activity.PrsCreated)
	assert.Equal(t, int64(0), activity.InputTokens)
	assert.Equal(t, int64(0), activity.OutputTokens)
	assert.Equal(t, float64(0), activity.EstimatedCostUsd)
}

func TestBuildClaudeActivity_IdDeterminism(t *testing.T) {
	// The same input should always produce the same ID (deterministic).
	connectionId := uint64(1)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	usage := &models.ClaudeUsage{
		ConnectionId: connectionId,
		Date:         date,
		UserEmail:    "det@example.com",
		Model:        "claude-sonnet-4-5",
	}

	idGen := testClaudeIdGen()
	a1 := buildClaudeActivity(idGen, connectionId, "", usage)
	a2 := buildClaudeActivity(idGen, connectionId, "", usage)

	assert.Equal(t, a1.Id, a2.Id, "ID generation must be deterministic for the same input")
}
