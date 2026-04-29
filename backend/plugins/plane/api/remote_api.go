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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	apihelper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	dsmodels "github.com/apache/incubator-devlake/helpers/pluginhelper/api/models"
	"github.com/apache/incubator-devlake/helpers/utils"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

type PlaneRemotePagination struct {
	Cursor string `json:"cursor" mapstructure:"cursor"`
}

type planeProjectListResponse struct {
	NextCursor      string               `json:"next_cursor"`
	NextPageResults bool                 `json:"next_page_results"`
	Results         []planeRemoteProject `json:"results"`
}

type planeRemoteProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
	Network     int    `json:"network"`
}

func (p planeRemoteProject) toScope(workspaceSlug string) *models.PlaneProject {
	return &models.PlaneProject{
		ProjectId:     p.ID,
		Name:          p.Name,
		Identifier:    p.Identifier,
		Description:   p.Description,
		Network:       p.Network,
		WorkspaceSlug: workspaceSlug,
	}
}

func (p planeRemoteProject) fullName(workspaceSlug string) string {
	if workspaceSlug == "" {
		if p.Identifier != "" {
			return p.Identifier
		}
		return p.Name
	}
	if p.Identifier != "" {
		return fmt.Sprintf("%s/%s", workspaceSlug, p.Identifier)
	}
	return fmt.Sprintf("%s/%s", workspaceSlug, p.Name)
}

func listPlaneRemoteScopes(
	connection *models.PlaneConnection,
	apiClient plugin.ApiClient,
	_ string,
	page PlaneRemotePagination,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject],
	nextPage *PlaneRemotePagination,
	err errors.Error,
) {
	if connection == nil {
		return nil, nil, errors.BadInput.New("connection is required")
	}
	if connection.WorkspaceSlug == "" {
		return nil, nil, errors.BadInput.New("WorkspaceSlug is required on the connection")
	}

	query := url.Values{
		"page_size": {"100"},
	}
	if page.Cursor != "" {
		query.Set("cursor", page.Cursor)
	}

	res, err := apiClient.Get(fmt.Sprintf("api/v1/workspaces/%s/projects/", url.PathEscape(connection.WorkspaceSlug)), query, nil)
	if err != nil {
		return nil, nil, err
	}

	var body planeProjectListResponse
	err = apihelper.UnmarshalResponse(res, &body)
	if err != nil {
		return nil, nil, err
	}

	for _, project := range body.Results {
		children = append(children, dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject]{
			Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
			Id:       project.ID,
			ParentId: nil,
			Name:     project.Name,
			FullName: project.fullName(connection.WorkspaceSlug),
			Data:     project.toScope(connection.WorkspaceSlug),
		})
	}

	if body.NextPageResults && body.NextCursor != "" {
		nextPage = &PlaneRemotePagination{Cursor: body.NextCursor}
	}
	return
}

// maxSearchPages caps the number of API pages fetched during a search to prevent
// unbounded API calls on large workspaces (100 results/page × 10 pages = 1000 max).
const maxSearchPages = 10

func searchPlaneRemoteScopes(
	apiClient plugin.ApiClient,
	workspaceSlug string,
	params *dsmodels.DsRemoteApiScopeSearchParams,
) (
	children []dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject],
	err errors.Error,
) {
	if params == nil {
		return []dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject]{}, nil
	}
	if workspaceSlug == "" {
		return nil, errors.BadInput.New("WorkspaceSlug is required")
	}

	query := url.Values{
		"page_size": {"100"},
	}
	cursor := ""
	matched := make([]dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject], 0)
	search := strings.TrimSpace(strings.ToLower(params.Search))
	pagesFetched := 0

	for {
		if cursor != "" {
			query.Set("cursor", cursor)
		} else {
			query.Del("cursor")
		}

		res, err := apiClient.Get(fmt.Sprintf("api/v1/workspaces/%s/projects/", url.PathEscape(workspaceSlug)), query, nil)
		if err != nil {
			return nil, err
		}

		var body planeProjectListResponse
		err = apihelper.UnmarshalResponse(res, &body)
		if err != nil {
			return nil, err
		}

		for _, project := range body.Results {
			fullName := project.fullName(workspaceSlug)
			if search == "" ||
				strings.Contains(strings.ToLower(project.Name), search) ||
				strings.Contains(strings.ToLower(project.Identifier), search) ||
				strings.Contains(strings.ToLower(fullName), search) {
				matched = append(matched, dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject]{
					Type:     apihelper.RAS_ENTRY_TYPE_SCOPE,
					Id:       project.ID,
					ParentId: nil,
					Name:     project.Name,
					FullName: fullName,
					Data:     project.toScope(workspaceSlug),
				})
			}
		}

		pagesFetched++
		if !body.NextPageResults || body.NextCursor == "" {
			break
		}
		if pagesFetched >= maxSearchPages {
			basicRes.GetLogger().Warn(nil, "searchPlaneRemoteScopes: reached maxSearchPages (%d), results may be incomplete for workspace %q", maxSearchPages, workspaceSlug)
			break
		}
		cursor = body.NextCursor
	}

	if params.Page <= 0 || params.PageSize <= 0 {
		return nil, errors.BadInput.New("page and pageSize must be positive integers")
	}
	start := (params.Page - 1) * params.PageSize
	if start >= len(matched) {
		return []dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject]{}, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], nil
}

func RemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return raScopeList.Get(input)
}

func SearchRemoteScopes(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection, err := connApi.FindByPk(input)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "find connection from db")
	}
	apiClient, err := apihelper.NewApiClientFromConnection(searchRequestContext(input), basicRes, connection)
	if err != nil {
		return nil, err
	}

	params := &dsmodels.DsRemoteApiScopeSearchParams{
		Page:     1,
		PageSize: 50,
	}
	if err := utils.DecodeMapStruct(input.Query, params, true); err != nil {
		return nil, err
	}
	if vld != nil {
		if err := vld.Struct(params); err != nil {
			return nil, errors.BadInput.Wrap(err, "invalid params")
		}
	}

	children, err := searchPlaneRemoteScopes(apiClient, connection.WorkspaceSlug, params)
	if err != nil {
		return nil, err
	}
	if children == nil {
		children = []dsmodels.DsRemoteApiScopeListEntry[models.PlaneProject]{}
	}
	for i := range children {
		children[i].ParentId = nil
	}
	return &plugin.ApiResourceOutput{
		Status: http.StatusOK,
		Body: map[string]interface{}{
			"children": children,
			"count":    len(children),
			"page":     params.Page,
			"pageSize": params.PageSize,
		},
	}, nil
}

func searchRequestContext(input *plugin.ApiResourceInput) context.Context {
	if input != nil && input.Request != nil {
		return input.Request.Context()
	}
	return context.TODO()
}
