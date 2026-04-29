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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

const planeCyclePageSize = 100

const planeCycleItemTypeWorkItem = "work_item"

type planeApiCycle struct {
	Id          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	StartDate   string     `json:"start_date"`
	EndDate     string     `json:"end_date"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type planeApiCycleItem struct {
	Id        string     `json:"id"`
	Cycle     planeApiId `json:"cycle"`
	Issue     planeApiId `json:"issue"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func extractPlaneCycle(data []byte, connectionId uint64, projectId string) (*models.PlaneCycle, errors.Error) {
	var apiCycle planeApiCycle
	if err := json.Unmarshal(data, &apiCycle); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane cycle")
	}
	description := ""
	if apiCycle.Description != nil {
		description = *apiCycle.Description
	}
	startDate, err := parsePlaneDate(apiCycle.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parsePlaneDate(apiCycle.EndDate)
	if err != nil {
		return nil, err
	}
	return &models.PlaneCycle{
		ConnectionId: connectionId,
		ProjectId:    projectId,
		CycleId:      apiCycle.Id,
		Name:         apiCycle.Name,
		Description:  description,
		Status:       apiCycle.Status,
		StartDate:    startDate,
		EndDate:      endDate,
		CompletedAt:  apiCycle.CompletedAt,
		CreatedDate:  apiCycle.CreatedAt,
		UpdatedDate:  apiCycle.UpdatedAt,
	}, nil
}

func extractPlaneCycleItem(data []byte, connectionId uint64, projectId, cycleId string) (*models.PlaneCycleItem, errors.Error) {
	var apiCycleItem planeApiCycleItem
	if err := json.Unmarshal(data, &apiCycleItem); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane cycle item")
	}
	// Plane's cycle-issues endpoint returns full work item objects directly,
	// so the issue ID is the top-level "id". Older API versions may nest it
	// under "issue", so we check that first.
	issueId := apiCycleItem.Issue.Id
	if issueId == "" {
		issueId = apiCycleItem.Id
	}
	if issueId == "" {
		return nil, nil
	}
	if apiCycleItem.Cycle.Id != "" && apiCycleItem.Cycle.Id != cycleId {
		return nil, nil
	}
	return &models.PlaneCycleItem{
		ConnectionId: connectionId,
		ProjectId:    projectId,
		CycleId:      cycleId,
		ItemId:       issueId,
		ItemType:     planeCycleItemTypeWorkItem,
		CreatedDate:  apiCycleItem.CreatedAt,
		UpdatedDate:  apiCycleItem.UpdatedAt,
	}, nil
}

func clearPlaneCycles(db dal.Dal, connectionId uint64, projectId string) errors.Error {
	return db.Delete(
		&models.PlaneCycle{},
		dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId),
	)
}

func loadPlaneCycles(db dal.Dal, connectionId uint64, projectId string) ([]models.PlaneCycle, errors.Error) {
	var cycles []models.PlaneCycle
	if err := db.All(
		&cycles,
		dal.From(&models.PlaneCycle{}),
		dal.Where("connection_id = ? AND project_id = ?", connectionId, projectId),
		dal.Orderby("start_date ASC, created_date ASC, cycle_id ASC"),
	); err != nil {
		return nil, err
	}
	return cycles, nil
}
