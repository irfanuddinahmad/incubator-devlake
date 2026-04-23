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

var _ plugin.SubTaskEntryPoint = ExtractEpics

var ExtractEpicsMeta = plugin.SubTaskMeta{
	Name:             "extractEpics",
	EntryPoint:       ExtractEpics,
	EnabledByDefault: true,
	Description:      "Extract Plane epics into the tool layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
	Dependencies:     []*plugin.SubTaskMeta{&ExtractStatesMeta, &ExtractEstimatesMeta, &ExtractWorkItemTypesMeta},
}

func ExtractEpics(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	stateMap, err := loadPlaneStateMap(taskCtx.GetDal(), data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}
	workItemTypeMap, err := loadPlaneWorkItemTypeMap(taskCtx.GetDal(), data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}
	estimateMap, err := loadPlaneEstimatePointMap(taskCtx.GetDal(), data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}

	extractor, err := api.NewStatefulApiExtractor(&api.StatefulApiExtractorArgs[planeApiEpic]{
		SubtaskCommonArgs: &api.SubtaskCommonArgs{
			SubTaskContext: taskCtx,
			Table:          RAW_EPIC_TABLE,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
		},
		Extract: func(body *planeApiEpic, _ *api.RawData) ([]any, errors.Error) {
			epic, err := mapPlaneEpic(
				body,
				data.Options.ConnectionId,
				data.Options.ProjectId,
				stateMap,
				workItemTypeMap,
				estimateMap,
			)
			if err != nil {
				return nil, err
			}
			return []interface{}{epic}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
