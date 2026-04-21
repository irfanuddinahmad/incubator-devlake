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
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

// clientSurfaceToInterfaceType maps Codex client_surface values to AiActivity InterfaceType.
var clientSurfaceToInterfaceType = map[string]string{
	"cli":         "cli",
	"ide":         "ide_plugin",
	"cloud":       "web_ui",
	"code_review": "code_review",
}

// buildCodexActivity converts a single CodexUsage tool-layer record into the
// unified AiActivity domain model.
// NumSessions maps to threads (parallel Codex tasks/sessions).
// SuggestionsCount maps to turns (conversation turns = interactions sent to the user).
func buildCodexActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, accountId string, u *models.CodexUsage) *ai.AiActivity {
	interfaceType := clientSurfaceToInterfaceType[u.ClientSurface]
	if interfaceType == "" {
		interfaceType = u.ClientSurface
	}

	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, u.ScopeId, u.Date, u.ClientSurface, u.UserEmail),
		},
		Provider:         "codex",
		AccountId:        accountId,
		UserEmail:        u.UserEmail,
		Date:             u.Date,
		Type:             "CODE_EDIT",
		InterfaceType:    interfaceType,
		NumSessions:      int(u.Threads),
		SuggestionsCount: int(u.Turns),
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
				WorkspaceId:  data.Connection.WorkspaceId,
			},
		},
		InputRowType: reflect.TypeOf(models.CodexUsage{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			u := inputRow.(*models.CodexUsage)
			accountId := resolveCodexAccountId(db, u.UserEmail)
			return []interface{}{buildCodexActivity(idGen, connectionId, accountId, u)}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}

// resolveCodexAccountId looks up the global DevLake AccountId for a given email.
// It queries the crossdomain accounts table. Returns an empty string when not found.
func resolveCodexAccountId(db dal.Dal, email string) string {
	if email == "" {
		return ""
	}
	var account crossdomain.Account
	if err := db.First(&account, dal.Where("email = ?", email)); err != nil {
		return ""
	}
	return account.Id
}
