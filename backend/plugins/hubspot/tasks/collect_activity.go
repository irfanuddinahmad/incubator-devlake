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
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var defaultHubspotObjectTypes = []string{"leads", "deals", "contacts", "companies", "quotes"}

const hubspotSearchAfterCeiling = 10000

var hubspotObjectTypeToDomainType = map[string]string{
	"appointments": "appointment",
	"carts":        "cart",
	"companies":    "company",
	"contacts":     "contact",
	"deals":        "deal",
	"emails":       "email",
	"invoices":     "invoice",
	"leads":        "lead",
	"line_items":   "line_item",
	"notes":        "note",
	"orders":       "order",
	"products":     "product",
	"quotes":       "quote",
	"services":     "service",
	"users":        "user",
}

type hubspotCollectionTarget struct {
	ObjectType       string
	DomainObjectType string
	RawTable         string
}

type hubspotSearchResponse struct {
	Results []json.RawMessage `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

var _ plugin.SubTaskEntryPoint = CollectActivity

var CollectActivityMeta = plugin.SubTaskMeta{
	Name:             "collectActivity",
	EntryPoint:       CollectActivity,
	EnabledByDefault: true,
	Description:      "Collect HubSpot activity events from configured HubSpot APIs",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func CollectActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*HubspotTaskData)
	if !ok {
		return errors.Default.New("task data is not HubspotTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	targets := resolveHubspotCollectionTargets(data.Options.ObjectTypes)
	for _, target := range targets {
		if err := collectHubspotObjectType(taskCtx, data, apiClient, target.ObjectType, target.RawTable); err != nil {
			return err
		}
	}

	return nil
}

func buildHubspotSearchRequestBody(objectType string, since *time.Time, until *time.Time, pageSize int, customData interface{}) map[string]interface{} {
	modifiedProperty := hubspotModifiedDateProperty(objectType)
	filters := make([]map[string]interface{}, 0, 2)
	if since != nil {
		filters = append(filters, map[string]interface{}{
			"propertyName": modifiedProperty,
			"operator":     "GTE",
			"value":        strconv.FormatInt(since.UnixMilli(), 10),
		})
	}
	if until != nil {
		filters = append(filters, map[string]interface{}{
			"propertyName": modifiedProperty,
			"operator":     "LTE",
			"value":        strconv.FormatInt(until.UTC().UnixMilli(), 10),
		})
	}

	body := map[string]interface{}{
		"limit": pageSize,
		"sorts": []map[string]string{{
			"propertyName": modifiedProperty,
			"direction":    "ASCENDING",
		}},
		"properties": resolveHubspotSearchProperties(objectType),
	}
	if len(filters) > 0 {
		body["filterGroups"] = []map[string]interface{}{{"filters": filters}}
	}
	if customData != nil {
		if after, ok := customData.(string); ok && strings.TrimSpace(after) != "" {
			body["after"] = after
		}
	}

	return body
}

func resolveHubspotSince(collectedSince *time.Time, occurredAfter *time.Time) *time.Time {
	if collectedSince != nil && !collectedSince.IsZero() {
		t := collectedSince.UTC()
		return &t
	}
	if occurredAfter != nil {
		t := occurredAfter.UTC()
		return &t
	}
	return nil
}

func resolveHubspotUntil(occurredBefore *time.Time, now time.Time) *time.Time {
	if occurredBefore != nil {
		t := occurredBefore.UTC()
		return &t
	}
	// Use a fixed upper bound to avoid paging over a moving dataset,
	// which can invalidate HubSpot "after" cursors during long runs.
	t := now.UTC()
	return &t
}

func resolveHubspotCollectionTargets(requested []string) []hubspotCollectionTarget {
	selected := requested
	if len(selected) == 0 {
		selected = defaultHubspotObjectTypes
	}

	result := make([]hubspotCollectionTarget, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, raw := range selected {
		objectType := strings.TrimSpace(strings.ToLower(raw))
		if objectType == "" {
			continue
		}
		domainObjectType, ok := hubspotObjectTypeToDomainType[objectType]
		if !ok {
			continue
		}
		if _, exists := seen[objectType]; exists {
			continue
		}
		seen[objectType] = struct{}{}
		result = append(result, hubspotCollectionTarget{
			ObjectType:       objectType,
			DomainObjectType: domainObjectType,
			RawTable:         rawHubspotObjectTable(objectType),
		})
	}

	if len(result) == 0 {
		for _, objectType := range defaultHubspotObjectTypes {
			domainObjectType := hubspotObjectTypeToDomainType[objectType]
			result = append(result, hubspotCollectionTarget{
				ObjectType:       objectType,
				DomainObjectType: domainObjectType,
				RawTable:         rawHubspotObjectTable(objectType),
			})
		}
	}

	return result
}

func rawHubspotObjectTable(objectType string) string {
	return fmt.Sprintf("_raw_hubspot_%s", strings.ReplaceAll(objectType, "-", "_"))
}

func hubspotModifiedDateProperty(objectType string) string {
	switch strings.TrimSpace(strings.ToLower(objectType)) {
	case "contacts":
		return "lastmodifieddate"
	default:
		return "hs_lastmodifieddate"
	}
}

func resolveHubspotSearchProperties(objectType string) []string {
	properties := []string{
		"hs_timestamp",
		"hs_lastmodifieddate",
		"hs_createdate",
		"lastmodifieddate",
		"createdate",
		"hubspot_owner_id",
		"hubspot_owner_email",
		"owner_email",
		"hs_created_by_user_id",
		"hs_updated_by_user_id",
		"hs_created_by_user_email",
	}

	if objectType == "emails" {
		properties = append(properties, "hs_email_from_email", "hs_email_sender_email")
	}

	return properties
}

func parseHubspotSearchResponse(body []byte) ([]json.RawMessage, errors.Error) {
	var envelope hubspotSearchResponse
	if err := errors.Convert(json.Unmarshal(body, &envelope)); err != nil {
		return nil, errors.Default.Wrap(err, "failed to parse HubSpot search response")
	}
	return envelope.Results, nil
}

func parseHubspotNextAfter(body []byte) (string, errors.Error) {
	var envelope hubspotSearchResponse
	if err := errors.Convert(json.Unmarshal(body, &envelope)); err != nil {
		return "", errors.Default.Wrap(err, "failed to parse HubSpot pagination response")
	}
	return strings.TrimSpace(envelope.Paging.Next.After), nil
}

func resolveHubspotNextCustomData(body []byte) (interface{}, errors.Error) {
	after, parseErr := parseHubspotNextAfter(body)
	if parseErr != nil {
		return nil, parseErr
	}
	if after == "" {
		return nil, helper.ErrFinishCollect
	}
	if offset, convErr := strconv.Atoi(after); convErr == nil && offset >= hubspotSearchAfterCeiling {
		// HubSpot search API fails with generic 400 once the next "after" cursor reaches
		// the hard pagination ceiling. Stop current collection window gracefully.
		return nil, helper.ErrFinishCollect
	}
	return after, nil
}

func collectHubspotObjectType(
	taskCtx plugin.SubTaskContext,
	data *HubspotTaskData,
	apiClient helper.RateLimitedApiClient,
	objectType string,
	rawTable string,
) errors.Error {
	rawArgs := helper.RawDataSubTaskArgs{
		Ctx:   taskCtx,
		Table: rawTable,
		Options: hubspotRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
		},
		Params: hubspotRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
		},
	}

	collector, err := helper.NewStatefulApiCollector(rawArgs)
	if err != nil {
		return err
	}

	since := resolveHubspotSince(collector.GetSince(), data.Options.OccurredAfter)
	until := resolveHubspotUntil(data.Options.OccurredBefore, time.Now())

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   apiClient,
		PageSize:    100,
		UrlTemplate: fmt.Sprintf("crm/v3/objects/%s/search", objectType),
		Method:      http.MethodPost,
		RequestBody: func(reqData *helper.RequestData) map[string]interface{} {
			return buildHubspotSearchRequestBody(objectType, since, until, reqData.Pager.Size, reqData.CustomData)
		},
		GetNextPageCustomData: func(prevReqData *helper.RequestData, prevPageResponse *http.Response) (interface{}, errors.Error) {
			body, readErr := io.ReadAll(prevPageResponse.Body)
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read HubSpot pagination response")
			}
			return resolveHubspotNextCustomData(body)
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				return nil, errors.Default.Wrap(readErr, "failed to read HubSpot response body")
			}
			return parseHubspotSearchResponse(body)
		},
		Incremental: true,
		Concurrency: 1,
	})
	if err != nil {
		return err
	}

	return collector.Execute()
}
