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
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

var ExtractUsageMeta = plugin.SubTaskMeta{
	Name:             "extractUsage",
	EntryPoint:       ExtractUsage,
	EnabledByDefault: true,
	Description:      "Extract Codex usage records (threads/turns/credits) into tool-layer models",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func ExtractUsage(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*CodexTaskData)
	if !ok {
		return errors.Default.New("task data is not CodexTaskData")
	}

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawUsageTable,
			Options: codexRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				WorkspaceId:  data.Connection.WorkspaceId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var record codexUsageRecord
			if err := errors.Convert(json.Unmarshal(row.Data, &record)); err != nil {
				return nil, err
			}

			// Parse the date string (expected format: "2024-01-15").
			day, parseErr := time.Parse("2006-01-02", record.Date)
			if parseErr != nil {
				return nil, errors.Default.Wrap(parseErr, "failed to parse date: "+record.Date)
			}
			day = day.UTC()

			usage := &models.CodexUsage{
				ConnectionId:  data.Options.ConnectionId,
				ScopeId:       data.Options.ScopeId,
				Date:          day,
				ClientSurface: record.ClientSurface,
				UserEmail:     record.UserEmail,
				Threads:       record.Threads,
				Turns:         record.Turns,
				Credits:       record.Credits,
			}
			return []interface{}{usage}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}
