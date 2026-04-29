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
	"strconv"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helperapi "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

const resolvedEstimateNilKey = "__nil__"

var _ plugin.SubTaskEntryPoint = ResolveWorkItemEstimates

var ResolveWorkItemEstimatesMeta = plugin.SubTaskMeta{
	Name:             "resolveWorkItemEstimates",
	EntryPoint:       ResolveWorkItemEstimates,
	EnabledByDefault: true,
	Description:      "Resolve Plane work-item estimate UUIDs into numeric story points",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ResolveWorkItemEstimates(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	db := taskCtx.GetDal()

	estimateMap, err := loadPlaneEstimatePointMap(db, data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}
	rawRows, err := loadPlaneRawWorkItemRows(taskCtx, data)
	if err != nil {
		return err
	}
	groupedIds, groupedValues, err := collectResolvedWorkItemEstimates(rawRows, estimateMap)
	if err != nil {
		return err
	}
	for key, workItemIds := range groupedIds {
		if err := db.UpdateColumns(
			&models.PlaneWorkItem{},
			[]dal.DalSet{{ColumnName: "estimate_point", Value: groupedValues[key]}},
			dal.Where("connection_id = ? AND project_id = ? AND work_item_id IN ?", data.Options.ConnectionId, data.Options.ProjectId, workItemIds),
		); err != nil {
			return err
		}
	}
	return nil
}

func loadPlaneRawWorkItemRows(taskCtx plugin.SubTaskContext, data *PlaneTaskData) ([]helperapi.RawData, errors.Error) {
	rawDataSubTask, err := helperapi.NewRawDataSubTask(helperapi.RawDataSubTaskArgs{
		Ctx:   taskCtx,
		Table: RAW_WORK_ITEM_TABLE,
		Params: &PlaneApiParams{
			ConnectionId:  data.Options.ConnectionId,
			WorkspaceSlug: data.Project.WorkspaceSlug,
			ProjectId:     data.Options.ProjectId,
		},
	})
	if err != nil {
		return nil, err
	}

	var rawRows []helperapi.RawData
	if err := taskCtx.GetDal().All(
		&rawRows,
		dal.From(rawDataSubTask.GetTable()),
		dal.Where("params = ?", rawDataSubTask.GetParams()),
	); err != nil {
		return nil, err
	}
	return rawRows, nil
}

func collectResolvedWorkItemEstimates(rawRows []helperapi.RawData, estimateMap map[string]*float64) (map[string][]string, map[string]interface{}, errors.Error) {
	latestRows, err := latestPlaneRawWorkItemRowsByID(rawRows)
	if err != nil {
		return nil, nil, err
	}
	groupedIds := make(map[string][]string)
	groupedValues := make(map[string]interface{})
	for workItemId, row := range latestRows {
		resolved := resolveEstimateForRow(row.Data, estimateMap)
		key := planeResolvedEstimateGroupKey(resolved)
		groupedIds[key] = append(groupedIds[key], workItemId)
		groupedValues[key] = resolved
	}
	return groupedIds, groupedValues, nil
}

func latestPlaneRawWorkItemRowsByID(rawRows []helperapi.RawData) (map[string]helperapi.RawData, errors.Error) {
	latestRows := make(map[string]helperapi.RawData, len(rawRows))
	for _, row := range rawRows {
		workItemId, err := extractPlaneWorkItemID(row.Data)
		if err != nil {
			return nil, err
		}
		if workItemId == "" {
			continue
		}
		latestRow, ok := latestRows[workItemId]
		if !ok || row.ID > latestRow.ID {
			latestRows[workItemId] = row
		}
	}
	return latestRows, nil
}

func extractPlaneWorkItemID(data []byte) (string, errors.Error) {
	var payload struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", errors.Default.Wrap(err, "error unmarshalling Plane work item identity")
	}
	return payload.Id, nil
}

func resolveEstimateForRow(data []byte, estimateMap map[string]*float64) interface{} {
	rawEstimatePoint := extractPlaneRawEstimatePointValue(data)
	if rawEstimatePoint == "" {
		return nil
	}
	if mappedEstimate, ok := estimateMap[rawEstimatePoint]; ok {
		return mappedEstimate
	}
	if parsedEstimate, _ := parsePlaneEstimatePointValue(rawEstimatePoint); parsedEstimate != nil {
		return parsedEstimate
	}
	return nil
}

func planeResolvedEstimateGroupKey(resolved interface{}) string {
	if resolved == nil {
		return resolvedEstimateNilKey
	}
	value, ok := resolved.(*float64)
	if !ok || value == nil {
		return resolvedEstimateNilKey
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}
