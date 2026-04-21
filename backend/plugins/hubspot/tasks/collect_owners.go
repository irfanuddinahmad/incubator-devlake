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
	"strconv"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
)

var _ plugin.SubTaskEntryPoint = CollectOwners

var CollectOwnersMeta = plugin.SubTaskMeta{
	Name:             "collectOwners",
	EntryPoint:       CollectOwners,
	EnabledByDefault: true,
	Description:      "Collect HubSpot owners for actor identity and email enrichment",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

type hubspotOwnerRecord struct {
	Id                      string `json:"id"`
	Email                   string `json:"email"`
	FirstName               string `json:"firstName"`
	LastName                string `json:"lastName"`
	UserId                  *int64 `json:"userId"`
	UserIdIncludingInactive *int64 `json:"userIdIncludingInactive"`
}

type hubspotOwnersResponse struct {
	Results []hubspotOwnerRecord `json:"results"`
	Paging  struct {
		Next struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

func CollectOwners(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*HubspotTaskData)
	if !ok {
		return errors.Default.New("task data is not HubspotTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId
	after := ""

	for {
		url := "crm/v3/owners?limit=500&archived=true"
		if strings.TrimSpace(after) != "" {
			url += "&after=" + after
		}

		res, apiErr := apiClient.Get(url, nil, nil)
		if apiErr != nil {
			return apiErr
		}

		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return errors.Default.Wrap(readErr, "failed to read HubSpot owners response")
		}
		if res.StatusCode != http.StatusOK {
			return errors.Default.New(fmt.Sprintf("HubSpot owners API returned status %d: %s", res.StatusCode, string(body)))
		}

		var envelope hubspotOwnersResponse
		if err := json.Unmarshal(body, &envelope); err != nil {
			return errors.Default.Wrap(err, "failed to parse HubSpot owners response")
		}

		for _, owner := range envelope.Results {
			ownerId := strings.TrimSpace(owner.Id)
			if ownerId == "" {
				continue
			}
			userId := resolveHubspotOwnerUserId(owner)
			firstName := strings.TrimSpace(owner.FirstName)
			lastName := strings.TrimSpace(owner.LastName)
			fullName := strings.TrimSpace(strings.TrimSpace(firstName + " " + lastName))
			if fullName == "" {
				fullName = strings.TrimSpace(owner.Email)
			}

			record := &models.HubspotOwner{
				ConnectionId: connectionId,
				OwnerId:      ownerId,
				UserId:       userId,
				Email:        strings.TrimSpace(owner.Email),
				FirstName:    firstName,
				LastName:     lastName,
				FullName:     fullName,
			}
			if saveErr := db.CreateOrUpdate(record, dal.Where("connection_id = ? AND owner_id = ?", connectionId, ownerId)); saveErr != nil {
				return saveErr
			}
		}

		after = strings.TrimSpace(envelope.Paging.Next.After)
		if after == "" {
			break
		}
	}

	return nil
}

func resolveHubspotOwnerUserId(owner hubspotOwnerRecord) string {
	if owner.UserIdIncludingInactive != nil {
		return strconv.FormatInt(*owner.UserIdIncludingInactive, 10)
	}
	if owner.UserId != nil {
		return strconv.FormatInt(*owner.UserId, 10)
	}
	return ""
}
