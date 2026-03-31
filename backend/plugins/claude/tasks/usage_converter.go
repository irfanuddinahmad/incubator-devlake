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
	"github.com/apache/incubator-devlake/plugins/claude/models"
)

// ConvertUsageMeta defines the metadata for the Claude usage converter subtask.
var ConvertUsageMeta = plugin.SubTaskMeta{
	Name:             "convertUsage",
	EntryPoint:       ConvertUsage,
	EnabledByDefault: true,
	Description:      "Convert Claude Code usage records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// buildClaudeActivity converts a single ClaudeUsage tool-layer record into the
// unified AiActivity domain model. accountId may be empty when no matching
// DevLake account was found for the user email.
func buildClaudeActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, accountId string, usage *models.ClaudeUsage) *ai.AiActivity {
	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, usage.Date, usage.UserEmail),
		},
		Provider:         "claude",
		AccountId:        accountId,
		UserEmail:        usage.UserEmail,
		Date:             usage.Date,
		Type:             "CODE_EDIT",
		InterfaceType:    "cli",
		Model:            usage.Model,
		NumSessions:      usage.NumSessions,
		LinesAdded:       usage.LinesAdded,
		LinesRemoved:     usage.LinesRemoved,
		CommitsCreated:   usage.CommitsByClaude,
		PrsCreated:       usage.PrsByClaude,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		EstimatedCostUsd: usage.EstimatedCostUsd,
	}
}

// ConvertUsage maps ClaudeUsage tool-layer records to the ai_activities domain table.
// It attempts to resolve each UserEmail to a global DevLake AccountId by looking up
// the crossdomain accounts table (matched by email across all plugins).
func ConvertUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*ClaudeTaskData)
	if !ok {
		return errors.Default.New("task data is not ClaudeTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.ClaudeUsage{})

	cursor, err := db.Cursor(
		dal.From(&models.ClaudeUsage{}),
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
			Options: claudeRawParams{
				ConnectionId:   connectionId,
				ScopeId:        data.Options.ScopeId,
				OrganizationId: data.Connection.OrganizationId,
			},
		},
		InputRowType: reflect.TypeOf(models.ClaudeUsage{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			usage := inputRow.(*models.ClaudeUsage)
			accountId := resolveAccountId(db, usage.UserEmail)
			return []interface{}{buildClaudeActivity(idGen, connectionId, accountId, usage)}, nil
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
	err := db.First(&account, dal.Where("email = ?", email))
	if err != nil {
		return ""
	}
	return account.Id
}
