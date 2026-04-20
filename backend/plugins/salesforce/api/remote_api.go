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
	"net/url"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	apihelper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
)

type SalesforceRemotePagination struct {
	Page int `json:"page"`
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}

func listSalesforceRemoteScopes(
	connection *models.SalesforceConnection,
	apiClient plugin.ApiClient,
	_ string,
	_ SalesforceRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope],
	nextPage *SalesforceRemotePagination,
	err errors.Error,
) {
	scope, err := resolveSalesforceRemoteScope(connection, apiClient)
	if err != nil {
		return nil, nil, err
	}
	if scope == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{}, nil, nil
	}
	return []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{makeSalesforceRemoteScopeEntry(scope)}, nil, nil
}

func searchSalesforceRemoteScopes(
	apiClient plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope],
	err errors.Error,
) {
	scope, err := resolveSalesforceRemoteScope(nil, apiClient)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{}, nil
	}
	query := ""
	if params != nil {
		query = strings.TrimSpace(strings.ToLower(params.Search))
	}
	if query != "" {
		if !strings.Contains(strings.ToLower(scope.Id), query) && !strings.Contains(strings.ToLower(scope.Name), query) {
			return []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{}, nil
		}
	}
	return []dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{makeSalesforceRemoteScopeEntry(scope)}, nil
}

func resolveSalesforceRemoteScope(connection *models.SalesforceConnection, apiClient plugin.ApiClient) (*models.SalesforceScope, errors.Error) {
	apiVersion := models.DefaultApiVersion
	if connection != nil {
		apiVersion = connection.GetVersion()
	}
	return querySalesforceOrganization(apiClient, apiVersion)
}

func querySalesforceOrganization(apiClient plugin.ApiClient, apiVersion string) (*models.SalesforceScope, errors.Error) {
	if apiClient == nil {
		return nil, errors.BadInput.New("api client is required")
	}
	res, err := apiClient.Get(
		fmt.Sprintf("services/data/%s/query", apiVersion),
		url.Values{"q": []string{"SELECT Id, Name FROM Organization LIMIT 1"}},
		nil,
	)
	if err != nil {
		return nil, err
	}

	var response struct {
		Records []struct {
			Id   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"records"`
	}
	if err := apihelper.UnmarshalResponse(res, &response); err != nil {
		return nil, err
	}
	if len(response.Records) == 0 {
		return nil, nil
	}

	record := response.Records[0]
	return &models.SalesforceScope{
		Id:   strings.TrimSpace(record.Id),
		Name: strings.TrimSpace(record.Name),
	}, nil
}

func makeSalesforceRemoteScopeEntry(scope *models.SalesforceScope) dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope] {
	return dsmodels.DsRemoteApiScopeListEntry[models.SalesforceScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       scope.Id,
		Name:     scope.Name,
		FullName: scope.ScopeFullName(),
		Data:     scope,
	}
}
