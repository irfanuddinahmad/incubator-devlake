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

const rawCodeReviewTable = "_raw_codex_code_reviews"

var CollectCodeReviewsMeta = plugin.SubTaskMeta{
	Name:             "collectCodeReviews",
	EntryPoint:       CollectCodeReviews,
	EnabledByDefault: true,
	Description:      "Collect Codex code-review activity from the Analytics API (/workspaces/{id}/code_reviews)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// codexCodeReviewsEnvelope is the top-level response from
// GET /analytics/codex/workspaces/{id}/code_reviews.
type codexCodeReviewsEnvelope struct {
	Data     []codexCodeReviewRecord `json:"data"`
	HasMore  bool                    `json:"has_more"`
	NextPage string                  `json:"next_page"`
}

// codexCodeReviewRecord represents one code-review record for a single PR on a given day.
type codexCodeReviewRecord struct {
	Date              string        `json:"date"`             // "2024-01-15"
	PrUrl             string        `json:"pull_request_url"` // canonical PR URL
	ReviewsCompleted  int64         `json:"reviews_completed"`
	CommentsGenerated int64         `json:"comments_generated"`
	Severity          codexSeverity `json:"severity"`
}

// codexSeverity holds the comment breakdown by priority level.
type codexSeverity struct {
	Low      int64 `json:"low"`
	Medium   int64 `json:"medium"`
	High     int64 `json:"high"`
	Critical int64 `json:"critical"`
}

func CollectCodeReviews(taskCtx plugin.SubTaskContext) errors.Error {
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
		Table: rawCodeReviewTable,
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

	urlTemplate := fmt.Sprintf("analytics/codex/workspaces/%s/code_reviews", workspaceId)

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
				return nil, errors.Default.Wrap(readErr, "failed to read code_reviews pagination response")
			}
			var envelope codexCodeReviewsEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse code_reviews pagination response")
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
				return nil, errors.Default.Wrap(readErr, "failed to read Codex code_reviews response body")
			}
			var envelope codexCodeReviewsEnvelope
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Codex code_reviews response")
			}

			var records []json.RawMessage
			for _, record := range envelope.Data {
				raw, marshalErr := json.Marshal(record)
				if marshalErr != nil {
					return nil, errors.Default.Wrap(marshalErr, "failed to marshal code_review record")
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
