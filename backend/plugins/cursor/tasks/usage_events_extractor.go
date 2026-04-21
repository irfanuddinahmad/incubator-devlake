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
	"strconv"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// ExtractUsageEventsMeta defines metadata for the usage-events extractor subtask.
var ExtractUsageEventsMeta = plugin.SubTaskMeta{
	Name:             "extractUsageEvents",
	EntryPoint:       ExtractUsageEvents,
	EnabledByDefault: true,
	Description:      "Extract Cursor individual AI request events into _tool_cursor_usage_events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorTokenUsageDetail holds the optional token breakdown within a usage event.
type cursorTokenUsageDetail struct {
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	CacheWriteTokens int64   `json:"cacheWriteTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens"`
	TotalCents       float64 `json:"totalCents"`
}

// cursorUsageEventRow is the raw JSON row from POST /teams/filtered-usage-events.
type cursorUsageEventRow struct {
	Timestamp        string                  `json:"timestamp"` // epoch ms as string
	UserEmail        string                  `json:"userEmail"`
	Model            string                  `json:"model"`
	Kind             string                  `json:"kind"`
	MaxMode          bool                    `json:"maxMode"`
	RequestsCosts    float64                 `json:"requestsCosts"`
	IsTokenBasedCall bool                    `json:"isTokenBasedCall"`
	IsChargeable     bool                    `json:"isChargeable"`
	IsHeadless       bool                    `json:"isHeadless"`
	TokenUsage       *cursorTokenUsageDetail `json:"tokenUsage"`
	ChargedCents     float64                 `json:"chargedCents"`
}

// ExtractUsageEvents reads raw cursor_usage_events records and writes
// strongly-typed CursorUsageEvent rows to the tool layer.
func ExtractUsageEvents(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}

	teamId := data.Options.TeamId

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageEventsTable,
			Options: cursorRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       teamId,
			},
			Params: cursorRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       teamId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var rawRow cursorUsageEventRow
			if err := json.Unmarshal(row.Data, &rawRow); err != nil {
				return nil, errors.Default.Wrap(err, "failed to unmarshal cursor usage-event row")
			}

			tsMs, parseErr := strconv.ParseInt(rawRow.Timestamp, 10, 64)
			if parseErr != nil {
				return nil, errors.Default.Wrap(parseErr, "failed to parse timestamp: "+rawRow.Timestamp)
			}
			ts := time.UnixMilli(tsMs).UTC()

			record := &models.CursorUsageEvent{
				ConnectionId:     data.Options.ConnectionId,
				ScopeId:          data.Options.ScopeId,
				Timestamp:        ts,
				UserEmail:        rawRow.UserEmail,
				Model:            rawRow.Model,
				Kind:             rawRow.Kind,
				MaxMode:          rawRow.MaxMode,
				RequestsCosts:    rawRow.RequestsCosts,
				IsTokenBasedCall: rawRow.IsTokenBasedCall,
				IsChargeable:     rawRow.IsChargeable,
				IsHeadless:       rawRow.IsHeadless,
				ChargedCents:     rawRow.ChargedCents,
				NoPKModel: common.NoPKModel{
					RawDataOrigin: common.RawDataOrigin{
						RawDataTable:  rawUsageEventsTable,
						RawDataParams: row.Params,
						RawDataId:     row.ID,
					},
				},
			}
			if rawRow.TokenUsage != nil {
				record.InputTokens = rawRow.TokenUsage.InputTokens
				record.OutputTokens = rawRow.TokenUsage.OutputTokens
				record.CacheWriteTokens = rawRow.TokenUsage.CacheWriteTokens
				record.CacheReadTokens = rawRow.TokenUsage.CacheReadTokens
				record.TotalCents = rawRow.TokenUsage.TotalCents
			}
			return []interface{}{record}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
