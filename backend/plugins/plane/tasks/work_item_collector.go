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
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var _ plugin.SubTaskEntryPoint = CollectWorkItems

var CollectWorkItemsMeta = plugin.SubTaskMeta{
	Name:             "collectWorkItems",
	EntryPoint:       CollectWorkItems,
	EnabledByDefault: true,
	Description:      "Collect Plane work items from the remote API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectWorkItems(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	taskCtx.GetLogger().Info(planeUpdatedAtOrderingVerification)

	collector, err := api.NewStatefulApiCollector(api.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: PlaneApiParams{
			ConnectionId:  data.Options.ConnectionId,
			WorkspaceSlug: data.Project.WorkspaceSlug,
			ProjectId:     data.Options.ProjectId,
		},
		Table: RAW_WORK_ITEM_TABLE,
	})
	if err != nil {
		return err
	}

	err = collector.InitCollector(api.ApiCollectorArgs{
		ApiClient:   data.ApiClient,
		PageSize:    planeWorkItemPageSize,
		UrlTemplate: "api/v1/workspaces/{{ .Params.WorkspaceSlug }}/projects/{{ .Params.ProjectId }}/work-items/",
		Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("per_page", strconv.Itoa(planeWorkItemPageSize))
			if cursor, ok := reqData.CustomData.(string); ok && cursor != "" {
				query.Set("cursor", cursor)
			}
			if since := collector.GetSince(); since != nil {
				query.Set("updated_at__gte", since.UTC().Format(time.RFC3339))
			}
			return query, nil
		},
		GetNextPageCustomData: func(_ *api.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			return parsePlaneNextCursor(prevPageResponse)
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			return parsePlaneWorkItemResultsForCollector(res, collector.GetSince())
		},
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
