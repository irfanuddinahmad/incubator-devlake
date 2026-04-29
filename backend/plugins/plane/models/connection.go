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

type PlaneConn struct {
	helper.RestConnection `mapstructure:",squash"`
	ApiKey                string `mapstructure:"apiKey" json:"apiKey" gorm:"serializer:encdec" validate:"required"`
	WorkspaceSlug         string `mapstructure:"workspaceSlug" json:"workspaceSlug" validate:"required"`
}

func (pc PlaneConn) Sanitize() PlaneConn {
	pc.ApiKey = utils.SanitizeString(pc.ApiKey)
	return pc
}

func (pc *PlaneConn) SetupAuthentication(req *http.Request) errors.Error {
	if pc.ApiKey == "" {
		return errors.BadInput.New("Plane API key is required")
	}
	req.Header.Set("X-API-Key", pc.ApiKey)
	return nil
}

type PlaneConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	PlaneConn             `mapstructure:",squash"`
}

func (PlaneConnection) TableName() string {
	return "_tool_plane_connections"
}

func (connection *PlaneConnection) MergeFromRequest(target *PlaneConnection, body map[string]interface{}) error {
	originalApiKey := connection.ApiKey
	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}
	if target.ApiKey == "" || target.ApiKey == utils.SanitizeString(originalApiKey) {
		target.ApiKey = originalApiKey
	}
	return nil
}

func (connection PlaneConnection) Sanitize() PlaneConnection {
	connection.PlaneConn = connection.PlaneConn.Sanitize()
	return connection
}
