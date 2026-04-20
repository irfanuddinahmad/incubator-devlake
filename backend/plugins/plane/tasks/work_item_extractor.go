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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

var _ plugin.SubTaskEntryPoint = ExtractWorkItems

var ExtractWorkItemsMeta = plugin.SubTaskMeta{
	Name:             "extractWorkItems",
	EntryPoint:       ExtractWorkItems,
	EnabledByDefault: true,
	Description:      "Extract Plane work items into the tool layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractWorkItems(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	db := taskCtx.GetDal()

	stateMap, err := loadPlaneStateMap(db, data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}
	workItemTypeMap, err := loadPlaneWorkItemTypeMap(db, data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_WORK_ITEM_TABLE,
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			workItem, err := extractPlaneWorkItem(
				row.Data,
				data.Options.ConnectionId,
				data.Options.ProjectId,
				stateMap,
				workItemTypeMap,
			)
			if err != nil {
				return nil, err
			}
			return []interface{}{workItem}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}

func loadPlaneStateMap(db dal.Dal, connectionId uint64, projectId string) (map[string]models.PlaneState, errors.Error) {
	var states []models.PlaneState
	if err := db.All(&states, dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId)); err != nil {
		return nil, err
	}
	stateMap := make(map[string]models.PlaneState, len(states))
	for _, state := range states {
		stateMap[state.StateId] = state
	}
	return stateMap, nil
}

func loadPlaneWorkItemTypeMap(db dal.Dal, connectionId uint64, projectId string) (map[string]models.PlaneWorkItemType, errors.Error) {
	var workItemTypes []models.PlaneWorkItemType
	if err := db.All(&workItemTypes, dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId)); err != nil {
		return nil, err
	}
	workItemTypeMap := make(map[string]models.PlaneWorkItemType, len(workItemTypes))
	for _, workItemType := range workItemTypes {
		workItemTypeMap[workItemType.TypeId] = workItemType
	}
	return workItemTypeMap, nil
}
