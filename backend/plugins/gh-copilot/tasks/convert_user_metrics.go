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
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/gh-copilot/models"
)

// ConvertUserMetricsMeta defines the metadata for the Copilot per-user converter subtask.
var ConvertUserMetricsMeta = plugin.SubTaskMeta{
	Name:             "convertUserMetrics",
	EntryPoint:       ConvertUserMetrics,
	EnabledByDefault: true,
	Description:      "Convert GhCopilotUserDailyMetrics records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// ConvertUserMetrics maps per-user daily Copilot metrics to the unified ai_activities table.
// Each GhCopilotUserDailyMetrics row becomes one AiActivity row keyed by (connectionId, day, userId).
//
// Field mapping:
//   - CodeGenerationActivityCount → SuggestionsCount (suggestions shown)
//   - CodeAcceptanceActivityCount → AcceptanceCount  (suggestions accepted)
//   - LocAddedSum                 → LinesAdded
//   - LocDeletedSum               → LinesRemoved
//   - UserLogin                   → UserEmail (best available identifier; AccountId resolved where possible)
func ConvertUserMetrics(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*GhCopilotTaskData)
	if !ok {
		return errors.Default.New("task data is not GhCopilotTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.GhCopilotUserDailyMetrics{})

	cursor, err := db.Cursor(
		dal.From(&models.GhCopilotUserDailyMetrics{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUserMetricsTable,
			Options: copilotRawParams{
				ConnectionId: connectionId,
				ScopeId:      data.Options.ScopeId,
				Organization: data.Connection.Organization,
				Endpoint:     data.Connection.Endpoint,
			},
		},
		InputRowType: reflect.TypeOf(models.GhCopilotUserDailyMetrics{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			m := inputRow.(*models.GhCopilotUserDailyMetrics)

			// Resolve UserLogin → global AccountId via crossdomain accounts (email match).
			// Copilot reports logins, not emails, so resolution may not succeed — that is acceptable.
			accountId := resolveAccountByLogin(db, m.UserLogin)

			activity := &ai.AiActivity{
				DomainEntity: domainlayer.DomainEntity{
					Id: idGen.Generate(connectionId, m.Day.Format("2006-01-02"), m.UserId),
				},
				Provider:         "gh-copilot",
				AccountId:        accountId,
				UserEmail:        m.UserLogin, // best available; may be a GitHub login
				Date:             m.Day,
				Type:             "CODE_EDIT",
				InterfaceType:    "ide_plugin",
				NumSessions:      m.UserInitiatedInteractionCount,
				SuggestionsCount: m.CodeGenerationActivityCount,
				AcceptanceCount:  m.CodeAcceptanceActivityCount,
				LinesAdded:       m.LocAddedSum,
				LinesRemoved:     m.LocDeletedSum,
			}
			return []interface{}{activity}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}

// resolveAccountByLogin looks up a global DevLake AccountId for a given GitHub login.
// It queries the crossdomain accounts table matching on the Name field.
// Returns an empty string when not found.
func resolveAccountByLogin(db dal.Dal, login string) string {
	if login == "" {
		return ""
	}
	var account crossdomain.Account
	if err := db.First(&account, dal.Where("name = ?", login)); err != nil {
		return ""
	}
	return account.Id
}
