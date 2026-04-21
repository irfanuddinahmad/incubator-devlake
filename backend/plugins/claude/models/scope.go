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
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
)

// ClaudeScope represents a Claude organization scope, keyed by OrganizationId.
type ClaudeScope struct {
	common.Scope   `mapstructure:",squash"`
	Id             string `json:"id" mapstructure:"id" gorm:"primaryKey;type:varchar(255)"`
	OrganizationId string `json:"organizationId" mapstructure:"organizationId" gorm:"type:varchar(255)"`
	Name           string `json:"name" mapstructure:"name" gorm:"type:varchar(255)"`
}

func (ClaudeScope) TableName() string {
	return "_tool_claude_scopes"
}

func (s ClaudeScope) ScopeId() string {
	return s.Id
}

func (s ClaudeScope) ScopeName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.Id
}

func (s ClaudeScope) ScopeFullName() string {
	return s.ScopeName()
}

func (s ClaudeScope) ScopeParams() interface{} {
	return &ClaudeScopeParams{
		ConnectionId:   s.ConnectionId,
		ScopeId:        s.Id,
		OrganizationId: s.OrganizationId,
	}
}

// ClaudeScopeParams is returned for blueprint configuration.
type ClaudeScopeParams struct {
	ConnectionId   uint64 `json:"connectionId"`
	ScopeId        string `json:"scopeId"`
	OrganizationId string `json:"organizationId"`
}

var _ plugin.ToolLayerScope = (*ClaudeScope)(nil)
