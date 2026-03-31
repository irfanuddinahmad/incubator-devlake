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
	"reflect"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ai"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

// buildCodexActivity converts a single CodexUsage tool-layer record into the
// unified AiActivity domain model.
// UserEmail and AccountId are intentionally empty: the OpenAI /v1/usage endpoint
// only provides project-level aggregates, not per-user breakdowns.
func buildCodexActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, u *models.CodexUsage) *ai.AiActivity {
	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, u.ScopeId, u.Date, u.Model),
		},
		Provider:         "codex",
		Date:             u.Date,
		Model:            u.Model,
		Type:             "CODE_EDIT",
		InterfaceType:    "cli",
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		EstimatedCostUsd: u.EstimatedCostUsd,
	}
}

var ConvertUsageMeta = plugin.SubTaskMeta{
	Name:             "convertUsage",
	EntryPoint:       ConvertUsage,
	EnabledByDefault: true,
	Description:      "Convert CodexUsage records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func ConvertUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.CodexUsage{})

	cursor, err := db.Cursor(
		dal.From(&models.CodexUsage{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				ProjectId:    data.Options.ProjectId,
			},
		},
		InputRowType: reflect.TypeOf(models.CodexUsage{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			u := inputRow.(*models.CodexUsage)
			return []interface{}{buildCodexActivity(idGen, connectionId, u)}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
