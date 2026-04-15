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
	"github.com/sirupsen/logrus"
)

var defaultHubspotObjectTypes = []string{"leads", "deals", "contacts", "companies", "quotes"}

const hubspotSearchAfterCeiling = 10000
const hubspotMinWindowWidth = time.Millisecond

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
	Total   int               `json:"total"`
	Results []json.RawMessage `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

type hubspotSearchWindow struct {
	Since *time.Time
	Until *time.Time
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
			"operator":     "LT",
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
		t := occurredBefore.UTC().Add(time.Millisecond)
		return &t
	}
	// Use a fixed exclusive upper bound to avoid paging over a moving dataset,
	// which can invalidate HubSpot "after" cursors during long runs.
	t := now.UTC().Add(time.Millisecond)
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
		logrus.WithField("after", after).Warn("[hubspot] reached search pagination ceiling; ending current collection window")
		return nil, helper.ErrFinishCollect
	}
	return after, nil
}

func probeHubspotSearchWindow(apiClient *helper.ApiAsyncClient, objectType string, window hubspotSearchWindow) (int, *time.Time, errors.Error) {
	res, err := apiClient.Post(
		fmt.Sprintf("crm/v3/objects/%s/search", objectType),
		nil,
		buildHubspotSearchRequestBody(objectType, window.Since, window.Until, 1, nil),
		nil,
	)
	if err != nil {
		return 0, nil, err
	}

	var envelope hubspotSearchResponse
	if err := helper.UnmarshalResponse(res, &envelope); err != nil {
		return 0, nil, err
	}

	if len(envelope.Results) == 0 {
		return envelope.Total, nil, nil
	}

	modifiedAt, err := parseHubspotWindowModifiedAt(envelope.Results[0], objectType)
	if err != nil {
		return 0, nil, err
	}
	if modifiedAt == nil {
		return envelope.Total, nil, nil
	}
	return envelope.Total, modifiedAt, nil
}

func parseHubspotWindowModifiedAt(item json.RawMessage, objectType string) (*time.Time, errors.Error) {
	var record hubspotObjectRecord
	if err := errors.Convert(json.Unmarshal(item, &record)); err != nil {
		return nil, err
	}
	modifiedAt, err := parseHubspotModifiedAt(record, objectType)
	if err != nil {
		return nil, err
	}
	if modifiedAt.IsZero() {
		return nil, nil
	}
	return &modifiedAt, nil
}

func parseHubspotModifiedAt(record hubspotObjectRecord, objectType string) (time.Time, errors.Error) {
	if record.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, record.UpdatedAt); err == nil {
			return t.UTC(), nil
		}
	}

	modifiedProperty := hubspotModifiedDateProperty(objectType)
	if rawValue, ok := record.Properties[modifiedProperty]; ok {
		switch value := rawValue.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				break
			}
			if ts, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
				return time.UnixMilli(ts).UTC(), nil
			}
			if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
				return t.UTC(), nil
			}
		}
	}

	return parseHubspotOccurredAt(record)
}

func splitHubspotSearchWindow(window hubspotSearchWindow) (hubspotSearchWindow, hubspotSearchWindow, bool) {
	if window.Since == nil || window.Until == nil {
		return hubspotSearchWindow{}, hubspotSearchWindow{}, false
	}
	if !window.Until.After(*window.Since) {
		return hubspotSearchWindow{}, hubspotSearchWindow{}, false
	}
	span := window.Until.Sub(*window.Since)
	if span <= hubspotMinWindowWidth {
		return hubspotSearchWindow{}, hubspotSearchWindow{}, false
	}
	mid := window.Since.Add(span / 2)
	if !mid.After(*window.Since) || !window.Until.After(mid) {
		return hubspotSearchWindow{}, hubspotSearchWindow{}, false
	}
	leftUntil := mid
	rightSince := mid
	return hubspotSearchWindow{Since: window.Since, Until: &leftUntil}, hubspotSearchWindow{Since: &rightSince, Until: window.Until}, true
}

func planHubspotCollectionWindows(apiClient *helper.ApiAsyncClient, objectType string, window hubspotSearchWindow) ([]hubspotSearchWindow, errors.Error) {
	total, earliest, err := probeHubspotSearchWindow(apiClient, objectType, window)
	if err != nil {
		return nil, errors.Default.Wrap(err, fmt.Sprintf("failed to probe HubSpot %s window", objectType))
	}
	if total == 0 {
		return nil, nil
	}
	if total <= hubspotSearchAfterCeiling {
		return []hubspotSearchWindow{window}, nil
	}

	if window.Since == nil && earliest != nil {
		window.Since = earliest
	}

	left, right, ok := splitHubspotSearchWindow(window)
	if !ok {
		logrus.WithFields(logrus.Fields{
			"objectType": objectType,
			"total":      total,
			"since":      window.Since,
			"until":      window.Until,
		}).Warn("[hubspot] unable to split overloaded search window further; fallback ceiling guard will apply")
		return []hubspotSearchWindow{window}, nil
	}

	leftWindows, err := planHubspotCollectionWindows(apiClient, objectType, left)
	if err != nil {
		return nil, err
	}
	rightWindows, err := planHubspotCollectionWindows(apiClient, objectType, right)
	if err != nil {
		return nil, err
	}
	return append(leftWindows, rightWindows...), nil
}

func collectHubspotObjectType(
	taskCtx plugin.SubTaskContext,
	data *HubspotTaskData,
	apiClient *helper.ApiAsyncClient,
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
	windows, err := planHubspotCollectionWindows(apiClient, objectType, hubspotSearchWindow{Since: since, Until: until})
	if err != nil {
		return err
	}
	if len(windows) == 0 {
		return nil
	}

	for _, window := range windows {
		window := window
		err = collector.InitCollector(helper.ApiCollectorArgs{
			ApiClient:   apiClient,
			PageSize:    100,
			UrlTemplate: fmt.Sprintf("crm/v3/objects/%s/search", objectType),
			Method:      http.MethodPost,
			RequestBody: func(reqData *helper.RequestData) map[string]interface{} {
				return buildHubspotSearchRequestBody(objectType, window.Since, window.Until, reqData.Pager.Size, reqData.CustomData)
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
	}

	return collector.Execute()
}
