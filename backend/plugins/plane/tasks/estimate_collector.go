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
	"strconv"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const planeEstimatePageSize = 100

var _ plugin.SubTaskEntryPoint = CollectEstimates

var CollectEstimatesMeta = plugin.SubTaskMeta{
	Name:             "collectEstimates",
	EntryPoint:       CollectEstimates,
	EnabledByDefault: true,
	Description:      "Collect Plane project estimates from the remote API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectEstimates(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_ESTIMATE_TABLE,
		},
		ApiClient:     data.ApiClient,
		PageSize:      planeEstimatePageSize,
		AfterResponse: ignoreHTTPStatus404,
		UrlTemplate:   "api/v1/workspaces/{{ .Params.WorkspaceSlug }}/projects/{{ .Params.ProjectId }}/estimates/",
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("limit", strconv.Itoa(planeEstimatePageSize))
			query.Set("per_page", strconv.Itoa(planeEstimatePageSize))
			switch custom := reqData.CustomData.(type) {
			case string:
				if custom != "" {
					query.Set("cursor", custom)
				}
			case int:
				if custom > 0 {
					query.Set("offset", strconv.Itoa(custom))
				}
			}
			return query, nil
		},
		GetNextPageCustomData: func(reqData *api.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			currentOffset := 0
			if offset, ok := reqData.CustomData.(int); ok {
				currentOffset = offset
			}
			return parsePlaneEstimateNextPage(prevPageResponse, currentOffset, reqData.Pager.Size)
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			results, err := parsePlaneEstimateResults(res)
			if err != nil {
				return nil, err
			}
			return enrichPlaneEstimateResults(results, data.ApiClient, data.Project.WorkspaceSlug, data.Options.ProjectId)
		},
	})
	if err != nil {
		return err
	}
	return collector.Execute()
}
