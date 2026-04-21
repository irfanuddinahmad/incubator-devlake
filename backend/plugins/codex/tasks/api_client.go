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

package tasks

import (
	"fmt"
	"net/http"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

func CreateApiClient(taskCtx plugin.TaskContext, connection *models.CodexConnection) (*helper.ApiAsyncClient, errors.Error) {
	apiClient, err := helper.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, connection)
	if err != nil {
		return nil, err
	}

	// Set OpenAI Bearer token auth
	apiClient.SetBeforeFunction(func(req *http.Request) errors.Error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", connection.ApiKey))
		return nil
	})

	rateLimiter := &helper.ApiRateLimitCalculator{
		UserRateLimitPerHour: connection.RateLimitPerHour,
	}

	asyncApiClient, err := helper.CreateAsyncApiClient(taskCtx, apiClient, rateLimiter)
	if err != nil {
		return nil, err
	}

	return asyncApiClient, nil
}
