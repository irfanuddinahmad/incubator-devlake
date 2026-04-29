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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"

	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	contextimpl "github.com/apache/incubator-devlake/impls/context"
	"github.com/apache/incubator-devlake/impls/logruslog"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/stretchr/testify/assert"
)

func TestTestConnection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/workspaces/workspace-alpha/projects/", r.URL.Path)
		assert.Equal(t, "plane-api-key", r.Header.Get("X-API-Key"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	basicRes = contextimpl.NewDefaultBasicRes(viper.New(), logruslog.Global, nil)
	connection := models.PlaneConnection{
		PlaneConn: models.PlaneConn{
			RestConnection: helper.RestConnection{Endpoint: server.URL},
			ApiKey:         "plane-api-key",
			WorkspaceSlug:  "workspace-alpha",
		},
	}

	result, err := testConnection(context.TODO(), connection)
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "success", result.Message)
	assert.NotEqual(t, "plane-api-key", result.Connection.ApiKey)
}

func TestTestConnection_MissingWorkspaceSlug(t *testing.T) {
	connection := models.PlaneConnection{
		PlaneConn: models.PlaneConn{
			RestConnection: helper.RestConnection{Endpoint: "https://api.plane.so"},
			ApiKey:         "plane-api-key",
		},
	}

	result, err := testConnection(context.TODO(), connection)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspaceSlug is required")
}

func TestTestConnection_BadApiKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	basicRes = contextimpl.NewDefaultBasicRes(viper.New(), logruslog.Global, nil)
	connection := models.PlaneConnection{
		PlaneConn: models.PlaneConn{
			RestConnection: helper.RestConnection{Endpoint: server.URL},
			ApiKey:         "bad-api-key",
			WorkspaceSlug:  "workspace-alpha",
		},
	}

	result, err := testConnection(context.TODO(), connection)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication error")
}
