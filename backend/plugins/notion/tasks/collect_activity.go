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
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const rawNotionActivityTable = "_raw_notion_data_source_pages"

type notionQueryResponse struct {
	Results    []json.RawMessage `json:"results"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor"`
}

var _ plugin.SubTaskEntryPoint = CollectActivity

var CollectActivityMeta = plugin.SubTaskMeta{
	Name:             "collectActivity",
	EntryPoint:       CollectActivity,
	EnabledByDefault: true,
	Description:      "Collect Notion activity events from configured Notion APIs",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func resolveNotionSince(collectedSince *time.Time, occurredAfter *time.Time, now time.Time) *time.Time {
	if collectedSince != nil && !collectedSince.IsZero() {
		t := collectedSince.UTC()
		return &t
	}
	if occurredAfter != nil {
		t := occurredAfter.UTC()
		return &t
	}
	t := now.UTC().AddDate(0, 0, -30)
	return &t
}

func buildNotionQueryRequestBody(since *time.Time, until *time.Time, pageSize int, customData interface{}) map[string]interface{} {
	body := map[string]interface{}{
		"page_size": pageSize,
		"sorts": []map[string]interface{}{{
			"timestamp": "last_edited_time",
			"direction": "ascending",
		}},
	}

	if since != nil {
		startFilter := map[string]interface{}{
			"timestamp": "last_edited_time",
			"last_edited_time": map[string]interface{}{
				"on_or_after": since.Format(time.RFC3339),
			},
		}
		if until != nil {
			body["filter"] = map[string]interface{}{
				"and": []map[string]interface{}{
					startFilter,
					{
						"timestamp": "last_edited_time",
						"last_edited_time": map[string]interface{}{
							"on_or_before": until.UTC().Format(time.RFC3339),
						},
					},
				},
			}
		} else {
			body["filter"] = startFilter
		}
	}

	if customData != nil {
		if cursor, ok := customData.(string); ok && strings.TrimSpace(cursor) != "" {
			body["start_cursor"] = cursor
		}
	}

	return body
}

func parseNotionQueryResponse(body []byte) ([]json.RawMessage, errors.Error) {
	var envelope notionQueryResponse
	if err := errors.Convert(json.Unmarshal(body, &envelope)); err != nil {
		return nil, errors.Default.Wrap(err, "failed to parse Notion query response")
	}
	return envelope.Results, nil
}

func parseNotionNextCursor(body []byte) (string, bool, errors.Error) {
	var envelope notionQueryResponse
	if err := errors.Convert(json.Unmarshal(body, &envelope)); err != nil {
		return "", false, errors.Default.Wrap(err, "failed to parse Notion pagination response")
	}
	return strings.TrimSpace(envelope.NextCursor), envelope.HasMore, nil
}

func resolveNotionNextCustomData(body []byte) (interface{}, errors.Error) {
	nextCursor, hasMore, parseErr := parseNotionNextCursor(body)
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasMore || nextCursor == "" {
		return nil, helper.ErrFinishCollect
	}
	return nextCursor, nil
}

func CollectActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*NotionTaskData)
	if !ok {
		return errors.Default.New("task data is not NotionTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	rawArgs := helper.RawDataSubTaskArgs{
		Ctx:   taskCtx,
		Table: rawNotionActivityTable,
		Options: notionRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
		},
		Params: notionRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
		},
	}

	collector, err := helper.NewStatefulApiCollector(rawArgs)
	if err != nil {
		return err
	}

	since := resolveNotionSince(collector.GetSince(), data.Options.OccurredAfter, time.Now())

	until := data.Options.OccurredBefore

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    100,
		UrlTemplate: fmt.Sprintf("v1/data_sources/%s/query", data.Options.ScopeId),
		Method:      http.MethodPost,
		RequestBody: func(reqData *helper.RequestData) map[string]interface{} {
			return buildNotionQueryRequestBody(since, until, reqData.Pager.Size, reqData.CustomData)
		},
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			body, readErr := io.ReadAll(prevPageResponse.Body)
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read Notion pagination response")
			}
			return resolveNotionNextCustomData(body)
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read Notion response body")
			}
			return parseNotionQueryResponse(body)
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
