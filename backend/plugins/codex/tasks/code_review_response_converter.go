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

// Note: resolveCodexAccountId is defined in usage_converter.go

// buildCodexCodeReviewResponseActivity maps a CodexCodeReviewResponse to AiActivity.
// Type = "CODE_REVIEW_RESPONSE"; AcceptanceCount = upvotes (positive reactions).
func buildCodexCodeReviewResponseActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, accountId string, r *models.CodexCodeReviewResponse) *ai.AiActivity {
	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, r.ScopeId, r.Date, r.UserEmail),
		},
		Provider:      "codex",
		AccountId:     accountId,
		UserEmail:     r.UserEmail,
		Date:          r.Date,
		Type:          "CODE_REVIEW_RESPONSE",
		InterfaceType: "code_review",
		// SuggestionsCount captures total engagement events (replies + upvotes + downvotes).
		SuggestionsCount: int(r.Replies + r.Upvotes + r.Downvotes),
		// AcceptanceCount captures positive reactions (upvotes = accepted/agreed).
		AcceptanceCount: int(r.Upvotes),
	}
}

var ConvertCodeReviewResponsesMeta = plugin.SubTaskMeta{
	Name:             "convertCodeReviewResponses",
	EntryPoint:       ConvertCodeReviewResponses,
	EnabledByDefault: true,
	Description:      "Convert CodexCodeReviewResponse records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func ConvertCodeReviewResponses(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.CodexCodeReviewResponse{})

	cursor, err := db.Cursor(
		dal.From(&models.CodexCodeReviewResponse{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawCodeReviewResponseTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				WorkspaceId:  data.Connection.WorkspaceId,
			},
		},
		InputRowType: reflect.TypeOf(models.CodexCodeReviewResponse{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			r := inputRow.(*models.CodexCodeReviewResponse)
			accountId := resolveCodexAccountId(db, r.UserEmail)
			return []interface{}{buildCodexCodeReviewResponseActivity(idGen, connectionId, accountId, r)}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
