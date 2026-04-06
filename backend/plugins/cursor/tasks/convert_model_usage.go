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
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/ai"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// ConvertModelUsageMeta defines the metadata for the Cursor usage-events converter subtask.
var ConvertModelUsageMeta = plugin.SubTaskMeta{
	Name:             "convertModelUsage",
	EntryPoint:       ConvertModelUsage,
	EnabledByDefault: true,
	Description:      "Convert CursorUsageEvent records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// ConvertModelUsage maps per-user individual AI request events to the unified ai_activities table.
//
// Field mapping:
//   - Model          → Model
//   - Timestamp      → Date (truncated to day)
//   - InputTokens    → InputTokens
//   - OutputTokens   → OutputTokens
//   - ChargedCents   → EstimatedCostUsd (divided by 100)
func ConvertModelUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.CursorUsageEvent{})

	cursor, err := db.Cursor(
		dal.From(&models.CursorUsageEvent{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageEventsTable,
			Options: cursorRawParams{
				ConnectionId: connectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       data.Options.TeamId,
			},
		},
		InputRowType: reflect.TypeOf(models.CursorUsageEvent{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			u := inputRow.(*models.CursorUsageEvent)
			accountId := resolveAccountId(db, u.UserEmail)
			activity := &ai.AiActivity{
				DomainEntity: domainlayer.DomainEntity{
					Id: idGen.Generate(connectionId, u.ScopeId, u.Timestamp, u.UserEmail, u.Model),
				},
				Provider:         "cursor",
				AccountId:        accountId,
				UserEmail:        u.UserEmail,
				Date:             u.Timestamp.Truncate(24 * time.Hour),
				Model:            u.Model,
				Type:             "CODE_EDIT",
				InterfaceType:    "ide_plugin",
				InputTokens:      u.InputTokens,
				OutputTokens:     u.OutputTokens,
				EstimatedCostUsd: u.ChargedCents / 100.0,
			}
			return []interface{}{activity}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
