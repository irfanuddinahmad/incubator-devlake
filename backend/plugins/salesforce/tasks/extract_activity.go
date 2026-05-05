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
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
)

type salesforceObjectRecord struct {
	Id               string `json:"Id"`
	CreatedDate      string `json:"CreatedDate"`
	CreatedById      string `json:"CreatedById"`
	LastModifiedDate string `json:"LastModifiedDate"`
	LastModifiedById string `json:"LastModifiedById"`
	SystemModstamp   string `json:"SystemModstamp"`
	Attributes       struct {
		Type string `json:"type"`
	} `json:"attributes"`
}

var _ plugin.SubTaskEntryPoint = ExtractActivity

var ExtractActivityMeta = plugin.SubTaskMeta{
	Name:             "extractActivity",
	EntryPoint:       ExtractActivity,
	EnabledByDefault: true,
	Description:      "Extract Salesforce activity records from raw rows into tool-layer events",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	Dependencies:     []*plugin.SubTaskMeta{&CollectActivityPollingMeta},
}

func ExtractActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*SalesforceTaskData)
	if !ok {
		return errors.Default.New("task data is not SalesforceTaskData")
	}

	for _, objectType := range ResolveObjectTypes(data.Options.ObjectTypes) {
		if err := extractSalesforceRawTable(taskCtx, data, rawSalesforceObjectTableSuffix(objectType), objectType); err != nil {
			return err
		}
	}

	return nil
}

func extractSalesforceRawTable(
	taskCtx plugin.SubTaskContext,
	data *SalesforceTaskData,
	rawTableSuffix string,
	sourceObjectType string,
) errors.Error {
	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawTableSuffix,
			Options: salesforceRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
				ObjectType:   sourceObjectType,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			event, err := buildSalesforceActivityEvent(row.Data, data.Options.ConnectionId, data.Options.ScopeId, sourceObjectType)
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

func buildSalesforceActivityEvent(
	rowData []byte,
	connectionId uint64,
	scopeId string,
	sourceObjectType string,
) (*models.SalesforceActivityEvent, errors.Error) {
	var record salesforceObjectRecord
	if err := json.Unmarshal(rowData, &record); err != nil {
		return nil, errors.Convert(err)
	}
	if strings.TrimSpace(record.Id) == "" {
		return nil, nil
	}

	createdAt, err := parseSalesforceTime(record.CreatedDate)
	if err != nil {
		return nil, err
	}
	lastModifiedAt, err := parseSalesforceTime(record.LastModifiedDate)
	if err != nil {
		return nil, err
	}
	systemModstamp, err := parseSalesforceTime(record.SystemModstamp)
	if err != nil {
		return nil, err
	}

	occurredAt := lastModifiedAt
	if occurredAt.IsZero() {
		occurredAt = systemModstamp
	}
	if occurredAt.IsZero() {
		occurredAt = createdAt
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	actionType := "updated"
	actingUserId := strings.TrimSpace(record.LastModifiedById)
	if !createdAt.IsZero() && (lastModifiedAt.IsZero() || createdAt.Equal(lastModifiedAt)) {
		actionType = "created"
		actingUserId = strings.TrimSpace(record.CreatedById)
	}
	if actingUserId == "" {
		actingUserId = strings.TrimSpace(record.CreatedById)
	}

	sourceType := strings.TrimSpace(sourceObjectType)
	if sourceType == "" {
		sourceType = strings.TrimSpace(record.Attributes.Type)
	}
	if sourceType == "" {
		sourceType = "SalesforceObject"
	}

	return &models.SalesforceActivityEvent{
		ConnectionId: connectionId,
		ScopeId:      scopeId,
		EventId:      fmt.Sprintf("%s:%s:%d", sourceType, strings.TrimSpace(record.Id), occurredAt.UnixMilli()),
		OccurredAt:   occurredAt.UTC(),
		ActingUserId: actingUserId,
		// ActingUserEmail is left blank here; convert_activity enriches it from the
		// SalesforceUser table since SOQL object queries do not include actor email.
		ActingUserEmail:  "",
		ActionType:       actionType,
		ObjectType:       strings.ToLower(sourceType),
		ObjectId:         strings.TrimSpace(record.Id),
		SourceObjectType: sourceType,
		RawData:          string(rowData),
	}, nil
}
