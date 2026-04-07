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

package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
	"github.com/apache/incubator-devlake/server/api/shared"
)

type hubspotWebhookEvent struct {
	EventId          int64  `json:"eventId"`
	OccurredAt       int64  `json:"occurredAt"`
	SubscriptionType string `json:"subscriptionType"`
	ObjectId         int64  `json:"objectId"`
	ChangeFlag       string `json:"changeFlag"`
	SourceId         string `json:"sourceId"`
}

func PostWebhookEvents(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connectionId, err := strconv.ParseUint(strings.TrimSpace(input.Params["connectionId"]), 10, 64)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid connectionId")
	}
	scopeId := strings.TrimSpace(input.Params["scopeId"])
	if scopeId == "" {
		return nil, errors.BadInput.New("scopeId is required")
	}

	body := make([]byte, 0)
	if input.Request != nil && input.Request.Body != nil {
		body, err = io.ReadAll(input.Request.Body)
		if err != nil {
			return nil, errors.BadInput.Wrap(err, "failed to read request body")
		}
	}
	if len(body) == 0 {
		body, err = json.Marshal(input.Body)
		if err != nil {
			return nil, errors.BadInput.Wrap(err, "failed to marshal request body")
		}
	}
	if len(body) == 0 {
		return nil, errors.BadInput.New("request body is required")
	}
	db := basicRes.GetDal()
	connection := &models.HubspotConnection{}
	if err := db.First(connection, dal.Where("id = ?", connectionId)); err != nil {
		if db.IsErrorNotFound(err) {
			return nil, errors.NotFound.New("HubSpot connection not found")
		}
		return nil, errors.Default.Wrap(err, "failed to find HubSpot connection")
	}
	if !connection.EnableWebhook {
		return nil, errors.BadInput.New("webhook is disabled for this HubSpot connection")
	}
	if strings.TrimSpace(connection.WebhookSharedKey) == "" {
		return nil, errors.BadInput.New("webhookSharedKey is required for HubSpot webhook verification")
	}
	if verifyErr := verifyHubspotWebhookSignature(input.Request, body, connection.WebhookSharedKey); verifyErr != nil {
		return nil, verifyErr
	}

	events, err := decodeHubspotWebhookEvents(body)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid webhook payload")
	}

	inserted := 0
	relevant := 0
	for i, event := range events {
		activity := mapHubspotWebhookEvent(connectionId, scopeId, event, i)
		if err := db.CreateOrUpdate(activity); err != nil {
			return nil, errors.Default.Wrap(err, "failed to store HubSpot webhook event")
		}
		inserted++
		if activity.ActionType != "ignored" {
			relevant++
		}
	}

	return &plugin.ApiResourceOutput{
		Status: http.StatusOK,
		Body: shared.ApiBody{
			Success: true,
			Message: fmt.Sprintf("stored %d webhook events (%d relevant to user activities)", inserted, relevant),
		},
	}, nil
}

func verifyHubspotWebhookSignature(request *http.Request, body []byte, secret string) errors.Error {
	if request == nil {
		return errors.BadInput.New("request context is required for HubSpot signature verification")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.BadInput.New("HubSpot webhook secret is empty")
	}

	if signatureV3 := strings.TrimSpace(request.Header.Get("X-HubSpot-Signature-v3")); signatureV3 != "" {
		timestamp := strings.TrimSpace(request.Header.Get("X-HubSpot-Request-Timestamp"))
		if timestamp == "" {
			return errors.Unauthorized.New("missing X-HubSpot-Request-Timestamp header")
		}
		timestampMs, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return errors.Unauthorized.Wrap(err, "invalid X-HubSpot-Request-Timestamp header")
		}
		if absInt64(time.Now().UnixMilli()-timestampMs) > 5*60*1000 {
			return errors.Unauthorized.New("expired HubSpot webhook timestamp")
		}
		raw := request.Method + buildRequestURI(request) + string(body) + timestamp
		expected := computeHmacBase64(raw, secret)
		if !timingSafeEqual(signatureV3, expected) {
			return errors.Unauthorized.New("invalid HubSpot v3 webhook signature")
		}
		return nil
	}

	signature := strings.TrimSpace(request.Header.Get("X-HubSpot-Signature"))
	if signature == "" {
		return errors.Unauthorized.New("missing HubSpot signature header")
	}
	version := strings.TrimSpace(strings.ToLower(request.Header.Get("X-HubSpot-Signature-Version")))
	if version == "" {
		version = "v1"
	}

	var source string
	switch version {
	case "v2":
		source = secret + request.Method + buildRequestURI(request) + string(body)
	default:
		source = secret + string(body)
	}
	h := sha256.New()
	_, _ = h.Write([]byte(source))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	if !timingSafeEqual(signature, expected) {
		return errors.Unauthorized.New("invalid HubSpot webhook signature")
	}

	return nil
}

