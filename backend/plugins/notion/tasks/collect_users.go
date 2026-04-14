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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

var _ plugin.SubTaskEntryPoint = CollectUsers

var CollectUsersMeta = plugin.SubTaskMeta{
	Name:             "collectUsers",
	EntryPoint:       CollectUsers,
	EnabledByDefault: true,
	Description:      "Collect Notion workspace users via GET /v1/users for display name resolution",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

type notionUserRecord struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Person struct {
		Email string `json:"email"`
	} `json:"person"`
}

type notionUsersResponse struct {
	Results    []notionUserRecord `json:"results"`
	HasMore    bool               `json:"has_more"`
	NextCursor string             `json:"next_cursor"`
}

func CollectUsers(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*NotionTaskData)
	if !ok {
		return errors.Default.New("task data is not NotionTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId

	var cursor string
	for {
		url := "v1/users?page_size=100"
		if cursor != "" {
			url += "&start_cursor=" + cursor
		}

		res, apiErr := apiClient.Get(url, nil, nil)
		if apiErr != nil {
			return apiErr
		}
		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return errors.Default.Wrap(readErr, "failed to read Notion users response")
		}
		if res.StatusCode != http.StatusOK {
			return errors.Default.New(fmt.Sprintf("Notion users API returned status %d: %s", res.StatusCode, string(body)))
		}

		var envelope notionUsersResponse
		if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
			return errors.Default.Wrap(jsonErr, "failed to parse Notion users response")
		}

		for _, u := range envelope.Results {
			userId := strings.TrimSpace(u.Id)
			if userId == "" {
				continue
			}
			user := &models.NotionUser{
				ConnectionId: connectionId,
				UserId:       userId,
				Name:         strings.TrimSpace(u.Name),
				Email:        strings.TrimSpace(u.Person.Email),
				UserType:     strings.TrimSpace(u.Type),
			}
			if saveErr := db.CreateOrUpdate(user, dal.Where("connection_id = ? AND user_id = ?", connectionId, userId)); saveErr != nil {
				return saveErr
			}
		}

		if !envelope.HasMore || strings.TrimSpace(envelope.NextCursor) == "" {
			break
		}
		cursor = envelope.NextCursor
	}

	return nil
}
