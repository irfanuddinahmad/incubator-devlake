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
	"encoding/json"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/claude/models"
)

// ExtractUsageMeta defines the metadata for the Claude usage extractor subtask.
var ExtractUsageMeta = plugin.SubTaskMeta{
	Name:             "extractUsage",
	EntryPoint:       ExtractUsage,
	EnabledByDefault: true,
	Description:      "Extract Claude Code usage records from raw data into tool-layer models",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// --- Anthropic API response structures ---

// claudeActor identifies the user or API key that performed the actions.
type claudeActor struct {
	Type         string `json:"type"`          // "user_actor" or "api_actor"
	EmailAddress string `json:"email_address"` // set when type == "user_actor"
	ApiKeyName   string `json:"api_key_name"`  // set when type == "api_actor"
}

type claudeLinesOfCode struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
}

type claudeCoreMetrics struct {
	NumSessions         int               `json:"num_sessions"`
	LinesOfCode         claudeLinesOfCode `json:"lines_of_code"`
	CommitsByClaudeCode int               `json:"commits_by_claude_code"`
	PrsByClaudeCode     int               `json:"pull_requests_by_claude_code"`
}

type claudeModelTokens struct {
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	CacheRead     int64 `json:"cache_read"`
	CacheCreation int64 `json:"cache_creation"`
}

type claudeToolAction struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

type claudeToolActions struct {
	EditTool         claudeToolAction `json:"edit_tool"`
	MultiEditTool    claudeToolAction `json:"multi_edit_tool"`
	WriteTool        claudeToolAction `json:"write_tool"`
	NotebookEditTool claudeToolAction `json:"notebook_edit_tool"`
}

type claudeModelCost struct {
	Amount   int    `json:"amount"` // in cents
	Currency string `json:"currency"`
}

type claudeModelBreakdownItem struct {
	Model         string            `json:"model"`
	Tokens        claudeModelTokens `json:"tokens"`
	EstimatedCost claudeModelCost   `json:"estimated_cost"`
}

// claudeUsageRecord mirrors the per-user record returned inside the `data` array
// by the Anthropic claude_code usage report endpoint.
type claudeUsageRecord struct {
	// RFC 3339 timestamp, e.g. "2025-09-01T00:00:00Z"
	Date           string                     `json:"date"`
	Actor          claudeActor                `json:"actor"`
	CoreMetrics    claudeCoreMetrics          `json:"core_metrics"`
	ToolActions    claudeToolActions          `json:"tool_actions"`
	ModelBreakdown []claudeModelBreakdownItem `json:"model_breakdown"`
}

// ExtractUsage parses raw Claude usage JSON rows into ClaudeUsage tool-layer records.
// Each raw row corresponds to one user's activity on a given date.
// When a `model_breakdown` array is present, one row per model is emitted.
func ExtractUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*ClaudeTaskData)
	if !ok {
		return errors.Default.New("task data is not ClaudeTaskData")
	}

	params := claudeRawParams{
		ConnectionId:   data.Options.ConnectionId,
		ScopeId:        data.Options.ScopeId,
		OrganizationId: data.Connection.OrganizationId,
	}

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:     taskCtx,
			Table:   rawUsageTable,
			Options: params,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var record claudeUsageRecord
			if parseErr := errors.Convert(json.Unmarshal(row.Data, &record)); parseErr != nil {
				return nil, parseErr
			}

			// Date field is RFC 3339, e.g. "2025-09-01T00:00:00Z"
			date, timeErr := time.Parse(time.RFC3339, record.Date)
			if timeErr != nil {
				// Fallback: try plain date format
				date, timeErr = time.Parse("2006-01-02", record.Date)
				if timeErr != nil {
					return nil, errors.Default.Wrap(timeErr, "failed to parse date: "+record.Date)
				}
			}

			// Resolve user identity from actor.
			userEmail := record.Actor.EmailAddress
			if userEmail == "" {
				// api_actor: use api_key_name as a stand-in identifier
				userEmail = record.Actor.ApiKeyName
			}

			cm := record.CoreMetrics
			ta := record.ToolActions

			// Emit one row per model in the breakdown.
			if len(record.ModelBreakdown) > 0 {
				results := make([]interface{}, 0, len(record.ModelBreakdown))
				for _, mb := range record.ModelBreakdown {
					usage := &models.ClaudeUsage{
						ConnectionId:             data.Options.ConnectionId,
						ScopeId:                  data.Options.ScopeId,
						Date:                     date,
						UserEmail:                userEmail,
						Model:                    mb.Model,
						NumSessions:              cm.NumSessions,
						LinesAdded:               cm.LinesOfCode.Added,
						LinesRemoved:             cm.LinesOfCode.Removed,
						CommitsByClaude:          cm.CommitsByClaudeCode,
						PrsByClaude:              cm.PrsByClaudeCode,
						EditToolAccepted:         ta.EditTool.Accepted,
						EditToolRejected:         ta.EditTool.Rejected,
						MultiEditToolAccepted:    ta.MultiEditTool.Accepted,
						MultiEditToolRejected:    ta.MultiEditTool.Rejected,
						WriteToolAccepted:        ta.WriteTool.Accepted,
						WriteToolRejected:        ta.WriteTool.Rejected,
						NotebookEditToolAccepted: ta.NotebookEditTool.Accepted,
						NotebookEditToolRejected: ta.NotebookEditTool.Rejected,
						InputTokens:              mb.Tokens.Input,
						OutputTokens:             mb.Tokens.Output,
						CacheReadTokens:          mb.Tokens.CacheRead,
						CacheCreationTokens:      mb.Tokens.CacheCreation,
						EstimatedCostUsd:         float64(mb.EstimatedCost.Amount) / 100.0,
					}
					results = append(results, usage)
				}
				return results, nil
			}

			// No model breakdown — emit a single aggregate row.
			usage := &models.ClaudeUsage{
				ConnectionId:             data.Options.ConnectionId,
				ScopeId:                  data.Options.ScopeId,
				Date:                     date,
				UserEmail:                userEmail,
				NumSessions:              cm.NumSessions,
				LinesAdded:               cm.LinesOfCode.Added,
				LinesRemoved:             cm.LinesOfCode.Removed,
				CommitsByClaude:          cm.CommitsByClaudeCode,
				PrsByClaude:              cm.PrsByClaudeCode,
				EditToolAccepted:         ta.EditTool.Accepted,
				EditToolRejected:         ta.EditTool.Rejected,
				MultiEditToolAccepted:    ta.MultiEditTool.Accepted,
				MultiEditToolRejected:    ta.MultiEditTool.Rejected,
				WriteToolAccepted:        ta.WriteTool.Accepted,
				WriteToolRejected:        ta.WriteTool.Rejected,
				NotebookEditToolAccepted: ta.NotebookEditTool.Accepted,
				NotebookEditToolRejected: ta.NotebookEditTool.Rejected,
			}
			return []interface{}{usage}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
