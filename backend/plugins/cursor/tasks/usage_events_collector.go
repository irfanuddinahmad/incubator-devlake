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

const rawUsageEventsTable = "cursor_usage_events"

// CollectUsageEventsMeta defines metadata for the usage-events collector subtask.
var CollectUsageEventsMeta = plugin.SubTaskMeta{
	Name:             "collectUsageEvents",
	EntryPoint:       CollectUsageEvents,
	EnabledByDefault: true,
	Description:      "Collect Cursor individual AI request events from POST /teams/filtered-usage-events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorUsageEventsPagination is the pagination object in the filtered-usage-events response.
type cursorUsageEventsPagination struct {
	NumPages        int  `json:"numPages"`
	CurrentPage     int  `json:"currentPage"`
	PageSize        int  `json:"pageSize"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

// cursorUsageEventsResponse is the envelope returned by POST /teams/filtered-usage-events.
type cursorUsageEventsResponse struct {
	TotalUsageEventsCount int                         `json:"totalUsageEventsCount"`
	Pagination            cursorUsageEventsPagination `json:"pagination"`
	UsageEvents           []json.RawMessage           `json:"usageEvents"`
	Period                map[string]interface{}      `json:"period"`
}

// CollectUsageEvents fetches individual AI request events from POST /teams/filtered-usage-events
// with incremental sync support.
func CollectUsageEvents(taskCtx plugin.SubTaskContext) errors.Error {
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
		startDate = time.Now().UTC().AddDate(0, 0, -90)
	}
	endDate := time.Now().UTC().AddDate(0, 0, 1)

	// Cursor admin API uses epoch milliseconds for date ranges.
	startMs := startDate.UnixMilli()
	endMs := endDate.UnixMilli()

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    500,
		Method:      http.MethodPost,
		UrlTemplate: "teams/filtered-usage-events",
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
			var envelope cursorUsageEventsResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return 0, errors.Default.Wrap(jsonErr, "failed to parse pagination response")
			}
			if envelope.Pagination.NumPages == 0 {
				return 1, nil
			}
			return envelope.Pagination.NumPages, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read Cursor usage-events response")
			}
			var envelope cursorUsageEventsResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Cursor usage-events response")
			}
			return envelope.UsageEvents, nil
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
