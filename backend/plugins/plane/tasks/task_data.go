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
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
)

type PlaneOptions struct {
	ConnectionId uint64 `json:"connectionId" mapstructure:"connectionId"`
	ProjectId    string `json:"projectId" mapstructure:"projectId"`
}

type PlaneTaskData struct {
	Options   *PlaneOptions
	Project   *models.PlaneProject
	ApiClient *helper.ApiAsyncClient
	Endpoint  string // base URL of the Plane instance, e.g. "https://api.plane.so"
}

// RAW_PROJECT_TABLE is the raw data table for Plane project API responses.
// Used by the project collector, extractor, and convertor.
const RAW_PROJECT_TABLE = "plane_api_projects"
const RAW_WORK_ITEM_TABLE = "plane_api_work_items"
const RAW_STATE_TABLE = "plane_api_states"
const RAW_WORK_ITEM_TYPE_TABLE = "plane_api_work_item_types"

// PlaneApiParams holds the identifiers used to scope raw data storage and retrieval.
type PlaneApiParams struct {
	ConnectionId  uint64
	WorkspaceSlug string
	ProjectId     string
}
