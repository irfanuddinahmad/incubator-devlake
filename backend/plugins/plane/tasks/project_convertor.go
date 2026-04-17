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
	"reflect"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

var _ plugin.SubTaskEntryPoint = ConvertProjects

var ConvertProjectsMeta = plugin.SubTaskMeta{
	Name:             "convertProjects",
	EntryPoint:       ConvertProjects,
	EnabledByDefault: true,
	Description:      "Convert Plane projects into DevLake ticket.Board domain records",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertProjects(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)
	logger := taskCtx.GetLogger()
	db := taskCtx.GetDal()
	logger.Info("convert project: %s", data.Options.ProjectId)

	idGen := didgen.NewDomainIdGenerator(&models.PlaneProject{})

	// Filter by both connection_id AND project_id to avoid cross-project contamination.
	cursor, err := db.Cursor(
		dal.Select("*"),
		dal.From(&models.PlaneProject{}),
		dal.Where("connection_id = ? AND project_id = ?", data.Options.ConnectionId, data.Options.ProjectId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := api.NewDataConverter(api.DataConverterArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_PROJECT_TABLE,
		},
		InputRowType: reflect.TypeOf(models.PlaneProject{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			project := inputRow.(*models.PlaneProject)
			boardUrl := fmt.Sprintf("%s/workspaces/%s/projects/%s",
				strings.TrimRight(data.Endpoint, "/"),
				project.WorkspaceSlug,
				project.ProjectId,
			)
			domainBoard := &ticket.Board{
				DomainEntity: domainlayer.DomainEntity{
					Id: idGen.Generate(data.Options.ConnectionId, project.ProjectId),
				},
				Name:        project.Name,
				Description: project.Description,
				Url:         boardUrl,
			}
			return []interface{}{domainBoard}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
