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
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var _ plugin.SubTaskEntryPoint = ExtractWorkItemTypes

var ExtractWorkItemTypesMeta = plugin.SubTaskMeta{
	Name:             "extractWorkItemTypes",
	EntryPoint:       ExtractWorkItemTypes,
	EnabledByDefault: true,
	Description:      "Extract Plane work item types into the tool layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractWorkItemTypes(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_WORK_ITEM_TYPE_TABLE,
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			workItemType, err := extractPlaneWorkItemType(row.Data, data.Options.ConnectionId, data.Options.ProjectId)
			if err != nil {
				return nil, err
			}
			return []interface{}{workItemType}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
