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
	"io"
	"net/http"
	neturl "net/url"

	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

type planeApiMember struct {
	Id          string `json:"id"`
	FirstName   string `json:"first_name"`
	DisplayName string `json:"display_name"`
}

func fetchPlaneMemberMap(
	apiClient *helper.ApiAsyncClient,
	workspaceSlug string,
	projectId string,
) (map[string]string, errors.Error) {
	projectMembers, err := fetchPlaneMembers(
		apiClient,
		"api/v1/workspaces/"+neturl.PathEscape(workspaceSlug)+"/projects/"+neturl.PathEscape(projectId)+"/members/",
	)
	if err == nil && len(projectMembers) > 0 {
		return projectMembers, nil
	}

	workspaceMembers, workspaceErr := fetchPlaneMembers(
		apiClient,
		"api/v1/workspaces/"+neturl.PathEscape(workspaceSlug)+"/members/",
	)
	if workspaceErr == nil {
		return workspaceMembers, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, workspaceErr
}

func FetchPlaneMemberMapForTaskData(
	apiClient *helper.ApiAsyncClient,
	workspaceSlug string,
	projectId string,
) (map[string]string, errors.Error) {
	return fetchPlaneMemberMap(apiClient, workspaceSlug, projectId)
}

func fetchPlaneMembers(apiClient *helper.ApiAsyncClient, path string) (map[string]string, errors.Error) {
	response, err := apiClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, errors.Unauthorized.New("authentication failed, please check your Plane API key")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.HttpStatus(response.StatusCode).New("error fetching Plane members")
	}

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, errors.Default.Wrap(readErr, "error reading Plane members response body")
	}

	var members []planeApiMember
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, errors.Default.Wrap(err, "error unmarshalling Plane members response")
	}

	memberMap := make(map[string]string, len(members))
	for _, member := range members {
		if member.Id == "" {
			continue
		}
		name := member.FirstName
		if name == "" {
			name = member.DisplayName
		}
		if name == "" {
			continue
		}
		memberMap[member.Id] = name
	}
	return memberMap, nil
}

func resolvePlanePrimaryAssignee(assignees []planeApiAssignee, assigneeNameById map[string]string) (string, string) {
	if len(assignees) == 0 {
		return "", ""
	}

	assigneeId := assignees[0].Id
	assigneeName := assignees[0].Name
	if assigneeName == "" && assigneeId != "" {
		assigneeName = assigneeNameById[assigneeId]
	}
	return assigneeId, assigneeName
}
