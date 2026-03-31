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

package api

import (
	"github.com/go-playground/validator/v10"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/claude/models"
)

var (
	basicRes         context.BasicRes
	vld              *validator.Validate
	connectionHelper *helper.ConnectionApiHelper
	dsHelper         *helper.DsHelper[models.ClaudeConnection, models.ClaudeScope, models.ClaudeScopeConfig]
	raProxy          *helper.DsRemoteApiProxyHelper[models.ClaudeConnection]
	raScopeList      *helper.DsRemoteApiScopeListHelper[models.ClaudeConnection, models.ClaudeScope, ClaudeRemotePagination]
	raScopeSearch    *helper.DsRemoteApiScopeSearchHelper[models.ClaudeConnection, models.ClaudeScope]
)

// Init stores basic resources and configures shared helpers for API handlers.
func Init(br context.BasicRes, meta plugin.PluginMeta) {
	basicRes = br
	vld = validator.New()
	connectionHelper = helper.NewConnectionHelper(basicRes, vld, meta.Name())
	dsHelper = helper.NewDataSourceHelper[
		models.ClaudeConnection, models.ClaudeScope, models.ClaudeScopeConfig,
	](
		basicRes,
		meta.Name(),
		[]string{"id", "organizationId"},
		func(c models.ClaudeConnection) models.ClaudeConnection {
			c.Normalize()
			return c.Sanitize()
		},
		func(s models.ClaudeScope) models.ClaudeScope { return s },
		nil,
	)
	raProxy = helper.NewDsRemoteApiProxyHelper[models.ClaudeConnection](dsHelper.ConnApi.ModelApiHelper)
	raScopeList = helper.NewDsRemoteApiScopeListHelper[models.ClaudeConnection, models.ClaudeScope, ClaudeRemotePagination](raProxy, listClaudeRemoteScopes)
	raScopeSearch = helper.NewDsRemoteApiScopeSearchHelper[models.ClaudeConnection, models.ClaudeScope](raProxy, searchClaudeRemoteScopes)
}
