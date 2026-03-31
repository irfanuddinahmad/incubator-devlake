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
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

// CodexRemotePagination is a placeholder for scope-list pagination.
type CodexRemotePagination struct {
	Page int `json:"page"`
}

func listCodexRemoteScopes(
	connection *models.CodexConnection,
	_ plugin.ApiClient,
	_ string,
	_ CodexRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.CodexScope],
	nextPage *CodexRemotePagination,
	err errors.Error,
) {
	if connection == nil {
		return nil, nil, errors.BadInput.New("connection is required")
	}
	projectId := strings.TrimSpace(connection.ProjectId)
	if projectId == "" {
		return []dsmodels.DsRemoteApiScopeListEntry[models.CodexScope]{}, nil, nil
	}
	children = append(children, dsmodels.DsRemoteApiScopeListEntry[models.CodexScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       projectId,
		Name:     projectId,
		FullName: projectId,
		Data: &models.CodexScope{
			Id:        projectId,
			ProjectId: projectId,
			Name:      projectId,
		},
	})
	return children, nil, nil
}

func searchCodexRemoteScopes(
	_ plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	[]dsmodels.DsRemoteApiScopeListEntry[models.CodexScope],
	errors.Error,
) {
	_ = params
	return []dsmodels.DsRemoteApiScopeListEntry[models.CodexScope]{}, nil
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}
