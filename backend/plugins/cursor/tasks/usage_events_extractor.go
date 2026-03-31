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
	Description:      "Extract Cursor usage event records into _tool_cursor_usage_events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorUsageEventRow is the raw JSON row from the filtered-usage-events endpoint.
type cursorUsageEventRow struct {
	EventId      string  `json:"eventId"`
	Timestamp    string  `json:"timestamp"`
	UserEmail    string  `json:"userEmail"`
	Model        string  `json:"model"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	RequestCost  float64 `json:"requestCost"`
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

			var ts time.Time
			var parseErr error
			ts, parseErr = time.Parse(time.RFC3339, rawRow.Timestamp)
			if parseErr != nil {
				ts, parseErr = time.Parse("2006-01-02", rawRow.Timestamp)
				if parseErr != nil {
					return nil, errors.Default.Wrap(parseErr, "failed to parse timestamp: "+rawRow.Timestamp)
				}
			}

			eventId := rawRow.EventId
			if eventId == "" {
				// Generate a stable surrogate key from params
				eventId = rawRow.UserEmail + "_" + rawRow.Timestamp + "_" + rawRow.Model
			}

			record := &models.CursorUsageEvent{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				EventId:      eventId,
				Timestamp:    ts,
				UserEmail:    rawRow.UserEmail,
				Model:        rawRow.Model,
				InputTokens:  rawRow.InputTokens,
				OutputTokens: rawRow.OutputTokens,
				RequestCost:  rawRow.RequestCost,
				NoPKModel: common.NoPKModel{
					RawDataOrigin: common.RawDataOrigin{
						RawDataTable:  rawUsageEventsTable,
						RawDataParams: row.Params,
						RawDataId:     row.ID,
					},
				},
			}
			return []interface{}{record}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