func buildRequestURI(request *http.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}
	scheme := strings.TrimSpace(request.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(request.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = request.Host
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, request.URL.RequestURI())
}

func computeHmacBase64(raw string, secret string) string {
	var mac hash.Hash = hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(raw))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func timingSafeEqual(left string, right string) bool {
	leftBytes := []byte(left)
	rightBytes := []byte(right)
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func decodeHubspotWebhookEvents(body []byte) ([]hubspotWebhookEvent, errors.Error) {
	events := make([]hubspotWebhookEvent, 0)
	if err := errors.Convert(json.Unmarshal(body, &events)); err == nil {
		return events, nil
	}
	var single hubspotWebhookEvent
	if err := errors.Convert(json.Unmarshal(body, &single)); err != nil {
		return nil, err
	}
	return []hubspotWebhookEvent{single}, nil
}

func mapHubspotWebhookEvent(connectionId uint64, scopeId string, event hubspotWebhookEvent, idx int) *models.HubspotActivityEvent {
	occurredAt := time.Now().UTC()
	if event.OccurredAt > 0 {
		occurredAt = time.UnixMilli(event.OccurredAt).UTC()
	}
	actionType := mapHubspotActionType(event.SubscriptionType, event.ChangeFlag)
	objectType := mapHubspotObjectType(event.SubscriptionType)
	objectId := ""
	if event.ObjectId > 0 {
		objectId = strconv.FormatInt(event.ObjectId, 10)
	}

	eventId := ""
	if event.EventId > 0 {
		eventId = strconv.FormatInt(event.EventId, 10)
	}
	if eventId == "" {
		eventId = fmt.Sprintf("webhook:%s:%d:%d", scopeId, occurredAt.UnixMilli(), idx)
	}

	rawData, _ := json.Marshal(event)

	return &models.HubspotActivityEvent{
		ConnectionId:     connectionId,
		ScopeId:          scopeId,
		EventId:          eventId,
		OccurredAt:       occurredAt,
		ActingUserId:     strings.TrimSpace(event.SourceId),
		ActingUserEmail:  "",
		ActionType:       actionType,
		ObjectType:       objectType,
		ObjectId:         objectId,
		SourceObjectType: strings.TrimSpace(event.SubscriptionType),
		RawData:          string(rawData),
	}
}

func mapHubspotObjectType(subscriptionType string) string {
	subscriptionType = strings.TrimSpace(strings.ToLower(subscriptionType))
	if subscriptionType == "" {
		return ""
	}
	parts := strings.Split(subscriptionType, ".")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func mapHubspotActionType(subscriptionType, changeFlag string) string {
	subscriptionType = strings.TrimSpace(strings.ToLower(subscriptionType))
	changeFlag = strings.TrimSpace(strings.ToLower(changeFlag))

	switch {
	case strings.Contains(subscriptionType, "creation") || changeFlag == "created" || changeFlag == "new":
		return "created"
	case strings.Contains(subscriptionType, "deletion") || changeFlag == "deleted":
		return "deleted"
	case strings.Contains(subscriptionType, "propertychange") || strings.Contains(subscriptionType, "change"):
		return "updated"
	default:
		return "ignored"
	}
}
