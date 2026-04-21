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
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

type NotionRemotePagination struct {
	Cursor   string `json:"cursor"`
	PageSize int    `json:"pageSize"`
}

type notionSearchResponse struct {
	Results    []notionDataSourceResult `json:"results"`
	HasMore    bool                     `json:"has_more"`
	NextCursor string                   `json:"next_cursor"`
}

type notionDataSourceResult struct {
	ID     string           `json:"id"`
	Object string           `json:"object"`
	Name   string           `json:"name"`
	Title  []notionRichText `json:"title"`
}

type notionRichText struct {
	PlainText string `json:"plain_text"`
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeSearch.Get(input)
}

func listNotionRemoteScopes(
	_ *models.NotionConnection,
	apiClient plugin.ApiClient,
	_ string,
	page NotionRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.NotionScope],
	nextPage *NotionRemotePagination,
	err errors.Error,
) {
	if page.PageSize <= 0 {
		page.PageSize = 50
	}
	children, response, err := notionSearchDataSources(apiClient, "", page.Cursor, page.PageSize)
	if err != nil {
		return nil, nil, err
	}
	if response.HasMore && strings.TrimSpace(response.NextCursor) != "" {
		nextPage = &NotionRemotePagination{Cursor: response.NextCursor, PageSize: page.PageSize}
	}
	return children, nextPage, nil
}

func searchNotionRemoteScopes(
	apiClient plugin.ApiClient,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.NotionScope],
	err errors.Error,
) {
	if params == nil {
		params = &dsmodels.DsRemoteApiScopeSearchParams{PageSize: 50}
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	cursor := ""
	remaining := pageSize
	for remaining > 0 {
		results, response, err := notionSearchDataSources(apiClient, params.Search, cursor, remaining)
		if err != nil {
			return nil, err
		}
		children = append(children, results...)
		remaining -= len(results)
		if !response.HasMore || strings.TrimSpace(response.NextCursor) == "" {
			break
		}
		cursor = response.NextCursor
	}
	return children, nil
}

func notionSearchDataSources(
	apiClient plugin.ApiClient,
	query string,
	startCursor string,
	pageSize int,
) ([]dsmodels.DsRemoteApiScopeListEntry[models.NotionScope], *notionSearchResponse, errors.Error) {
	if apiClient == nil {
		return nil, nil, errors.BadInput.New("api client is required")
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	requestBody := map[string]interface{}{
		"filter": map[string]interface{}{
			"property": "object",
			"value":    "data_source",
		},
		"page_size": pageSize,
	}
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery != "" {
		requestBody["query"] = trimmedQuery
	}
	trimmedCursor := strings.TrimSpace(startCursor)
	if trimmedCursor != "" {
		requestBody["start_cursor"] = trimmedCursor
	}
	res, err := apiClient.Post("v1/search", nil, requestBody, nil)
	if err != nil {
		return nil, nil, err
	}
	response := &notionSearchResponse{}
	if err := apihelper.UnmarshalResponse(res, response); err != nil {
		return nil, nil, err
	}
	children := make([]dsmodels.DsRemoteApiScopeListEntry[models.NotionScope], 0, len(response.Results))
	for _, result := range response.Results {
		entry, ok := makeNotionRemoteScopeEntry(result)
		if !ok {
			continue
		}
		children = append(children, entry)
	}
	return children, response, nil
}

func makeNotionRemoteScopeEntry(result notionDataSourceResult) (dsmodels.DsRemoteApiScopeListEntry[models.NotionScope], bool) {
	id := strings.TrimSpace(result.ID)
	if id == "" {
		return dsmodels.DsRemoteApiScopeListEntry[models.NotionScope]{}, false
	}
	name := strings.TrimSpace(result.Name)
	if name == "" {
		var titleParts []string
		for _, item := range result.Title {
			plainText := strings.TrimSpace(item.PlainText)
			if plainText != "" {
				titleParts = append(titleParts, plainText)
			}
		}
		name = strings.Join(titleParts, "")
	}
	if name == "" {
		name = fmt.Sprintf("Data source %s", id)
	}
	scope := &models.NotionScope{
		Id:   id,
		Name: name,
	}
	return dsmodels.DsRemoteApiScopeListEntry[models.NotionScope]{
		Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
		Id:       scope.Id,
		Name:     scope.Name,
		FullName: scope.ScopeFullName(),
		Data:     scope,
	}, true
}
