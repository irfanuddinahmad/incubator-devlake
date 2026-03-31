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

// ExtractDailyUsageMeta defines metadata for the daily-usage extractor subtask.
var ExtractDailyUsageMeta = plugin.SubTaskMeta{
	Name:             "extractDailyUsage",
	EntryPoint:       ExtractDailyUsage,
	EnabledByDefault: true,
	Description:      "Extract Cursor daily usage records into _tool_cursor_daily_usage",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorDailyUsageRow is the raw JSON row from the Cursor daily-usage endpoint.
type cursorDailyUsageRow struct {
	Date               string `json:"date"`
	UserEmail          string `json:"userEmail"`
	TotalTabsShown     int    `json:"totalTabsShown"`
	TotalTabsAccepted  int    `json:"totalTabsAccepted"`
	TotalLinesAdded    int    `json:"totalLinesAdded"`
	AcceptedLinesAdded int    `json:"acceptedLinesAdded"`
	TotalLinesDeleted  int    `json:"totalLinesDeleted"`
}

// ExtractDailyUsage reads raw cursor_daily_usage records and writes
// strongly-typed CursorDailyUsage rows to the tool layer.
func ExtractDailyUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}

	teamId := data.Options.TeamId

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawDailyUsageTable,
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
			var rawRow cursorDailyUsageRow
			if err := json.Unmarshal(row.Data, &rawRow); err != nil {
				return nil, errors.Default.Wrap(err, "failed to unmarshal cursor daily-usage row")
			}

			day, parseErr := time.Parse("2006-01-02", rawRow.Date)
			if parseErr != nil {
				// Try RFC3339 fallback
				day, parseErr = time.Parse(time.RFC3339, rawRow.Date)
				if parseErr != nil {
					return nil, errors.Default.Wrap(parseErr, "failed to parse date: "+rawRow.Date)
				}
			}

			record := &models.CursorDailyUsage{
				ConnectionId:       data.Options.ConnectionId,
				ScopeId:            data.Options.ScopeId,
				Day:                day,
				UserEmail:          rawRow.UserEmail,
				TotalTabsShown:     rawRow.TotalTabsShown,
				TotalTabsAccepted:  rawRow.TotalTabsAccepted,
				TotalLinesAdded:    rawRow.TotalLinesAdded,
				AcceptedLinesAdded: rawRow.AcceptedLinesAdded,
				TotalLinesDeleted:  rawRow.TotalLinesDeleted,
				NoPKModel: common.NoPKModel{
					RawDataOrigin: common.RawDataOrigin{
						RawDataTable:  rawDailyUsageTable,
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
