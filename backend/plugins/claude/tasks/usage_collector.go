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

const rawUsageTable = "claude_usage"

// CollectUsageMeta defines the metadata for the Claude usage collector subtask.
var CollectUsageMeta = plugin.SubTaskMeta{
	Name:             "collectUsage",
	EntryPoint:       CollectUsage,
	EnabledByDefault: true,
	Description:      "Collect Claude Code daily usage reports from the Anthropic Admin API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// claudeUsageResponse models the top-level Anthropic API response envelope.
type claudeUsageResponse struct {
	Data     []json.RawMessage `json:"data"`
	HasMore  bool              `json:"has_more"`
	NextPage string            `json:"next_page"`
	FirstID  string            `json:"first_id"`
	LastID   string            `json:"last_id"`
}

// CollectUsage fetches daily Claude Code usage records from the Anthropic Admin API,
// supporting both pagination (via `next_page` cursor) and incremental sync
// (via `starting_at` parameter derived from the last successful sync date).
func CollectUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*ClaudeTaskData)
	if !ok {
		return errors.Default.New("task data is not ClaudeTaskData")
	}
	connection := data.Connection
	connection.Normalize()

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), connection)
	if err != nil {
		return err
	}

	// The Admin API endpoint does not take an org ID in the path — the Admin
	// API key implicitly scopes the request to the caller's organisation.
	// Reference: https://platform.claude.com/docs/en/build-with-claude/claude-code-analytics-api
	urlPath := "organizations/usage_report/claude_code"

	// For raw-data deduplication we still store the org ID if available.
	orgId := connection.OrganizationId
	if orgId == "" {
		orgId = data.Options.OrganizationId
	}

	rawArgs := helper.RawDataSubTaskArgs{
		Ctx:   taskCtx,
		Table: rawUsageTable,
		Options: claudeRawParams{
			ConnectionId:   data.Options.ConnectionId,
			ScopeId:        data.Options.ScopeId,
			OrganizationId: orgId,
		},
		Params: claudeRawParams{
			ConnectionId:   data.Options.ConnectionId,
			ScopeId:        data.Options.ScopeId,
			OrganizationId: orgId,
		},
	}

	collector, err := helper.NewStatefulApiCollector(rawArgs)
	if err != nil {
		return err
	}

	// Determine the starting date for incremental sync.
	since := collector.GetSince()
	var startingAt time.Time
	if since != nil && !since.IsZero() {
		startingAt = since.UTC()
	} else {
		// Default to 90 days of history on a full sync.
		startingAt = time.Now().UTC().AddDate(0, 0, -90)
	}

	endingBefore := time.Now().UTC().AddDate(0, 0, 1) // tomorrow inclusive

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    1_000_000, // large value; pagination is cursor-driven via ErrFinishCollect
		UrlTemplate: urlPath,
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			q := url.Values{}
			q.Set("starting_at", startingAt.Format("2006-01-02"))
			q.Set("ending_before", endingBefore.Format("2006-01-02"))
			// Pass cursor on subsequent pages.
			if reqData.CustomData != nil {
				if cursor, ok := reqData.CustomData.(string); ok && cursor != "" {
					q.Set("after_id", cursor)
				}
			}
			return q, nil
		},
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			body, readErr := io.ReadAll(prevPageResponse.Body)
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read pagination response")
			}
			var envelope claudeUsageResponse
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
				return nil, errors.Default.Wrap(readErr, "failed to read Claude usage response body")
			}

			var envelope claudeUsageResponse
			if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
				return nil, errors.Default.Wrap(jsonErr, "failed to parse Claude usage response")
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

// claudeRawParams identifies the raw data scope for Claude usage records.
type claudeRawParams struct {
	ConnectionId   uint64 `json:"connectionId"`
	ScopeId        string `json:"scopeId"`
	OrganizationId string `json:"organizationId"`
}

// GetParams implements helper.TaskOptions.
func (p claudeRawParams) GetParams() any {
	return p
}
