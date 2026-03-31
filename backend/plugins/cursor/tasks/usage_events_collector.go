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
	"net/url"
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
	Description:      "Collect Cursor raw usage events (model, tokens, cost) from GET /teams/filtered-usage-events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorUsageEventsResponse is the envelope returned by GET /teams/filtered-usage-events.
type cursorUsageEventsResponse struct {
	Data     []json.RawMessage `json:"data"`
	HasMore  bool              `json:"hasMore"`
	NextPage string            `json:"nextPage"`
}

// CollectUsageEvents fetches raw usage events from the Cursor API.
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

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    1_000_000,
		UrlTemplate: "teams/filtered-usage-events",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			q := url.Values{}
			q.Set("startDate", startDate.Format("2006-01-02"))
			q.Set("endDate", endDate.Format("2006-01-02"))
			if reqData.CustomData != nil {
				if cursor, ok := reqData.CustomData.(string); ok && cursor != "" {
					q.Set("cursor", cursor)
				}
			}
			return q, nil
		},
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			body, readErr := io.ReadAll(prevPageResponse.Body)
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read pagination response")
			}
			var envelope cursorUsageEventsResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse pagination response")
			}
			if !envelope.HasMore || envelope.NextPage == "" {
				return nil, helper.ErrFinishCollect
			}
			return envelope.NextPage, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read Cursor usage-events response")
			}
			var envelope cursorUsageEventsResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr == nil && envelope.Data != nil {
				return envelope.Data, nil
			}
			var rows []json.RawMessage
			if jsonErr := json.Unmarshal(body, &rows); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Cursor usage-events response as array")
			}
			return rows, nil
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
