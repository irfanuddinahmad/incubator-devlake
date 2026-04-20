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

var _ plugin.SubTaskEntryPoint = ConvertWorkItems

var ConvertWorkItemsMeta = plugin.SubTaskMeta{
	Name:             "convertWorkItems",
	EntryPoint:       ConvertWorkItems,
	EnabledByDefault: true,
	Description:      "Convert Plane work items into DevLake ticket domain records",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertWorkItems(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	db := taskCtx.GetDal()

	issueIdGen := didgen.NewDomainIdGenerator(&models.PlaneWorkItem{})
	boardIdGen := didgen.NewDomainIdGenerator(&models.PlaneProject{})
	boardId := boardIdGen.Generate(data.Options.ConnectionId, data.Options.ProjectId)

	converter, err := api.NewStatefulDataConverter(&api.StatefulDataConverterArgs[models.PlaneWorkItem]{
		SubtaskCommonArgs: &api.SubtaskCommonArgs{
			SubTaskContext: taskCtx,
			Table:          RAW_WORK_ITEM_TABLE,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
		},
		Input: func(stateManager *api.SubtaskStateManager) (dal.Rows, errors.Error) {
			clauses := []dal.Clause{
				dal.Select("*"),
				dal.From(&models.PlaneWorkItem{}),
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
		Convert: func(workItem *models.PlaneWorkItem) ([]any, errors.Error) {
			issue := &ticket.Issue{
				DomainEntity: domainlayer.DomainEntity{
					Id: issueIdGen.Generate(workItem.ConnectionId, workItem.ProjectId, workItem.WorkItemId),
				},
				Url:            buildPlaneWorkItemURL(data.Endpoint, data.Project.WorkspaceSlug, data.Project.Identifier, workItem.SequenceId),
				IssueKey:       fmt.Sprintf("#%d", workItem.SequenceId),
				Title:          workItem.Title,
				Description:    workItem.Description,
				Type:           planeWorkItemTypeToStandardType(workItem.TypeName),
				OriginalType:   workItem.TypeName,
				Status:         planeStateGroupToStandardStatus(workItem.StateGroup),
				OriginalStatus: workItem.StateName,
				Priority:       workItem.Priority,
				AssigneeId:     workItem.AssigneeId,
				AssigneeName:   workItem.AssigneeName,
				StoryPoint:     workItem.EstimatePoint,
				CreatedDate:    workItem.CreatedDate,
				UpdatedDate:    workItem.UpdatedDate,
				ResolutionDate: workItem.CompletedAt,
				LeadTimeMinutes: computePlaneLeadTimeMinutes(
					workItem.CreatedDate,
					workItem.CompletedAt,
				),
			}
			if workItem.ParentId != nil && *workItem.ParentId != "" {
				issue.ParentIssueId = issueIdGen.Generate(workItem.ConnectionId, workItem.ProjectId, *workItem.ParentId)
				issue.IsSubtask = true
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
