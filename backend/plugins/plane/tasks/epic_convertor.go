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
	"fmt"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

var _ plugin.SubTaskEntryPoint = ConvertEpics

var ConvertEpicsMeta = plugin.SubTaskMeta{
	Name:             "convertEpics",
	EntryPoint:       ConvertEpics,
	EnabledByDefault: true,
	Description:      "Convert Plane epics into DevLake ticket domain records",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertEpics(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	db := taskCtx.GetDal()

	epicIdGen := didgen.NewDomainIdGenerator(&models.PlaneEpic{})
	boardIdGen := didgen.NewDomainIdGenerator(&models.PlaneProject{})
	boardId := boardIdGen.Generate(data.Options.ConnectionId, data.Options.ProjectId)
	epicIDSet, err := loadPlaneEpicIDSet(db, data.Options.ConnectionId, data.Options.ProjectId)
	if err != nil {
		return err
	}

	converter, err := api.NewStatefulDataConverter(&api.StatefulDataConverterArgs[models.PlaneEpic]{
		SubtaskCommonArgs: &api.SubtaskCommonArgs{
			SubTaskContext: taskCtx,
			Table:          RAW_EPIC_TABLE,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
		},
		Input: func(stateManager *api.SubtaskStateManager) (dal.Rows, errors.Error) {
			clauses := []dal.Clause{
				dal.Select("*"),
				dal.From(&models.PlaneEpic{}),
				dal.Where("connection_id = ? AND project_id = ?", data.Options.ConnectionId, data.Options.ProjectId),
			}
			if stateManager.IsIncremental() {
				since := stateManager.GetSince()
				if since != nil {
					clauses = append(clauses, dal.Where("updated_at >= ?", since))
				}
			}
			return db.Cursor(clauses...)
		},
		Convert: func(epic *models.PlaneEpic) ([]any, errors.Error) {
			issue := &ticket.Issue{
				DomainEntity: domainlayer.DomainEntity{
					Id: epicIdGen.Generate(epic.ConnectionId, epic.ProjectId, epic.EpicId),
				},
				Url:            buildPlaneEpicURL(data.Endpoint, data.Project.WorkspaceSlug, data.Project.Identifier, epic.SequenceId),
				IssueKey:       fmt.Sprintf("#%d", epic.SequenceId),
				Title:          epic.Name,
				Description:    epic.Description,
				Type:           "EPIC",
				OriginalType:   "Epic",
				Status:         planeStateGroupToStandardStatus(epic.StateGroup),
				OriginalStatus: epic.StateName,
				Priority:       epic.Priority,
				AssigneeId:     epic.AssigneeId,
				AssigneeName:   epic.AssigneeName,
				StoryPoint:     planeEpicStoryPoint(epic),
				CreatedDate:    epic.CreatedDate,
				UpdatedDate:    epic.UpdatedDate,
				ResolutionDate: epic.CompletedAt,
				LeadTimeMinutes: computePlaneLeadTimeMinutes(
					epic.CreatedDate,
					epic.CompletedAt,
				),
			}
			if epic.ParentId != nil && *epic.ParentId != "" {
				// Plane epics currently point only to other epics; we intentionally skip cross-entity fallback here.
				if _, ok := epicIDSet[*epic.ParentId]; ok {
					issue.ParentIssueId = epicIdGen.Generate(epic.ConnectionId, epic.ProjectId, *epic.ParentId)
				}
			}
			boardIssue := &ticket.BoardIssue{
				BoardId: boardId,
				IssueId: issue.Id,
			}
			return []interface{}{issue, boardIssue}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
