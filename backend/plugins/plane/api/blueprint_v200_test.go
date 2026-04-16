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

package api

import (
	"testing"

	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/srvhelper"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testScopeDetails = []*srvhelper.ScopeDetail[models.PlaneProject, models.PlaneScopeConfig]{
	{
		Scope: models.PlaneProject{
			Scope:     common.Scope{ConnectionId: 1},
			ProjectId: "project-1",
			Name:      "Project One",
		},
		ScopeConfig: &models.PlaneScopeConfig{},
	},
	{
		Scope: models.PlaneProject{
			Scope:     common.Scope{ConnectionId: 1},
			ProjectId: "project-2",
			Name:      "Project Two",
		},
		ScopeConfig: &models.PlaneScopeConfig{},
	},
}

func TestMakeDataSourcePipelinePlanV200(t *testing.T) {
	actualPlans, err := makeDataSourcePipelinePlanV200([]plugin.SubTaskMeta{}, testScopeDetails)
	assert.NoError(t, err)
	assert.Equal(t, coreModels.PipelinePlan{
		{
			{
				Plugin:   "plane",
				Subtasks: []string{},
				Options: map[string]interface{}{
					"connectionId": uint64(1),
					"projectId":    "project-1",
				},
			},
		},
		{
			{
				Plugin:   "plane",
				Subtasks: []string{},
				Options: map[string]interface{}{
					"connectionId": uint64(1),
					"projectId":    "project-2",
				},
			},
		},
	}, actualPlans)
}

func TestMakeScopesV200(t *testing.T) {
	scopes, err := makeScopesV200(testScopeDetails)
	require.NoError(t, err)
	require.Len(t, scopes, 2)

	board1, ok := scopes[0].(*ticket.Board)
	require.True(t, ok)
	assert.Equal(t, "Project One", board1.Name)

	board2, ok := scopes[1].(*ticket.Board)
	require.True(t, ok)
	assert.Equal(t, "Project Two", board2.Name)
}
