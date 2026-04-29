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

var _ plugin.SubTaskEntryPoint = ExtractCycleItems

var ExtractCycleItemsMeta = plugin.SubTaskMeta{
	Name:             "extractCycleItems",
	EntryPoint:       ExtractCycleItems,
	EnabledByDefault: true,
	Description:      "Extract Plane cycle work-item membership into the tool layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractCycleItems(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	db := taskCtx.GetDal()

	cycles, err := loadPlaneCycles(db, data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}

	for _, cycle := range cycles {
		items, err := gatherCycleItems(taskCtx, data, cycle.CycleId)
		if err != nil {
			return err
		}
		if err := clearPlaneCycleItems(db, data.Options.ConnectionId, data.Options.ProjectId, cycle.CycleId); err != nil {
			return err
		}
		for _, item := range items {
			if err := db.CreateOrUpdate(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func gatherCycleItems(taskCtx plugin.SubTaskContext, data *PlaneTaskData, cycleId string) ([]*models.PlaneCycleItem, errors.Error) {
	var items []*models.PlaneCycleItem
	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneCycleItemApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
				CycleId:       cycleId,
			},
			Table: RAW_CYCLE_ITEM_TABLE,
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			item, err := extractPlaneCycleItem(row.Data, data.Options.ConnectionId, data.Options.ProjectId, cycleId)
			if err != nil {
				return nil, err
			}
			if item != nil {
				items = append(items, item)
			}
			return nil, nil
		},
	})
	if err != nil {
		return nil, err
	}
	if err := extractor.Execute(); err != nil {
		return nil, err
	}
	return items, nil
}

func clearPlaneCycleItems(
	db dal.Dal,
	connectionId uint64,
	projectId string,
	cycleId string,
) errors.Error {
	return db.Delete(
		&models.PlaneCycleItem{},
		dal.Where("connection_id = ? AND project_id = ? AND cycle_id = ?", connectionId, projectId, cycleId),
	)
}
