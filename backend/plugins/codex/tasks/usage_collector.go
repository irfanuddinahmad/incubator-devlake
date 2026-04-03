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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const rawUsageTable = "_raw_codex_usage"

var CollectUsageMeta = plugin.SubTaskMeta{
	Name:             "collectUsage",
	EntryPoint:       CollectUsage,
	EnabledByDefault: true,
	Description:      "Collect daily usage metrics from the Codex Analytics API (/workspaces/{id}/usage)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// codexUsageEnvelope is the top-level response from GET /analytics/codex/workspaces/{id}/usage.
type codexUsageEnvelope struct {
	Data     []codexUsageRecord `json:"data"`
	HasMore  bool               `json:"has_more"`
	NextPage string             `json:"next_page"`
}

// codexUsageRecord represents one daily usage record for a workspace/user/surface combination.
type codexUsageRecord struct {
	Date          string  `json:"date"`           // "2024-01-15"
	ClientSurface string  `json:"client_surface"` // "cli", "ide", "cloud", "code_review"
	UserEmail     string  `json:"user_email"`     // empty when not per-user
	Threads       int64   `json:"threads"`
	Turns         int64   `json:"turns"`
	Credits       float64 `json:"credits"`
}

func CollectUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	workspaceId := data.Connection.WorkspaceId
	if workspaceId == "" {
		return errors.Default.New("workspaceId is required for Codex Analytics API")
	}

	rawArgs := helper.RawDataSubTaskArgs{
		Ctx:   taskCtx,
		Table: rawUsageTable,
		Options: codexRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
			WorkspaceId:  workspaceId,
		},
		Params: codexRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
			WorkspaceId:  workspaceId,
		},
	}

	collector, err := helper.NewStatefulApiCollector(rawArgs)
	if err != nil {
		return err
	}

	// Build time range for the query.
	endTime := time.Now().UTC()
	if data.Options.EndDate != nil {
		endTime = data.Options.EndDate.UTC()
	}
	startTime := endTime.AddDate(0, 0, -30)
	if since := collector.GetSince(); since != nil && !since.IsZero() {
		startTime = since.UTC()
	} else if data.Options.StartDate != nil {
		startTime = data.Options.StartDate.UTC()
	}

	urlTemplate := fmt.Sprintf("analytics/codex/workspaces/%s/usage", workspaceId)

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    100,
		UrlTemplate: urlTemplate,
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			q := url.Values{}
			q.Set("start_time", startTime.Format(time.RFC3339))
			q.Set("end_time", endTime.Format(time.RFC3339))
			// Request per-user breakdown for maximum data granularity.
			q.Set("per_user", "true")
			if reqData.CustomData != nil {
				if cursor, ok := reqData.CustomData.(string); ok && cursor != "" {
					q.Set("next_page", cursor)
				}
			}
			return q, nil
		},
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			body, readErr := io.ReadAll(prevPageResponse.Body)
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read usage pagination response")
			}
			var envelope codexUsageEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse usage pagination response")
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
				return nil, errors.Default.Wrap(readErr, "failed to read Codex usage response body")
			}
			var envelope codexUsageEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Codex usage response")
			}

			var records []json.RawMessage
			for _, record := range envelope.Data {
				raw, marshalErr := json.Marshal(record)
				if marshalErr != nil {
					return nil, errors.Default.Wrap(marshalErr, "failed to marshal usage record")
				}
				records = append(records, raw)
			}
			return records, nil
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
