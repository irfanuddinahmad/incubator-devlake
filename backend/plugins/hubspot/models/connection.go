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

package models

import (
	"net/http"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/utils"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"gorm.io/gorm"
)

const DefaultEndpoint = "https://api.hubapi.com"

type HubspotConn struct {
	helper.RestConnection `mapstructure:",squash"`
	ApiToken              string `mapstructure:"token" json:"token" gorm:"column:api_token;serializer:encdec"`
	PortalId              string `mapstructure:"portalId" json:"portalId" gorm:"column:portal_id;type:varchar(255)"`
	EnableWebhook         bool   `mapstructure:"enableWebhook" json:"enableWebhook" gorm:"column:enable_webhook"`
	WebhookSharedKey      string `mapstructure:"webhookSharedKey" json:"webhookSharedKey" gorm:"column:webhook_shared_key;type:varchar(255);serializer:encdec"`
	RateLimitPerHour      int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`
}

func (conn *HubspotConn) SetupAuthentication(request *http.Request) errors.Error {
	if conn == nil {
		return errors.BadInput.New("connection is required")
	}
	token := strings.TrimSpace(conn.ApiToken)
	if token == "" {
		return errors.BadInput.New("token is required")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (conn HubspotConn) Sanitize() HubspotConn {
	conn.ApiToken = utils.SanitizeString(conn.ApiToken)
	conn.WebhookSharedKey = utils.SanitizeString(conn.WebhookSharedKey)
	return conn
}

type HubspotConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	HubspotConn           `mapstructure:",squash"`
}

func (HubspotConnection) TableName() string {
	return "_tool_hubspot_connections"
}

func (connection HubspotConnection) Sanitize() HubspotConnection {
	connection.HubspotConn = connection.HubspotConn.Sanitize()
	return connection
}

func (connection *HubspotConnection) MergeFromRequest(target *HubspotConnection, body map[string]interface{}) error {
	if target == nil {
		return nil
	}
	originalToken := target.ApiToken
	originalWebhookSharedKey := target.WebhookSharedKey
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	sanitized := utils.SanitizeString(originalToken)
	if target.ApiToken == "" || target.ApiToken == sanitized {
		target.ApiToken = originalToken
	}
	sanitizedWebhookSharedKey := utils.SanitizeString(originalWebhookSharedKey)
	if target.WebhookSharedKey == "" || target.WebhookSharedKey == sanitizedWebhookSharedKey {
		target.WebhookSharedKey = originalWebhookSharedKey
	}
	if target.EnableWebhook && strings.TrimSpace(target.WebhookSharedKey) == "" {
		return errors.BadInput.New("webhookSharedKey is required when enableWebhook is true")
	}
	return nil
}

func (connection *HubspotConnection) Normalize() {
	if connection == nil {
		return
	}
	if strings.TrimSpace(connection.Endpoint) == "" {
		connection.Endpoint = DefaultEndpoint
	}
	if connection.RateLimitPerHour <= 0 {
		connection.RateLimitPerHour = 3600
	}
}

func (connection *HubspotConnection) BeforeSave(_ *gorm.DB) error {
	if connection == nil {
		return nil
	}
	if connection.EnableWebhook && strings.TrimSpace(connection.WebhookSharedKey) == "" {
		return errors.BadInput.New("webhookSharedKey is required when enableWebhook is true")
	}
	return nil
}
