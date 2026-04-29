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

package models

import (
	"fmt"

	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
)

type PlaneProject struct {
	common.Scope  `mapstructure:",squash"`
	ProjectId     string `json:"projectId" mapstructure:"projectId" gorm:"primaryKey;type:varchar(255)"`
	Name          string `json:"name" mapstructure:"name" gorm:"type:varchar(255)"`
	Identifier    string `json:"identifier" mapstructure:"identifier" gorm:"type:varchar(255)"`
	Description   string `json:"description" mapstructure:"description" gorm:"type:text"`
	Network       int    `json:"network" mapstructure:"network"`
	WorkspaceSlug string `json:"workspaceSlug" mapstructure:"workspaceSlug" gorm:"type:varchar(255)"`
}

func (PlaneProject) TableName() string {
	return "_tool_plane_projects"
}

func (p PlaneProject) ScopeId() string {
	return p.ProjectId
}

func (p PlaneProject) ScopeName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.ProjectId
}

func (p PlaneProject) ScopeFullName() string {
	if p.WorkspaceSlug == "" {
		return p.ScopeName()
	}
	if p.Identifier != "" {
		return fmt.Sprintf("%s/%s", p.WorkspaceSlug, p.Identifier)
	}
	return fmt.Sprintf("%s/%s", p.WorkspaceSlug, p.ScopeName())
}

func (p PlaneProject) ScopeParams() interface{} {
	return &plugin.ApiResourceInput{
		Params: map[string]string{
			"connectionId": fmt.Sprintf("%d", p.ConnectionId),
			"projectId":    p.ProjectId,
		},
	}
}

var _ plugin.ToolLayerScope = (*PlaneProject)(nil)
