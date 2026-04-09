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
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
)

type hubspotObjectRecord struct {
	Id         string                 `json:"id"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
	Properties map[string]interface{} `json:"properties"`
}

var _ plugin.SubTaskEntryPoint = ExtractActivity

var ExtractActivityMeta = plugin.SubTaskMeta{
	Name:             "extractActivity",
	EntryPoint:       ExtractActivity,
	EnabledByDefault: true,
	Description:      "Extract HubSpot activity events from raw records into tool-layer rows",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	Dependencies:     []*plugin.SubTaskMeta{&CollectActivityMeta},
}

func ExtractActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*HubspotTaskData)
	if !ok {
		return errors.Default.New("task data is not HubspotTaskData")
	}

	targets := resolveHubspotCollectionTargets(data.Options.ObjectTypes)
	for _, target := range targets {
		if err := extractHubspotRawTable(taskCtx, data, target.RawTable, target.DomainObjectType); err != nil {
			return err
		}
	}

	return nil
}

func extractHubspotRawTable(
	taskCtx plugin.SubTaskContext,
	data *HubspotTaskData,
	rawTable string,
	sourceObjectType string,
) errors.Error {
	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx:   taskCtx,
			Table: rawTable,
			Options: hubspotRawParams{
				ConnectionId: data.Options.ConnectionId,
				ScopeId:      data.Options.ScopeId,
			},
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var record hubspotObjectRecord
			if err := errors.Convert(json.Unmarshal(row.Data, &record)); err != nil {
				return nil, err
			}

			if strings.TrimSpace(record.Id) == "" {
				return nil, nil
			}

			occurredAt, err := parseHubspotOccurredAt(record)
			if err != nil {
				return nil, err
			}
			if occurredAt.IsZero() {
				occurredAt = time.Now().UTC()
			}

			eventId := fmt.Sprintf("%s:%d", record.Id, occurredAt.UnixMilli())
			ownerId := extractHubspotOwnerId(record.Properties)
			actingEmail := extractHubspotOwnerEmail(record.Properties)

			event := &models.HubspotActivityEvent{
				ConnectionId:     data.Options.ConnectionId,
				ScopeId:          data.Options.ScopeId,
				EventId:          eventId,
				OccurredAt:       occurredAt,
				ActingUserId:     ownerId,
				ActingUserEmail:  actingEmail,
				ActionType:       extractHubspotActionType(record),
				ObjectType:       sourceObjectType,
				ObjectId:         record.Id,
				SourceObjectType: sourceObjectType,
				RawData:          string(row.Data),
			}
			return []interface{}{event}, nil
		},
	})
	if err != nil {
		return err
	}
	return extractor.Execute()
}

func parseHubspotOccurredAt(record hubspotObjectRecord) (time.Time, errors.Error) {
	if record.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, record.UpdatedAt); err == nil {
			return t.UTC(), nil
		}
	}
	if record.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, record.CreatedAt); err == nil {
			return t.UTC(), nil
		}
	}

	for _, key := range []string{"hs_lastmodifieddate", "hs_timestamp", "hs_createdate"} {
		if rawValue, ok := record.Properties[key]; ok {
			switch value := rawValue.(type) {
			case string:
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					continue
				}
				if ts, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
					return time.UnixMilli(ts).UTC(), nil
				}
				if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
					return t.UTC(), nil
				}
			}
		}
	}

	return time.Time{}, nil
}

func extractHubspotOwnerId(properties map[string]interface{}) string {
	if properties == nil {
		return ""
	}
	for _, key := range []string{"hubspot_owner_id", "owner_id", "hs_created_by_user_id", "hs_updated_by_user_id"} {
		raw, ok := properties[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case string:
			value := strings.TrimSpace(v)
			if value != "" {
				return value
			}
		case float64:
			return strconv.FormatInt(int64(v), 10)
		}
	}
	return ""
}

func extractHubspotOwnerEmail(properties map[string]interface{}) string {
	if properties == nil {
		return ""
	}
	for _, key := range []string{
		"hubspot_owner_email",
		"owner_email",
		"hs_updated_by_user_email",
		"hs_email_from_email",
		"hs_email_sender_email",
		"hs_created_by_user_email",
	} {
		raw, ok := properties[key]
		if !ok || raw == nil {
			continue
		}
		if email, ok := raw.(string); ok {
			email = strings.TrimSpace(email)
			if email != "" {
				return email
			}
		}
	}
	return ""
}

func extractHubspotActionType(record hubspotObjectRecord) string {
	updatedAt, hasUpdatedAt := parseHubspotRecordTime(record.UpdatedAt)
	createdAt, hasCreatedAt := parseHubspotRecordTime(record.CreatedAt)
	if hasCreatedAt && (!hasUpdatedAt || createdAt.Equal(updatedAt) || updatedAt.Before(createdAt)) {
		return "created"
	}

	createdMs, hasCreatedMs := parseHubspotPropertyMillis(record.Properties, "hs_createdate")
	modifiedMs, hasModifiedMs := parseHubspotPropertyMillis(record.Properties, "hs_lastmodifieddate")
	if hasCreatedMs && hasModifiedMs && createdMs == modifiedMs {
		return "created"
	}

	return "updated"
}

func parseHubspotRecordTime(raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func parseHubspotPropertyMillis(properties map[string]interface{}, key string) (int64, bool) {
	if properties == nil {
		return 0, false
	}
	raw, ok := properties[key]
	if !ok || raw == nil {
		return 0, false
	}
	str, ok := raw.(string)
	if !ok {
		return 0, false
	}
	value := strings.TrimSpace(str)
	if value == "" {
		return 0, false
	}
	ms, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return ms, true
}
