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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/apache/incubator-devlake/server/api/shared"
)

type PlaneTestConnResponse struct {
	shared.ApiBody
	Connection *models.PlaneConnection
}

func validateConnection(connection *models.PlaneConnection) errors.Error {
	if connection.Endpoint == "" {
		return errors.BadInput.New("plane endpoint is required")
	}
	if connection.WorkspaceSlug == "" {
		return errors.BadInput.New("plane workspaceSlug is required")
	}
	if connection.ApiKey == "" {
		return errors.BadInput.New("plane apiKey is required")
	}
	return nil
}

func testConnection(ctx context.Context, connection models.PlaneConnection) (*PlaneTestConnResponse, errors.Error) {
	if err := validateConnection(&connection); err != nil {
		return nil, err
	}

	apiClient, err := helper.NewApiClientFromConnection(ctx, basicRes, &connection)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("api/v1/workspaces/%s/projects/", url.PathEscape(connection.WorkspaceSlug))
	res, err := apiClient.Get(path, nil, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "error testing plane connection")
	}
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, errors.HttpStatus(http.StatusBadRequest).New("authentication error when testing connection - please check your API key")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.HttpStatus(res.StatusCode).New(fmt.Sprintf("unexpected status code: %d", res.StatusCode))
	}

	connection = connection.Sanitize()
	body := PlaneTestConnResponse{}
	body.Success = true
	body.Message = "success"
	body.Connection = &connection
	return &body, nil
}

// TestConnection tests the Plane connection.
func TestConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	input.Body = normalizeConnectionBody(input.Body)
	var connection models.PlaneConnection
	err := helper.DecodeMapStruct(input.Body, &connection, false)
	if err != nil {
		return nil, err
	}
	result, err := testConnection(apiRequestContext(input), connection)
	if err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	return &plugin.ApiResourceOutput{Body: result, Status: http.StatusOK}, nil
}

// TestExistingConnection tests an existing Plane connection.
func TestExistingConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	input.Body = normalizeConnectionBody(input.Body)
	connection, err := dsHelper.ConnApi.GetMergedConnection(input)
	if err != nil {
		return nil, errors.BadInput.Wrap(err, "find connection from db")
	}
	if err := helper.DecodeMapStruct(input.Body, connection, false); err != nil {
		return nil, err
	}
	result, err := testConnection(apiRequestContext(input), *connection)
	if err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	return &plugin.ApiResourceOutput{Body: result, Status: http.StatusOK}, nil
}

func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	input.Body = normalizeConnectionBody(input.Body)
	var connection models.PlaneConnection
	if err := validateConnectionInput(&connection, input); err != nil {
		return nil, err
	}
	if err := connHelper.Create(&connection, input); err != nil {
		return nil, err
	}
	sanitized := connection.Sanitize()
	return &plugin.ApiResourceOutput{Body: sanitized, Status: http.StatusCreated}, nil
}

func ListConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetAll(input)
}

func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	return dsHelper.ConnApi.GetDetail(input)
}

func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	input.Body = normalizeConnectionBody(input.Body)
	model, err := connApi.PatchModel(input, true)
	if err != nil {
		return nil, errors.Convert(err)
	}
	if err := validateConnection(model); err != nil {
		return nil, err
	}
	if err := dsHelper.ConnSrv.Update(model); err != nil {
		return nil, err
	}
	model = connApi.Sanitize(model)
	return &plugin.ApiResourceOutput{Body: model}, nil
}

func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	var connection models.PlaneConnection
	return connHelper.Delete(&connection, input)
}

func apiRequestContext(input *plugin.ApiResourceInput) context.Context {
	if input != nil && input.Request != nil {
		return input.Request.Context()
	}
	return context.TODO()
}

func validateConnectionInput(connection *models.PlaneConnection, input *plugin.ApiResourceInput) errors.Error {
	if err := helper.DecodeMapStruct(input.Body, connection, false); err != nil {
		return err
	}
	return validateConnection(connection)
}

func normalizeConnectionBody(body map[string]interface{}) map[string]interface{} {
	if body == nil {
		return nil
	}
	normalized := make(map[string]interface{}, len(body)+1)
	for key, value := range body {
		normalized[key] = value
	}
	if normalized["apiKey"] == nil {
		if token, ok := normalized["token"]; ok && token != nil && token != "" {
			normalized["apiKey"] = token
		}
	}
	return normalized
}
