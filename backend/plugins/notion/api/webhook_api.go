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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/notion/models"
	"github.com/apache/incubator-devlake/server/api/shared"
)

type notionWebhookAuthor struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type notionWebhookEntity struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type notionWebhookEvent struct {
	Id                string                `json:"id"`
	Timestamp         string                `json:"timestamp"`
	Type              string                `json:"type"`
	Authors           []notionWebhookAuthor `json:"authors"`
	Entity            notionWebhookEntity   `json:"entity"`
	VerificationToken string                `json:"verification_token"`
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
	connection := &models.NotionConnection{}
	if err := db.First(connection, dal.Where("id = ?", connectionId)); err != nil {
		if db.IsErrorNotFound(err) {
			return nil, errors.NotFound.New("Notion connection not found")
		}
		return nil, errors.Default.Wrap(err, "failed to find Notion connection")
	}
	if !connection.EnableWebhook {
		return nil, errors.BadInput.New("webhook is disabled for this Notion connection")
	}
	if strings.TrimSpace(connection.WebhookSharedKey) == "" {
		return nil, errors.BadInput.New("webhookSharedKey is required for Notion webhook verification")
	}

	event, err := decodeNotionWebhookEvent(body)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid webhook payload")
	}

	if strings.TrimSpace(event.VerificationToken) != "" {
		return &plugin.ApiResourceOutput{
			Status: http.StatusOK,
			Body: shared.ApiBody{
				Success: true,
				Message: "verification token received",
			},
		}, nil
	}
	if verifyErr := verifyNotionWebhookSignature(input.Request, body, connection.WebhookSharedKey); verifyErr != nil {
		return nil, verifyErr
	}

	activity := mapNotionWebhookEvent(connectionId, scopeId, event)
	if err := basicRes.GetDal().CreateOrUpdate(activity); err != nil {
		return nil, errors.Default.Wrap(err, "failed to store Notion webhook event")
	}

	relevant := 0
	if activity.ActionType != "ignored" {
		relevant = 1
	}

	return &plugin.ApiResourceOutput{
		Status: http.StatusOK,
		Body: shared.ApiBody{
			Success: true,
			Message: fmt.Sprintf("stored 1 webhook event (%d relevant to user activities)", relevant),
		},
	}, nil
}

func decodeNotionWebhookEvent(body []byte) (*notionWebhookEvent, errors.Error) {
	event := &notionWebhookEvent{}
	if err := errors.Convert(json.Unmarshal(body, event)); err != nil {
		return nil, err
	}
	return event, nil
}

func mapNotionWebhookEvent(connectionId uint64, scopeId string, event *notionWebhookEvent) *models.NotionActivityEvent {
	occurredAt := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(event.Timestamp)); err == nil {
		occurredAt = t.UTC()
	}

	actionType, objectType := mapNotionType(strings.TrimSpace(event.Type))
	eventId := strings.TrimSpace(event.Id)
	if eventId == "" {
		eventId = fmt.Sprintf("webhook:%s:%d", scopeId, occurredAt.UnixMilli())
	}
	objectId := strings.TrimSpace(event.Entity.Id)
	if objectId == "" {
		objectId = eventId
	}

	editorUserId := ""
	for _, author := range event.Authors {
		if strings.TrimSpace(author.Id) == "" {
			continue
		}
		editorUserId = strings.TrimSpace(author.Id)
		if strings.EqualFold(strings.TrimSpace(author.Type), "person") {
			break
		}
	}

	rawData, _ := json.Marshal(event)
	return &models.NotionActivityEvent{
		ConnectionId:     connectionId,
		ScopeId:          scopeId,
		EventId:          eventId,
		OccurredAt:       occurredAt,
		EditorUserId:     editorUserId,
		EditorUserEmail:  "",
		ActionType:       actionType,
		ObjectType:       objectType,
		ObjectId:         objectId,
		SourceObjectType: strings.TrimSpace(event.Type),
		RawData:          string(rawData),
	}
}

func mapNotionType(eventType string) (string, string) {
	eventType = strings.TrimSpace(strings.ToLower(eventType))
	if eventType == "" {
		return "ignored", ""
	}

	parts := strings.Split(eventType, ".")
	if len(parts) != 2 {
		return "ignored", ""
	}

	objectType := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])

	switch action {
	case "created":
		return "created", objectType
	case "deleted":
		return "deleted", objectType
	case "undeleted":
		return "restored", objectType
	case "moved":
		return "moved", objectType
	case "locked":
		return "locked", objectType
	case "unlocked":
		return "unlocked", objectType
	case "updated", "content_updated", "properties_updated", "schema_updated":
		return "updated", objectType
	default:
		return "ignored", objectType
	}
}

func verifyNotionWebhookSignature(request *http.Request, body []byte, secret string) errors.Error {
	if request == nil {
		return errors.BadInput.New("request context is required for Notion signature verification")
	}
	signature := strings.TrimSpace(request.Header.Get("X-Notion-Signature"))
	if signature == "" {
		return errors.Unauthorized.New("missing X-Notion-Signature header")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.BadInput.New("Notion webhook secret is empty")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := "sha256=" + fmt.Sprintf("%x", mac.Sum(nil))
	if !timingSafeEqual(signature, expected) {
		return errors.Unauthorized.New("invalid Notion webhook signature")
	}
	return nil
}

func timingSafeEqual(left string, right string) bool {
	leftBytes := []byte(left)
	rightBytes := []byte(right)
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}
