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
	"testing"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPlaneCycle(t *testing.T) {
	cycle, err := extractPlaneCycle([]byte(`{
		"id": "cycle-1",
		"name": "Cycle A",
		"description": "Sprint replacement",
		"status": "current",
		"start_date": "2024-02-01T00:00:00Z",
		"end_date": "2024-02-14T00:00:00Z",
		"completed_at": "2024-02-14T18:30:00Z",
		"created_at": "2024-01-25T10:00:00Z",
		"updated_at": "2024-02-10T11:00:00Z"
	}`), 7, "project-1")
	require.NoError(t, err)
	require.NotNil(t, cycle)

	assert.Equal(t, uint64(7), cycle.ConnectionId)
	assert.Equal(t, "project-1", cycle.ProjectId)
	assert.Equal(t, "cycle-1", cycle.CycleId)
	assert.Equal(t, "Cycle A", cycle.Name)
	assert.Equal(t, "Sprint replacement", cycle.Description)
	assert.Equal(t, "current", cycle.Status)
	require.NotNil(t, cycle.StartDate)
	assert.Equal(t, mustParsePlaneTime(t, "2024-02-01T00:00:00Z"), cycle.StartDate)
	require.NotNil(t, cycle.EndDate)
	assert.Equal(t, mustParsePlaneTime(t, "2024-02-14T00:00:00Z"), cycle.EndDate)
	require.NotNil(t, cycle.CompletedAt)
	assert.Equal(t, mustParsePlaneTime(t, "2024-02-14T18:30:00Z"), cycle.CompletedAt)
}

func TestExtractPlaneCycleHandlesNullDescription(t *testing.T) {
	cycle, err := extractPlaneCycle([]byte(`{
		"id": "cycle-1",
		"name": "Cycle A",
		"description": null,
		"status": "draft"
	}`), 7, "project-1")
	require.NoError(t, err)
	require.NotNil(t, cycle)
	assert.Equal(t, "", cycle.Description)
	assert.Equal(t, "draft", cycle.Status)
}

func TestExtractPlaneCycleAcceptsTimestampDates(t *testing.T) {
	cycle, err := extractPlaneCycle([]byte(`{
		"id": "cycle-2",
		"name": "Cycle B",
		"description": "",
		"status": "current",
		"start_date": "2026-04-21T17:11:30.247304Z",
		"end_date": "2026-04-28T17:11:30.247304Z"
	}`), 7, "project-1")
	require.NoError(t, err)
	require.NotNil(t, cycle)
	require.NotNil(t, cycle.StartDate)
	require.NotNil(t, cycle.EndDate)
	assert.Equal(t, mustParsePlaneTime(t, "2026-04-21T17:11:30.247304Z"), cycle.StartDate)
	assert.Equal(t, mustParsePlaneTime(t, "2026-04-28T17:11:30.247304Z"), cycle.EndDate)
}

func TestExtractPlaneCycleItem(t *testing.T) {
	cycleItem, err := extractPlaneCycleItem([]byte(`{
		"id": "membership-1",
		"cycle": "cycle-1",
		"issue": "item-42",
		"created_at": "2024-02-02T09:00:00Z",
		"updated_at": "2024-02-03T12:00:00Z"
	}`), 7, "project-1", "cycle-1")
	require.NoError(t, err)
	require.NotNil(t, cycleItem)

	assert.Equal(t, uint64(7), cycleItem.ConnectionId)
	assert.Equal(t, "project-1", cycleItem.ProjectId)
	assert.Equal(t, "cycle-1", cycleItem.CycleId)
	assert.Equal(t, "item-42", cycleItem.ItemId)
	assert.Equal(t, planeCycleItemTypeWorkItem, cycleItem.ItemType)
	assert.Equal(t, mustParsePlaneTime(t, "2024-02-02T09:00:00Z"), cycleItem.CreatedDate)
	assert.Equal(t, mustParsePlaneTime(t, "2024-02-03T12:00:00Z"), cycleItem.UpdatedDate)
}

func TestExtractPlaneCycleItemSkipsMismatchedCycle(t *testing.T) {
	cycleItem, err := extractPlaneCycleItem([]byte(`{
		"id": "membership-1",
		"cycle": "cycle-from-response",
		"issue": "item-42"
	}`), 7, "project-1", "cycle-from-collector")
	require.NoError(t, err)
	assert.Nil(t, cycleItem)
}

func TestExtractPlaneCycleItemSkipsEmptyIssue(t *testing.T) {
	cycleItem, err := extractPlaneCycleItem([]byte(`{"id":"membership-1","cycle":"cycle-1"}`), 7, "project-1", "cycle-1")
	require.NoError(t, err)
	assert.Nil(t, cycleItem)
}

func TestClearPlaneCycleItems(t *testing.T) {
	spy := &planeCycleSpyDal{}
	err := clearPlaneCycleItems(spy, 7, "project-1", "cycle-1")
	require.NoError(t, err)

	require.Len(t, spy.deleteClauses, 1)
	assert.IsType(t, &models.PlaneCycleItem{}, spy.deleteEntity)
}

func TestLoadPlaneCycles(t *testing.T) {
	expected := []models.PlaneCycle{
		{ConnectionId: 7, ProjectId: "project-1", CycleId: "cycle-1"},
		{ConnectionId: 7, ProjectId: "project-1", CycleId: "cycle-2"},
	}
	spy := &loadCyclesSpyDal{returnCycles: expected}
	cycles, err := loadPlaneCycles(spy, 7, "project-1")
	require.NoError(t, err)
	assert.Equal(t, expected, cycles)
}

type planeCycleSpyDal struct {
	dal.Dal
	deleteEntity  interface{}
	deleteClauses [][]dal.Clause
}

func (d *planeCycleSpyDal) Delete(entity interface{}, clauses ...dal.Clause) errors.Error {
	d.deleteEntity = entity
	d.deleteClauses = append(d.deleteClauses, clauses)
	return nil
}

type loadCyclesSpyDal struct {
	dal.Dal
	returnCycles []models.PlaneCycle
}

func (d *loadCyclesSpyDal) All(out interface{}, _ ...dal.Clause) errors.Error {
	*out.(*[]models.PlaneCycle) = d.returnCycles
	return nil
}
