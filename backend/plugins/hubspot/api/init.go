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
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
)

var basicRes context.BasicRes
var vld *validator.Validate
var connectionHelper *helper.ConnectionApiHelper
var dsHelper *helper.DsHelper[models.HubspotConnection, models.HubspotScope, models.HubspotScopeConfig]
var raProxy *helper.DsRemoteApiProxyHelper[models.HubspotConnection]
var raScopeList *helper.DsRemoteApiScopeListHelper[models.HubspotConnection, models.HubspotScope, HubspotRemotePagination]
var raScopeSearch *helper.DsRemoteApiScopeSearchHelper[models.HubspotConnection, models.HubspotScope]

func Init(br context.BasicRes, meta plugin.PluginMeta) {
	basicRes = br
	vld = validator.New()
	connectionHelper = helper.NewConnectionHelper(basicRes, vld, meta.Name())
	dsHelper = helper.NewDataSourceHelper[
		models.HubspotConnection, models.HubspotScope, models.HubspotScopeConfig,
	](
		basicRes,
		meta.Name(),
		[]string{"id", "name"},
		func(c models.HubspotConnection) models.HubspotConnection {
			c.Normalize()
			return c.Sanitize()
		},
		func(s models.HubspotScope) models.HubspotScope { return s },
		nil,
	)
	raProxy = helper.NewDsRemoteApiProxyHelper[models.HubspotConnection](dsHelper.ConnApi.ModelApiHelper)
	raScopeList = helper.NewDsRemoteApiScopeListHelper[models.HubspotConnection, models.HubspotScope, HubspotRemotePagination](raProxy, listHubspotRemoteScopes)
	raScopeSearch = helper.NewDsRemoteApiScopeSearchHelper[models.HubspotConnection, models.HubspotScope](raProxy, searchHubspotRemoteScopes)
}
