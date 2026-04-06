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
)

const (
	// DefaultEndpoint is the Anthropic API base URL.
	DefaultEndpoint = "https://api.anthropic.com/v1"
	// DefaultRateLimitPerHour is a conservative rate limit for the Admin API.
	DefaultRateLimitPerHour = 1000
	// AnthropicVersion is the required anthropic-version header value.
	AnthropicVersion = "2023-06-01"
)

// ClaudeConn stores Anthropic Claude connection settings.
type ClaudeConn struct {
	helper.RestConnection `mapstructure:",squash"`
	AdminApiKey           string `mapstructure:"token" json:"token" gorm:"column:admin_api_key;serializer:encdec"`
	OrganizationId        string `mapstructure:"organizationId" json:"organizationId" gorm:"type:varchar(255)"`
	RateLimitPerHour      int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`
}

// SetupAuthentication implements plugin.ApiAuthenticator so helper.NewApiClientFromConnection
// can attach the x-api-key header for Anthropic API requests.
func (conn *ClaudeConn) SetupAuthentication(request *http.Request) errors.Error {
	if conn == nil {
		return errors.BadInput.New("connection is required")
	}
	key := strings.TrimSpace(conn.AdminApiKey)
	if key == "" {
		return errors.BadInput.New("adminApiKey is required")
	}
	request.Header.Set("x-api-key", key)
	request.Header.Set("anthropic-version", AnthropicVersion)
	return nil
}

// Sanitize returns a copy with secrets redacted.
func (conn ClaudeConn) Sanitize() ClaudeConn {
	conn.AdminApiKey = utils.SanitizeString(conn.AdminApiKey)
	return conn
}

// ClaudeConnection persists Claude connection details with metadata required by DevLake.
type ClaudeConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	ClaudeConn            `mapstructure:",squash"`
}

func (ClaudeConnection) TableName() string {
	return "_tool_claude_connections"
}

// Sanitize returns a safe copy of the connection for API responses.
func (connection ClaudeConnection) Sanitize() ClaudeConnection {
	connection.ClaudeConn = connection.ClaudeConn.Sanitize()
	return connection
}

// MergeFromRequest merges user-supplied fields onto the existing connection,
// preserving the stored API key if the caller sends back the sanitized placeholder.
func (connection *ClaudeConnection) MergeFromRequest(target *ClaudeConnection, body map[string]interface{}) error {
	if target == nil {
		return nil
	}
	originalKey := target.AdminApiKey
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	sanitized := utils.SanitizeString(originalKey)
	if target.AdminApiKey == "" || target.AdminApiKey == sanitized {
		target.AdminApiKey = originalKey
	}
	return nil
}

// Normalize applies default connection values where necessary.
func (connection *ClaudeConnection) Normalize() {
	if connection == nil {
		return
	}
	if strings.TrimSpace(connection.Endpoint) == "" {
		connection.Endpoint = DefaultEndpoint
	}
	if connection.RateLimitPerHour <= 0 {
		connection.RateLimitPerHour = DefaultRateLimitPerHour
	}
}
