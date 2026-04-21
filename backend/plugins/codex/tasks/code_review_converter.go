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

// buildCodexCodeReviewActivity maps a CodexCodeReview tool record to AiActivity.
// Type = "CODE_REVIEW"; SuggestionsCount = total comments Codex generated (review suggestions).
func buildCodexCodeReviewActivity(idGen *didgen.DomainIdGenerator, connectionId uint64, cr *models.CodexCodeReview) *ai.AiActivity {
	return &ai.AiActivity{
		DomainEntity: domainlayer.DomainEntity{
			Id: idGen.Generate(connectionId, cr.ScopeId, cr.Date, cr.PrUrl),
		},
		Provider:         "codex",
		Date:             cr.Date,
		Type:             "CODE_REVIEW",
		InterfaceType:    "code_review",
		SuggestionsCount: int(cr.CommentsGenerated),
	}
}

var ConvertCodeReviewsMeta = plugin.SubTaskMeta{
	Name:             "convertCodeReviews",
	EntryPoint:       ConvertCodeReviews,
	EnabledByDefault: true,
	Description:      "Convert CodexCodeReview records into DevLake's ai_activities domain table",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func ConvertCodeReviews(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	idGen := didgen.NewDomainIdGenerator(&models.CodexCodeReview{})

	cursor, err := db.Cursor(
		dal.From(&models.CodexCodeReview{}),
		dal.Where("connection_id = ?", connectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawCodeReviewTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				WorkspaceId:  data.Connection.WorkspaceId,
			},
		},
		InputRowType: reflect.TypeOf(models.CodexCodeReview{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, errors.Error) {
			cr := inputRow.(*models.CodexCodeReview)
			return []interface{}{buildCodexCodeReviewActivity(idGen, connectionId, cr)}, nil
		},
	})
	if err != nil {
		return err
	}
	return converter.Execute()
}
