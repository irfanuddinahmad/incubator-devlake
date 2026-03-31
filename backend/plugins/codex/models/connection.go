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
	DefaultEndpoint         = "https://api.openai.com/v1"
	DefaultRateLimitPerHour = 1000
)

type CodexConn struct {
	helper.RestConnection `mapstructure:",squash"`
	ApiKey                string `mapstructure:"token" json:"token" gorm:"column:api_key;serializer:encdec"`
	ProjectId             string `mapstructure:"projectId" json:"projectId" gorm:"column:project_id;type:varchar(255)"`
	RateLimitPerHour      int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`
}

func (conn *CodexConn) SetupAuthentication(request *http.Request) errors.Error {
	if conn == nil {
		return errors.BadInput.New("connection is required")
	}
	key := strings.TrimSpace(conn.ApiKey)
	if key == "" {
		return errors.BadInput.New("apiKey is required")
	}
	request.Header.Set("Authorization", "Bearer "+key)
	return nil
}

func (conn CodexConn) Sanitize() CodexConn {
	conn.ApiKey = utils.SanitizeString(conn.ApiKey)
	return conn
}

type CodexConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	CodexConn             `mapstructure:",squash"`
}

func (CodexConnection) TableName() string {
	return "_tool_codex_connections"
}

func (connection CodexConnection) Sanitize() CodexConnection {
	connection.CodexConn = connection.CodexConn.Sanitize()
	return connection
}

func (connection *CodexConnection) MergeFromRequest(target *CodexConnection, body map[string]interface{}) error {
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

func (connection *CodexConnection) Normalize() {
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
