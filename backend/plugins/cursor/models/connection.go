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
	// DefaultEndpoint is the Cursor API base URL.
	DefaultEndpoint = "https://api.cursor.com"
	// DefaultRateLimitPerHour is a conservative default.
	DefaultRateLimitPerHour = 1000
)

// CursorConn stores Cursor API connection credentials.
type CursorConn struct {
	helper.RestConnection `mapstructure:",squash"`
	// ApiKey is stored under the "token" key so the standard masked UI field works.
	ApiKey string `mapstructure:"token" json:"token" gorm:"column:api_key;serializer:encdec"`
	// TeamId is optional — when provided it is used to pre-populate the data scope.
	TeamId           string `mapstructure:"teamId" json:"teamId" gorm:"column:team_id"`
	RateLimitPerHour int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`
}

// SetupAuthentication implements plugin.ApiAuthenticator.
// Cursor uses HTTP Basic Auth: API key as username, empty password.
func (conn *CursorConn) SetupAuthentication(request *http.Request) errors.Error {
	if conn == nil {
		return errors.BadInput.New("connection is required")
	}
	key := strings.TrimSpace(conn.ApiKey)
	if key == "" {
		return errors.BadInput.New("apiKey is required")
	}
	request.SetBasicAuth(key, "")
	return nil
}

// Sanitize returns a copy with secrets redacted.
func (conn CursorConn) Sanitize() CursorConn {
	conn.ApiKey = utils.SanitizeString(conn.ApiKey)
	return conn
}

// CursorConnection persists Cursor connection details.
type CursorConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	CursorConn            `mapstructure:",squash"`
}

func (CursorConnection) TableName() string {
	return "_tool_cursor_connections"
}

// Sanitize returns a safe copy for API responses.
func (connection CursorConnection) Sanitize() CursorConnection {
	connection.CursorConn = connection.CursorConn.Sanitize()
	return connection
}

// MergeFromRequest merges user-supplied fields, preserving the stored API key
// when the caller sends back the sanitized placeholder.
func (connection *CursorConnection) MergeFromRequest(target *CursorConnection, body map[string]interface{}) error {
	if target == nil {
		return nil
	}
	originalKey := target.ApiKey
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	sanitized := utils.SanitizeString(originalKey)
	if target.ApiKey == "" || target.ApiKey == sanitized {
		target.ApiKey = originalKey
	}
	return nil
}

// Normalize applies defaults where necessary.
func (connection *CursorConnection) Normalize() {
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
