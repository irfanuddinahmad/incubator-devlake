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
	"io"
	"net/http"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const rawDailyUsageTable = "cursor_daily_usage"

// CollectDailyUsageMeta defines metadata for the daily-usage collector subtask.
var CollectDailyUsageMeta = plugin.SubTaskMeta{
	Name:             "collectDailyUsage",
	EntryPoint:       CollectDailyUsage,
	EnabledByDefault: true,
	Description:      "Collect Cursor daily usage data per user from POST /teams/daily-usage-data",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorDailyUsagePagination is the pagination object in the daily-usage-data response.
type cursorDailyUsagePagination struct {
	Page            int  `json:"page"`
	PageSize        int  `json:"pageSize"`
	TotalUsers      int  `json:"totalUsers"`
	TotalPages      int  `json:"totalPages"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

// cursorDailyUsageResponse is the envelope returned by POST /teams/daily-usage-data.
type cursorDailyUsageResponse struct {
	Data       []json.RawMessage          `json:"data"`
	Period     map[string]interface{}     `json:"period"`
	Pagination cursorDailyUsagePagination `json:"pagination"`
}

// CollectDailyUsage fetches per-user daily usage metrics from POST /teams/daily-usage-data
// with incremental sync support.
func CollectDailyUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}
	connection := data.Connection
	connection.Normalize()

	apiClient, err := createApiClient(taskCtx.TaskContext(), connection)
	if err != nil {
		return err
	}

	teamId := data.Options.TeamId

	rawArgs := helper.RawDataSubTaskArgs{
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
	}

	collector, err := helper.NewStatefulApiCollector(rawArgs)
	if err != nil {
		return err
	}

	since := collector.GetSince()
	var startDate time.Time
	if since != nil && !since.IsZero() {
		startDate = since.UTC()
	} else {
		startDate = time.Now().UTC().AddDate(0, 0, -30)
	}
	endDate := time.Now().UTC().AddDate(0, 0, 1)
	maxStartDate := endDate.AddDate(0, 0, -30)
	if startDate.Before(maxStartDate) {
		startDate = maxStartDate
	}
	startMs := startDate.UnixMilli()
	endMs := endDate.UnixMilli()

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    1000, // max page size for daily-usage-data
		Method:      http.MethodPost,
		UrlTemplate: "teams/daily-usage-data",
		RequestBody: func(reqData *helper.RequestData) map[string]interface{} {
			return map[string]interface{}{
				"startDate": startMs,
				"endDate":   endMs,
				"page":      reqData.Pager.Page,
				"pageSize":  reqData.Pager.Size,
			}
		},
		GetTotalPages: func(res *http.Response, args *helper.ApiCollectorArgs) (int, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			if readErr != nil {
				return 0, errors.Default.Wrap(readErr, "failed to read pagination response")
			}
			var envelope cursorDailyUsageResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return 0, errors.Default.Wrap(jsonErr, "failed to parse pagination response")
			}
			if envelope.Pagination.TotalPages == 0 {
				return 1, nil
			}
			return envelope.Pagination.TotalPages, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read Cursor daily-usage response")
			}
			var envelope cursorDailyUsageResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Cursor daily-usage response")
			}
			return envelope.Data, nil
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
