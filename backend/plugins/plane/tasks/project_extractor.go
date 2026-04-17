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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

var _ plugin.SubTaskEntryPoint = ExtractProjects

var ExtractProjectsMeta = plugin.SubTaskMeta{
	Name:             "extractProjects",
	EntryPoint:       ExtractProjects,
	EnabledByDefault: true,
	Description:      "Extract Plane project metadata into the tool layer",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractProjects(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*PlaneTaskData)

	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: PlaneApiParams{
				ConnectionId:  data.Options.ConnectionId,
				WorkspaceSlug: data.Project.WorkspaceSlug,
				ProjectId:     data.Options.ProjectId,
			},
			Table: RAW_PROJECT_TABLE,
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			// Plane single-project API response shape
			var apiProject struct {
				Id          string `json:"id"`
				Name        string `json:"name"`
				Identifier  string `json:"identifier"`
				Description string `json:"description"`
				Network     int    `json:"network"`
			}
			if err := json.Unmarshal(row.Data, &apiProject); err != nil {
				return nil, errors.Default.Wrap(err, "error unmarshalling Plane project")
			}

			project := &models.PlaneProject{
				Scope: common.Scope{
					ConnectionId:  data.Options.ConnectionId,
					ScopeConfigId: data.Project.ScopeConfigId, // preserve existing scope config binding
				},
				ProjectId:     apiProject.Id,
				Name:          apiProject.Name,
				Identifier:    apiProject.Identifier,
				Description:   apiProject.Description,
				Network:       apiProject.Network,
				WorkspaceSlug: data.Project.WorkspaceSlug,
			}

			return []interface{}{project}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
