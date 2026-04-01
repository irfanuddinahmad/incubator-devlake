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

package service

import (
	stdctx "context"
	"net/http"

	corectx "github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/cursor/models"
)

// TestConnectionResult is the payload returned by the connection-test endpoints.
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestConnection calls the Cursor API to verify credentials.
func TestConnection(ctx stdctx.Context, br corectx.BasicRes, connection *models.CursorConnection) (*TestConnectionResult, errors.Error) {
	if connection == nil {
		return nil, errors.BadInput.New("connection is required")
	}

	connection.Normalize()

	apiClient, err := helper.NewApiClientFromConnection(ctx, br, connection)
	if err != nil {
		return nil, err
	}

	// Use /teams/members to verify the API key — lightweight and works on all plans.
	resp, err := apiClient.Get("teams/members", nil, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to reach Cursor API")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.Unauthorized.New("invalid API key")
	}
	if resp.StatusCode >= 400 {
		return nil, errors.Default.New("Cursor API returned an unexpected error")
	}

	return &TestConnectionResult{
		Success: true,
		Message: "Connected to Cursor API successfully",
	}, nil
}
