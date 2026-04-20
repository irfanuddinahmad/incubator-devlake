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

var _ plugin.SubTaskEntryPoint = CollectWorkItemTypes

var CollectWorkItemTypesMeta = plugin.SubTaskMeta{
	Name:             "collectWorkItemTypes",
	EntryPoint:       CollectWorkItemTypes,
	EnabledByDefault: true,
	Description:      "Collect Plane project work item types from the remote API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectWorkItemTypes(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	collector, err := api.NewApiCollector(api.ApiCollectorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_WORK_ITEM_TYPE_TABLE,
		},
		ApiClient:   data.ApiClient,
		PageSize:    100,
		UrlTemplate: "api/v1/workspaces/{{ .Params.WorkspaceSlug }}/projects/{{ .Params.ProjectId }}/work-item-types/",
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("limit", "100")
			if offset, ok := reqData.CustomData.(int); ok && offset > 0 {
				query.Set("offset", strconv.Itoa(offset))
			}
			return query, nil
		},
		GetNextPageCustomData: func(prevReqData *api.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			var result []json.RawMessage
			if err := api.UnmarshalResponse(prevPageResponse, &result); err != nil {
				return nil, err
			}
			offset := 0
			if currentOffset, ok := prevReqData.CustomData.(int); ok {
				offset = currentOffset
			}
			return offset + prevReqData.Pager.Size, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var result []json.RawMessage
			if err := api.UnmarshalResponse(res, &result); err != nil {
				return nil, err
			}
			return result, nil
		},
	})
	if err != nil {
		return err
	}
	return collector.Execute()
}
