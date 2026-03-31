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
	Description:      "Collect daily usage metrics from the Codex (OpenAI) API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
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

	// Collect last 30 days by default if missing
	endTime := time.Now()
	if data.Options.EndDate != nil {
		endTime = *data.Options.EndDate
	}
	startTime := endTime.AddDate(0, 0, -30)
	if data.Options.StartDate != nil {
		startTime = *data.Options.StartDate
	}

	// Generate a slice of all days in [startTime, endTime]
	var days []string
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		days = append(days, d.Format("2006-01-02"))
	}

	// OpenAI API returns one record object per day for /v1/usage
	collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				ProjectId:    data.Options.ProjectId,
			},
		},
		ApiClient:   apiClient,
		Concurrency: 5,
		Input:       &StringSliceIterator{elements: days},
		UrlTemplate: "usage",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			day := reqData.Input.(string)
			query.Set("date", day)
			// project_id is optional but required if scoped to a project
			if data.Options.ProjectId != "" {
				query.Set("project_id", data.Options.ProjectId)
			}
			return query, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			// The OpenAI usage API returns a single object containing `data` array representing items.
			// Because we query one day at a time, we just store the top-level response.
			// e.g. {"object": "list", "data": [...]}
			// We can pass the whole body down to the extractor.
			var result json.RawMessage
			if err := helper.UnmarshalResponse(res, &result); err != nil {
				return nil, err
			}
			return []json.RawMessage{result}, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}

// StringSliceIterator implements plugin.Iterator for a slice of strings.
type StringSliceIterator struct {
	elements []string
	index    int
}

func (s *StringSliceIterator) HasNext() bool {
	return s.index < len(s.elements)
}

func (s *StringSliceIterator) Fetch() (interface{}, errors.Error) {
	if s.index >= len(s.elements) {
		return nil, errors.Default.New("iterator out of bounds")
	}
	val := s.elements[s.index]
	s.index++
	return val, nil
}

func (s *StringSliceIterator) Close() errors.Error {
	return nil
}
