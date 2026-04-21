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
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// ConvertDailyUsageMeta defines the metadata for the Cursor daily-usage converter subtask.
var ConvertDailyUsageMeta = plugin.SubTaskMeta{
	Name:             "convertDailyUsage",
	EntryPoint:       ConvertDailyUsage,
	EnabledByDefault: true,
	Description:      "Convert CursorDailyUsage records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// buildCursorDailyActivity converts a single CursorDailyUsage tool-layer record into
// the unified AiActivity domain model.
//
// Field mapping:
//   - TotalTabsShown    → SuggestionsCount  (autocomplete tabs suggested)
//   - TotalTabsAccepted → AcceptanceCount   (autocomplete tabs accepted)
//   - AcceptedLinesAdded → LinesAdded
//   - TotalLinesDeleted → LinesRemoved
//   - ComposerRequests+ChatRequests+AgentRequests → NumSessions
//   - MostUsedModel     → Model
func buildCursorDailyActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, accountId string, u *models.CursorDailyUsage) *ai.AiActivity {
	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, u.ScopeId, u.Day, u.UserEmail),
		},
		Provider:         "cursor",
		AccountId:        accountId,
		UserEmail:        u.UserEmail,
		Date:             u.Day,
		Type:             "CODE_EDIT",
		InterfaceType:    "ide_plugin",
		SuggestionsCount: u.TotalTabsShown,
		AcceptanceCount:  u.TotalTabsAccepted,
		LinesAdded:       u.AcceptedLinesAdded,
		LinesRemoved:     u.TotalLinesDeleted,
		NumSessions:      u.ComposerRequests + u.ChatRequests + u.AgentRequests,
		Model:            u.MostUsedModel,
	}
}

// ConvertDailyUsage maps per-user daily Cursor usage to the unified ai_activities table.
//
// Field mapping:
//   - TotalTabsShown    → SuggestionsCount  (autocomplete tabs suggested)
//   - TotalTabsAccepted → AcceptanceCount   (autocomplete tabs accepted)
//   - TotalLinesAdded   → LinesAdded
//   - TotalLinesDeleted → LinesRemoved
func ConvertDailyUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.CursorDailyUsage{})

	cursor, err := db.Cursor(
		dal.From(&models.CursorDailyUsage{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawDailyUsageTable,
			Options: cursorRawParams{
				ConnectionId: connectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       data.Options.TeamId,
			},
		},
		InputRowType: reflect.TypeOf(models.CursorDailyUsage{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			u := inputRow.(*models.CursorDailyUsage)
			accountId := resolveAccountId(db, u.UserEmail)
			return []interface{}{buildCursorDailyActivity(idGen, connectionId, accountId, u)}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}

// resolveAccountId looks up the global DevLake AccountId for a given email.
// It queries the crossdomain accounts table. Returns an empty string when not found.
func resolveAccountId(db dal.Dal, email string) string {
	if email == "" {
		return ""
	}
	var account crossdomain.Account
	if err := db.First(&account, dal.Where("email = ?", email)); err != nil {
		return ""
	}
	return account.Id
}
