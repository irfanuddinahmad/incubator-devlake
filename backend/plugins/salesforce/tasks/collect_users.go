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
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
)

var _ plugin.SubTaskEntryPoint = CollectUsers

var CollectUsersMeta = plugin.SubTaskMeta{
	Name:             "collectUsers",
	EntryPoint:       CollectUsers,
	EnabledByDefault: true,
	Description:      "Collect Salesforce users for actor identity and email enrichment",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func CollectUsers(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*SalesforceTaskData)
	if !ok {
		return errors.Default.New("task data is not SalesforceTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()
	maxUsers := data.Options.MaxUsers
	if maxUsers <= 0 {
		maxUsers = DefaultMaxUsers
	}

	soql := buildSalesforceUsersQuery()
	next := ""
	collected := 0

	for {
		response, _, err := executeSalesforceQuery(apiClient, data.Connection.GetVersion(), soql, next)
		if err != nil {
			return err
		}

		for _, row := range response.Records {
			var user struct {
				Id       string `json:"Id"`
				Name     string `json:"Name"`
				Username string `json:"Username"`
				Email    string `json:"Email"`
				IsActive bool   `json:"IsActive"`
			}
			if err := json.Unmarshal(row, &user); err != nil {
				return errors.Convert(err)
			}
			userId := strings.TrimSpace(user.Id)
			if userId == "" {
				continue
			}

			record := &models.SalesforceUser{
				ConnectionId: data.Options.ConnectionId,
				UserId:       userId,
				Name:         strings.TrimSpace(user.Name),
				Username:     strings.TrimSpace(user.Username),
				Email:        strings.TrimSpace(user.Email),
				IsActive:     user.IsActive,
			}
			if saveErr := db.CreateOrUpdate(record, dal.Where("connection_id = ? AND user_id = ?", data.Options.ConnectionId, userId)); saveErr != nil {
				return saveErr
			}
			collected++
			if collected >= maxUsers {
				logger.Warn(nil, "salesforce user collection hit MaxUsers cap of %d; further users will be skipped", maxUsers)
				return nil
			}
		}

		next = strings.TrimSpace(response.NextRecordsURL)
		if next == "" {
			break
		}
	}

	logger.Info("salesforce user collection completed: %d users persisted", collected)
	return nil
}

func buildSalesforceUsersQuery() string {
	return "SELECT Id, Name, Username, Email, IsActive FROM User WHERE IsActive = true ORDER BY Id ASC"
}
