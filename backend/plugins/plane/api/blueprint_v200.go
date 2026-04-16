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
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/helpers/srvhelper"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/apache/incubator-devlake/plugins/plane/tasks"
)

func MakeDataSourcePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	connectionId uint64,
	bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	_, err := dsHelper.ConnSrv.FindByPk(connectionId)
	if err != nil {
		return nil, nil, err
	}
	scopeDetails, err := dsHelper.ScopeSrv.MapScopeDetails(connectionId, bpScopes)
	if err != nil {
		return nil, nil, err
	}

	plan, err := makeDataSourcePipelinePlanV200(subtaskMetas, scopeDetails)
	if err != nil {
		return nil, nil, err
	}
	scopes, err := makeScopesV200(scopeDetails)
	if err != nil {
		return nil, nil, err
	}
	return plan, scopes, nil
}

func makeScopesV200(
	scopeDetails []*srvhelper.ScopeDetail[models.PlaneProject, models.PlaneScopeConfig],
) ([]plugin.Scope, errors.Error) {
	scopes := make([]plugin.Scope, 0, len(scopeDetails))
	idGen := didgen.NewDomainIdGenerator(&models.PlaneProject{})
	for _, scopeDetail := range scopeDetails {
		project := scopeDetail.Scope
		scopes = append(scopes, &ticket.Board{
			DomainEntity: domainlayer.DomainEntity{
				Id: idGen.Generate(project.ConnectionId, project.ProjectId),
			},
			Name: project.ScopeName(),
		})
	}
	return scopes, nil
}

func makeDataSourcePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	scopeDetails []*srvhelper.ScopeDetail[models.PlaneProject, models.PlaneScopeConfig],
) (coreModels.PipelinePlan, errors.Error) {
	plan := make(coreModels.PipelinePlan, len(scopeDetails))
	for i, scopeDetail := range scopeDetails {
		stage := coreModels.PipelineStage{}

		scope := scopeDetail.Scope
		if scope.ProjectId == "" {
			return nil, errors.BadInput.New("scope is missing ProjectId")
		}
		task, err := helper.MakePipelinePlanTask(
			"plane",
			subtaskMetas,
			nil,
			tasks.PlaneOptions{
				ConnectionId: scope.ConnectionId,
				ProjectId:    scope.ProjectId,
			},
		)
		if err != nil {
			return nil, err
		}

		stage = append(stage, task)
		plan[i] = stage
	}
	return plan, nil
}
