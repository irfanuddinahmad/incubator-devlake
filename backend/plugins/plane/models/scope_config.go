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
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
)

type TypeMapping struct {
	StandardType string `json:"standardType"`
}

type PlaneScopeConfig struct {
	common.ScopeConfig `mapstructure:",squash" json:",inline" gorm:"embedded"`
	TypeMappings       map[string]TypeMapping `mapstructure:"typeMappings,omitempty" json:"typeMappings" gorm:"type:json;serializer:json"`
	StateMappings      map[string]string      `mapstructure:"stateMappings,omitempty" json:"stateMappings" gorm:"type:json;serializer:json"`
}

func (PlaneScopeConfig) TableName() string {
	return "_tool_plane_scope_configs"
}

func (r *PlaneScopeConfig) SetConnectionId(c *PlaneScopeConfig, connectionId uint64) {
	c.ConnectionId = connectionId
	c.ScopeConfig.ConnectionId = connectionId
}

func (r *PlaneScopeConfig) Validate() errors.Error {
	if r.ConnectionId == 0 {
		return errors.BadInput.New("ConnectionId is required")
	}
	return nil
}
