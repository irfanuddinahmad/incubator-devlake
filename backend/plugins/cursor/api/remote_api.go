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
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	apihelper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// CursorRemotePagination is a placeholder for scope-list pagination.
// Cursor scopes are team-level and always return a single entry.
type CursorRemotePagination struct {
	Page int `json:"page"`
}

// RemoteScopes delegates to the raScopeList helper.
func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

// SearchRemoteScopes delegates to the raScopeSearch helper.
func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}

func listCursorRemoteScopes(
	connection *models.CursorConnection,
	_ plugin.ApiClient,
	_ string,
	_ CursorRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.CursorScope],
	nextPage *CursorRemotePagination,
	err errors.Error,
) {
	if connection == nil {
		return nil, nil, errors.BadInput.New("connection is required")
	}
	teamId := strings.TrimSpace(connection.TeamId)
	if teamId == "" {
		return []dsmodels.DsRemoteApiScopeListEntry[models.CursorScope]{}, nil, nil
	}
	children = append(children, dsmodels.DsRemoteApiScopeListEntry[models.CursorScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       teamId,
		Name:     teamId,
		FullName: teamId,
		Data: &models.CursorScope{
			Id:     teamId,
			TeamId: teamId,
			Name:   teamId,
		},
	})
	return children, nil, nil
}

func searchCursorRemoteScopes(
	_ plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.CursorScope],
	err errors.Error,
) {
	_ = params
	// Cursor scopes are implicitly defined by the API key; no search endpoint exists.
	return []dsmodels.DsRemoteApiScopeListEntry[models.CursorScope]{}, nil
}
