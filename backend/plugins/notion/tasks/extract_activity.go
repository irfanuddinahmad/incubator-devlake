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
	"fmt"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

type notionPageRecord struct {
	Id             string `json:"id"`
	Object         string `json:"object"`
	CreatedTime    string `json:"created_time"`
	LastEditedTime string `json:"last_edited_time"`
	CreatedBy      struct {
		Id     string `json:"id"`
		Object string `json:"object"`
		Type   string `json:"type"`
		Person struct {
			Email string `json:"email"`
		} `json:"person"`
	} `json:"created_by"`
	LastEditedBy struct {
		Id     string `json:"id"`
		Object string `json:"object"`
		Type   string `json:"type"`
		Person struct {
			Email string `json:"email"`
		} `json:"person"`
	} `json:"last_edited_by"`
}

var _ plugin.SubTaskEntryPoint = ExtractActivity

var ExtractActivityMeta = plugin.SubTaskMeta{
	Name:             "extractActivity",
	EntryPoint:       ExtractActivity,
	EnabledByDefault: true,
	Description:      "Extract Notion activity events from raw records into tool-layer rows",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	Dependencies:     []*plugin.SubTaskMeta{&CollectActivityMeta},
}

func buildNotionActivityEvent(rowData []byte, connectionId uint64, scopeId string) (*models.NotionActivityEvent, errors.Error) {
	var record notionPageRecord
	if err := errors.Convert(json.Unmarshal(rowData, &record)); err != nil {
		return nil, err
	}

	if strings.TrimSpace(record.Id) == "" {
		return nil, nil
	}

	occurredAt, err := time.Parse(time.RFC3339, record.LastEditedTime)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to parse notion last_edited_time")
	}

	// Determine action type: "created" when the page has never been separately edited.
	// In that case use created_by as the actor; otherwise use last_edited_by.
	actionType := "edited"
	actorId := strings.TrimSpace(record.LastEditedBy.Id)
	actorEmail := strings.TrimSpace(record.LastEditedBy.Person.Email)
	if record.CreatedTime != "" && record.CreatedTime == record.LastEditedTime {
		actionType = "created"
		actorId = strings.TrimSpace(record.CreatedBy.Id)
		actorEmail = strings.TrimSpace(record.CreatedBy.Person.Email)
	}

	event := &models.NotionActivityEvent{
		ConnectionId:     connectionId,
		ScopeId:          scopeId,
		EventId:          fmt.Sprintf("%s:%d", record.Id, occurredAt.UnixMilli()),
		OccurredAt:       occurredAt.UTC(),
		EditorUserId:     actorId,
		EditorUserEmail:  actorEmail,
		ActionType:       actionType,
		ObjectType:       strings.TrimSpace(record.Object),
		ObjectId:         strings.TrimSpace(record.Id),
		SourceObjectType: "notion_data_source_page",
		RawData:          string(rowData),
	}

	if event.ObjectType == "" {
		event.ObjectType = "page"
	}

	return event, nil
}

func ExtractActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*NotionTaskData)
	if !ok {
		return errors.Default.New("task data is not NotionTaskData")
	}

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawNotionActivityTable,
			Options: notionRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			event, err := buildNotionActivityEvent(row.Data, data.Options.ConnectionId, data.Options.ScopeId)
			if err != nil {
				return nil, err
			}
			if event == nil {
				return nil, nil
			}

			return []interface{}{event}, nil
		},
	})
	if err != nil {
		return err
	}

	return extractor.Execute()
}
