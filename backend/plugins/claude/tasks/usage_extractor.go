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

// claudeUsageRecord mirrors the per-user record returned inside the `data` array
// by the Anthropic claude_code usage report endpoint.
type claudeUsageRecord struct {
	// Date of the usage report (YYYY-MM-DD).
	Date string `json:"date"`

	// User identity
	UserEmail string `json:"user_email"`

	// Core metrics
	NumSessions     int `json:"num_sessions"`
	LinesAdded      int `json:"lines_added"`
	LinesRemoved    int `json:"lines_removed"`
	CommitsByClaude int `json:"commits_by_claude"`
	PrsByClaude     int `json:"prs_by_claude"`

	// Model breakdown – the API may return a flat record or a nested breakdown.
	// We capture the primary model from the top-level field when present.
	Model string `json:"model"`

	// Token usage & cost
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUsd float64 `json:"estimated_cost_usd"`

	// Nested model breakdown (optional – aggregated when present).
	ModelBreakdown []claudeModelBreakdown `json:"model_breakdown"`
}

// claudeModelBreakdown represents per-model token/cost breakdown within a usage record.
type claudeModelBreakdown struct {
	Model            string  `json:"model"`
	InputTokens      int64   `json:"input_tokens"`
	OutputTokens     int64   `json:"output_tokens"`
	EstimatedCostUsd float64 `json:"estimated_cost_usd"`
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

			date, timeErr := time.Parse("2006-01-02", record.Date)
			if timeErr != nil {
				return nil, errors.Default.Wrap(timeErr, "failed to parse date: "+record.Date)
			}

			// If the record has a per-model breakdown, emit one row per model.
			if len(record.ModelBreakdown) > 0 {
				results := make([]interface{}, 0, len(record.ModelBreakdown))
				for _, mb := range record.ModelBreakdown {
					usage := &models.ClaudeUsage{
						ConnectionId:     data.Options.ConnectionId,
						Date:             date,
						UserEmail:        record.UserEmail,
						NumSessions:      record.NumSessions,
						LinesAdded:       record.LinesAdded,
						LinesRemoved:     record.LinesRemoved,
						CommitsByClaude:  record.CommitsByClaude,
						PrsByClaude:      record.PrsByClaude,
						Model:            mb.Model,
						InputTokens:      mb.InputTokens,
						OutputTokens:     mb.OutputTokens,
						EstimatedCostUsd: mb.EstimatedCostUsd,
					}
					results = append(results, usage)
				}
				return results, nil
			}

			// Flat record – emit a single row.
			usage := &models.ClaudeUsage{
				ConnectionId:     data.Options.ConnectionId,
				Date:             date,
				UserEmail:        record.UserEmail,
				NumSessions:      record.NumSessions,
				LinesAdded:       record.LinesAdded,
				LinesRemoved:     record.LinesRemoved,
				CommitsByClaude:  record.CommitsByClaude,
				PrsByClaude:      record.PrsByClaude,
				Model:            record.Model,
				InputTokens:      record.InputTokens,
				OutputTokens:     record.OutputTokens,
				EstimatedCostUsd: record.EstimatedCostUsd,
			}
			return []interface{}{usage}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
