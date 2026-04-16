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

func TestListPlaneRemoteScopes_WithCursorPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/workspaces/workspace-alpha/projects/", r.URL.Path)
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"next_cursor":"cursor-2","next_page_results":true,"results":[{"id":"proj-1","name":"Alpha","identifier":"ALPHA","description":"A","network":2}]}`))
			return
		}
		assert.Equal(t, "cursor-2", r.URL.Query().Get("cursor"))
		_, _ = w.Write([]byte(`{"next_cursor":"","next_page_results":false,"results":[{"id":"proj-2","name":"Beta","identifier":"BETA","description":"B","network":0}]}`))
	}))
	defer server.Close()

	basicRes = contextimpl.NewDefaultBasicRes(viper.New(), logruslog.Global, nil)
	connection := &models.PlaneConnection{
		PlaneConn: models.PlaneConn{
			RestConnection: helper.RestConnection{Endpoint: server.URL},
			ApiKey:         "plane-api-key",
			WorkspaceSlug:  "workspace-alpha",
		},
	}

	apiClient, err := helper.NewApiClientFromConnection(context.TODO(), basicRes, connection)
	assert.NoError(t, err)

	children, nextPage, err := listPlaneRemoteScopes(connection, apiClient, "", PlaneRemotePagination{})
	assert.NoError(t, err)
	assert.Len(t, children, 1)
	assert.Equal(t, "proj-1", children[0].Id)
	assert.Equal(t, "workspace-alpha/ALPHA", children[0].FullName)
	assert.NotNil(t, nextPage)
	assert.Equal(t, "cursor-2", nextPage.Cursor)

	children, nextPage, err = listPlaneRemoteScopes(connection, apiClient, "", *nextPage)
	assert.NoError(t, err)
	assert.Len(t, children, 1)
	assert.Equal(t, "proj-2", children[0].Id)
	assert.Equal(t, "workspace-alpha/BETA", children[0].FullName)
	assert.Nil(t, nextPage)
}
