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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	corectx "github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/claude/models"
)

// TestConnectionResult represents the payload returned by the connection test endpoints.
type TestConnectionResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	OrganizationId string `json:"organizationId,omitempty"`
}

// TestConnection exercises the Anthropic API to validate credentials.
func TestConnection(ctx stdctx.Context, br corectx.BasicRes, connection *models.ClaudeConnection) (*TestConnectionResult, errors.Error) {
	if connection == nil {
		return nil, errors.BadInput.New("connection is required")
	}

	connection.Normalize()

	apiClient, err := helper.NewApiClientFromConnection(ctx, br, connection)
	if err != nil {
		return nil, err
	}

	// Validate against the Claude Code analytics endpoint so the test reflects
	// the same Admin API permission path used by the collector.
	query := url.Values{}
	query.Set("starting_at", time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02"))
	res, err := apiClient.Get("organizations/usage_report/claude_code", query, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		msg := "Successfully connected to Anthropic Claude Code Analytics API"
		if connection.OrganizationId != "" {
			msg = fmt.Sprintf("%s (organization: %s)", msg, connection.OrganizationId)
		}
		return &TestConnectionResult{
			Success:        true,
			Message:        msg,
			OrganizationId: connection.OrganizationId,
		}, nil
	case http.StatusUnauthorized:
		body, _ := io.ReadAll(res.Body)
		return nil, errors.HttpStatus(401).New(fmt.Sprintf("Unauthorized: invalid API key. Details: %s", string(body)))
	case http.StatusForbidden:
		body, _ := io.ReadAll(res.Body)
		return nil, errors.HttpStatus(403).New(fmt.Sprintf("Forbidden: insufficient permissions. Details: %s", string(body)))
	default:
		body, _ := io.ReadAll(res.Body)
		return nil, errors.HttpStatus(res.StatusCode).New(fmt.Sprintf("Anthropic API request failed with status %d. Details: %s", res.StatusCode, string(body)))
	}
}
