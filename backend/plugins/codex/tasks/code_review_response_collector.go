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

const rawCodeReviewResponseTable = "_raw_codex_code_review_responses"

var CollectCodeReviewResponsesMeta = plugin.SubTaskMeta{
	Name:             "collectCodeReviewResponses",
	EntryPoint:       CollectCodeReviewResponses,
	EnabledByDefault: true,
	Description:      "Collect user engagement with Codex code reviews from the Analytics API (/workspaces/{id}/code_review_responses)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// codexCodeReviewResponsesEnvelope is the top-level response from
// GET /analytics/codex/workspaces/{id}/code_review_responses.
type codexCodeReviewResponsesEnvelope struct {
	Data     []codexCodeReviewResponseRecord `json:"data"`
	HasMore  bool                            `json:"has_more"`
	NextPage string                          `json:"next_page"`
}

// codexCodeReviewResponseRecord represents one user-engagement record for a given day.
type codexCodeReviewResponseRecord struct {
	Date      string `json:"date"`       // "2024-01-15"
	UserEmail string `json:"user_email"` // the user who reacted/replied
	Replies   int64  `json:"replies"`
	Upvotes   int64  `json:"upvotes"`
	Downvotes int64  `json:"downvotes"`
}

func CollectCodeReviewResponses(taskCtx plugin.SubTaskContext) errors.Error {
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
		Table: rawCodeReviewResponseTable,
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

	urlTemplate := fmt.Sprintf("analytics/codex/workspaces/%s/code_review_responses", workspaceId)

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    100,
		UrlTemplate: urlTemplate,
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			q := url.Values{}
			q.Set("start_time", startTime.Format(time.RFC3339))
			q.Set("end_time", endTime.Format(time.RFC3339))
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
				return nil, errors.Default.Wrap(readErr, "failed to read code_review_responses pagination response")
			}
			var envelope codexCodeReviewResponsesEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse code_review_responses pagination response")
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
				return nil, errors.Default.Wrap(readErr, "failed to read Codex code_review_responses response body")
			}
			var envelope codexCodeReviewResponsesEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Codex code_review_responses response")
			}

			var records []json.RawMessage
			for _, record := range envelope.Data {
				raw, marshalErr := json.Marshal(record)
				if marshalErr != nil {
					return nil, errors.Default.Wrap(marshalErr, "failed to marshal code_review_response record")
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
