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
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// ExtractCommitAiShareMeta defines metadata for the commit AI-share extractor.
var ExtractCommitAiShareMeta = plugin.SubTaskMeta{
	Name:             "extractCommitAiShare",
	EntryPoint:       ExtractCommitAiShare,
	EnabledByDefault: false,
	Description:      "Extract Cursor commit AI-share records into _tool_cursor_commit_ai_share (Enterprise only)",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

// cursorCommitAiShareRow is the raw JSON row from the ai-code/commits endpoint.
type cursorCommitAiShareRow struct {
	RepoName           string `json:"repoName"`
	CommitSha          string `json:"commitSha"`
	CommitDate         string `json:"commitDate"`
	TabLinesAdded      int    `json:"tabLinesAdded"`
	ComposerLinesAdded int    `json:"composerLinesAdded"`
	ManualLinesAdded   int    `json:"manualLinesAdded"`
}

// ExtractCommitAiShare reads raw cursor_commit_ai_share records and writes
// strongly-typed CursorCommitAiShare rows to the tool layer.
func ExtractCommitAiShare(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CursorTaskData)
	if !ok {
		return errors.Default.New("task data is not CursorTaskData")
	}

	teamId := data.Options.TeamId

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawCommitAiShareTable,
			Options: cursorRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       teamId,
			},
			Params: cursorRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				TeamId:       teamId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var rawRow cursorCommitAiShareRow
			if err := json.Unmarshal(row.Data, &rawRow); err != nil {
				return nil, errors.Default.Wrap(err, "failed to unmarshal cursor commit-ai-share row")
			}

			commitDate, parseErr := time.Parse("2006-01-02", rawRow.CommitDate)
			if parseErr != nil {
				commitDate, parseErr = time.Parse(time.RFC3339, rawRow.CommitDate)
				if parseErr != nil {
					return nil, errors.Default.Wrap(parseErr, "failed to parse commitDate: "+rawRow.CommitDate)
				}
			}

			record := &models.CursorCommitAiShare{
				ConnectionId:       data.Options.ConnectionId,
				ScopeId:            data.Options.ScopeId,
				RepoName:           rawRow.RepoName,
				CommitSha:          rawRow.CommitSha,
				CommitDate:         commitDate,
				TabLinesAdded:      rawRow.TabLinesAdded,
				ComposerLinesAdded: rawRow.ComposerLinesAdded,
				ManualLinesAdded:   rawRow.ManualLinesAdded,
				NoPKModel: common.NoPKModel{
					RawDataOrigin: common.RawDataOrigin{
						RawDataTable:  rawCommitAiShareTable,
						RawDataParams: row.Params,
						RawDataId:     row.ID,
					},
				},
			}
			return []interface{}{record}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
