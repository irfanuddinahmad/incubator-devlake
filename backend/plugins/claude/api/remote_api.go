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
	"github.com/apache/incubator-devlake/plugins/claude/models"
)

// ClaudeRemotePagination is a placeholder for scope list pagination.
type ClaudeRemotePagination struct {
	Page int `json:"page"`
}

func listClaudeRemoteScopes(
	connection *models.ClaudeConnection,
	_ plugin.ApiClient,
	_ string,
	_ ClaudeRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClaudeScope],
	nextPage *ClaudeRemotePagination,
	err errors.Error,
) {
	if connection == nil {
		return nil, nil, errors.BadInput.New("connection is required")
	}

	orgId := strings.TrimSpace(connection.OrganizationId)
	if orgId == "" {
		return children, nil, nil
	}

	children = append(children, dsmodels.DsRemoteApiScopeListEntry[models.ClaudeScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       orgId,
		Name:     orgId,
		FullName: orgId,
		Data: &models.ClaudeScope{
			Id:             orgId,
			OrganizationId: orgId,
			Name:           orgId,
		},
	})

	return children, nil, nil
}

func searchClaudeRemoteScopes(
	_ plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.ClaudeScope],
	err errors.Error,
) {
	if params == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.ClaudeScope]{}, nil
	}
	return children, nil
}

// RemoteScopes handles the GET remote-scopes endpoint.
func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

// SearchRemoteScopes handles the GET search-remote-scopes endpoint.
func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}
