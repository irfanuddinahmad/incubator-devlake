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
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

var ExtractUsageMeta = plugin.SubTaskMeta{
	Name:             "extractUsage",
	EntryPoint:       ExtractUsage,
	EnabledByDefault: true,
	Description:      "Extract primitive OpenAI usage API responses to the tool layer table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// openAIUsageResponse represents the payload from GET /v1/usage
type openAIUsageResponse struct {
	Object string                 `json:"object"`
	Data   []openAIUsageDataPoint `json:"data"`
}

type openAIUsageDataPoint struct {
	SnapshotId   string `json:"snapshot_id"` // format: "2024-03-01T...|model_name" or "model_name"
	NumRequests  int64  `json:"n_requests"`
	InputTokens  int64  `json:"n_context_tokens_total"`
	OutputTokens int64  `json:"n_generated_tokens_total"`
}

func ExtractUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				ProjectId:    data.Options.ProjectId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			// Extract the day string from the JSON input object
			var inputDayStr string
			if err := errors.Convert(json.Unmarshal(row.Input, &inputDayStr)); err != nil {
				return nil, errors.Default.Wrap(err, "failed to unmarshal day string from row input")
			}

			day, parseErr := time.Parse("2006-01-02", inputDayStr)
			if parseErr != nil {
				return nil, errors.Default.Wrap(parseErr, "failed to parse day from row input")
			}

			var usageResp openAIUsageResponse
			if err := errors.Convert(json.Unmarshal(row.Data, &usageResp)); err != nil {
				return nil, err
			}

			var results []interface{}
			for _, dp := range usageResp.Data {
				modelStr := dp.SnapshotId

				// Optional: mapping to exact cost could go here or in converter,
				// but OpenAI doesn't natively return $ cost in /usage — you have to map pricing.
				// We leave EstimatedCostUsd as 0 for now (future work: pricing tables).

				results = append(results, &models.CodexUsage{
					ConnectionId: data.Options.ConnectionId,
					ScopeId:      data.Options.ScopeId,
					Date:         day,
					Model:        modelStr,
					InputTokens:  dp.InputTokens,
					OutputTokens: dp.OutputTokens,
				})
			}

			return results, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
