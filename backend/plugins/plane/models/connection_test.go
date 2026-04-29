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
	"testing"

	"github.com/apache/incubator-devlake/core/utils"
	"github.com/stretchr/testify/assert"
)

func TestPlaneConn_SetupAuthentication(t *testing.T) {
	conn := PlaneConn{ApiKey: "plane-api-key"}
	req, err := http.NewRequest("GET", "https://api.plane.so/api/v1/workspaces/", nil)
	assert.NoError(t, err)

	authErr := conn.SetupAuthentication(req)
	assert.Nil(t, authErr)
	assert.Equal(t, "plane-api-key", req.Header.Get("X-API-Key"))
}

func TestPlaneConn_SetupAuthentication_RequiresApiKey(t *testing.T) {
	conn := PlaneConn{}
	req, err := http.NewRequest("GET", "https://api.plane.so/api/v1/workspaces/", nil)
	assert.NoError(t, err)

	authErr := conn.SetupAuthentication(req)
	assert.Error(t, authErr)
	assert.Contains(t, authErr.Error(), "Plane API key is required")
}

func TestPlaneConnection_MergeFromRequest_PreservesApiKey(t *testing.T) {
	original := &PlaneConnection{
		PlaneConn: PlaneConn{
			ApiKey:        "secret-api-key",
			WorkspaceSlug: "alpha",
		},
	}
	target := &PlaneConnection{}
	*target = *original

	body := map[string]interface{}{
		"workspaceSlug": "beta",
		"apiKey":        utils.SanitizeString(original.ApiKey),
	}

	err := original.MergeFromRequest(target, body)
	assert.NoError(t, err)
	assert.Equal(t, "secret-api-key", target.ApiKey)
	assert.Equal(t, "beta", target.WorkspaceSlug)
}

func TestPlaneConnection_MergeFromRequest_UpdatesApiKey(t *testing.T) {
	original := &PlaneConnection{
		PlaneConn: PlaneConn{
			ApiKey:        "old-api-key",
			WorkspaceSlug: "alpha",
		},
	}
	target := &PlaneConnection{}
	*target = *original

	body := map[string]interface{}{
		"apiKey":        "new-api-key",
		"workspaceSlug": "alpha",
	}

	err := original.MergeFromRequest(target, body)
	assert.NoError(t, err)
	assert.Equal(t, "new-api-key", target.ApiKey)
}

func TestPlaneConnection_MergeFromRequest_AliasedReceiver(t *testing.T) {
	model := &PlaneConnection{
		PlaneConn: PlaneConn{ApiKey: "secret-api-key", WorkspaceSlug: "alpha"},
	}
	body := map[string]interface{}{
		"apiKey":        utils.SanitizeString("secret-api-key"),
		"workspaceSlug": "alpha",
	}
	err := model.MergeFromRequest(model, body)
	assert.NoError(t, err)
	assert.Equal(t, "secret-api-key", model.ApiKey)
}
