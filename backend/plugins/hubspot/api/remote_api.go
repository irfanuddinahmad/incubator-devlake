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
	"fmt"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	apihelper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
)

type HubspotRemotePagination struct {
	Page int `json:"page"`
}

type hubspotPrivateAppTokenInfo struct {
	HubID uint64 `json:"hubId"`
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}

func listHubspotRemoteScopes(
	connection *models.HubspotConnection,
	apiClient plugin.ApiClient,
	_ string,
	_ HubspotRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope],
	nextPage *HubspotRemotePagination,
	err errors.Error,
) {
	scope, err := resolveHubspotRemoteScope(connection, apiClient)
	if err != nil {
		return nil, nil, err
	}
	if scope == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{}, nil, nil
	}
	return []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{makeHubspotRemoteScopeEntry(scope)}, nil, nil
}

func searchHubspotRemoteScopes(
	apiClient plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope],
	err errors.Error,
) {
	scope, err := resolveHubspotRemoteScope(nil, apiClient)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{}, nil
	}
	query := ""
	if params != nil {
		query = strings.TrimSpace(strings.ToLower(params.Search))
	}
	if query != "" {
		id := strings.ToLower(scope.Id)
		name := strings.ToLower(scope.Name)
		if !strings.Contains(id, query) && !strings.Contains(name, query) {
			return []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{}, nil
		}
	}
	return []dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{makeHubspotRemoteScopeEntry(scope)}, nil
}

func resolveHubspotRemoteScope(connection *models.HubspotConnection, apiClient plugin.ApiClient) (*models.HubspotScope, errors.Error) {
	portalID := ""
	if connection != nil {
		portalID = strings.TrimSpace(connection.PortalId)
	}
	if portalID == "" {
		var err errors.Error
		portalID, err = fetchHubspotPortalID(apiClient)
		if err != nil {
			return nil, err
		}
	}
	if portalID == "" {
		return nil, nil
	}
	return &models.HubspotScope{
		Id:   portalID,
		Name: fmt.Sprintf("Portal %s", portalID),
	}, nil
}

func fetchHubspotPortalID(apiClient plugin.ApiClient) (string, errors.Error) {
	if apiClient == nil {
		return "", errors.BadInput.New("api client is required")
	}
	res, err := apiClient.Post("oauth/v2/private-apps/get/access-token-info", nil, nil, nil)
	if err != nil {
		return "", err
	}
	var tokenInfo hubspotPrivateAppTokenInfo
	if err := apihelper.UnmarshalResponse(res, &tokenInfo); err != nil {
		return "", err
	}
	if tokenInfo.HubID == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d", tokenInfo.HubID), nil
}

func makeHubspotRemoteScopeEntry(scope *models.HubspotScope) dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope] {
	return dsmodels.DsRemoteApiScopeListEntry[models.HubspotScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       scope.Id,
		Name:     scope.Name,
		FullName: scope.ScopeFullName(),
		Data:     scope,
	}
}
