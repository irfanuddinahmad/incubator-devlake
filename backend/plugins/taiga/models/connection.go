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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/utils"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// TaigaConn holds the essential information to connect to the Taiga API
type TaigaConn struct {
	helper.RestConnection `mapstructure:",squash"`
	helper.AccessToken    `mapstructure:",squash"`
}

func (tc *TaigaConn) Sanitize() TaigaConn {
	tc.AccessToken.Token = utils.SanitizeString(tc.AccessToken.Token)
	return *tc
}

// SetupAuthentication sets up the HTTP request with authentication
func (tc *TaigaConn) SetupAuthentication(req *http.Request) errors.Error {
	if tc.Token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.Token)
	}
	return nil
}

// TaigaConnection holds TaigaConn plus ID/Name for database storage
type TaigaConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	TaigaConn             `mapstructure:",squash"`
}

func (TaigaConnection) TableName() string {
	return "_tool_taiga_connections"
}

func (connection *TaigaConnection) MergeFromRequest(target *TaigaConnection, body map[string]interface{}) error {
	token := target.Token

	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}

	modifiedToken := target.Token

	// preserve existing token if not modified
	if modifiedToken == "" || modifiedToken == utils.SanitizeString(token) {
		target.Token = token
	}

	return nil
}

func (connection TaigaConnection) Sanitize() TaigaConnection {
	connection.TaigaConn = connection.TaigaConn.Sanitize()
	return connection
}
