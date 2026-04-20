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
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
)

var basicRes context.BasicRes
var vld *validator.Validate
var connectionHelper *helper.ConnectionApiHelper
var dsHelper *helper.DsHelper[models.SalesforceConnection, models.SalesforceScope, models.SalesforceScopeConfig]
var raProxy *helper.DsRemoteApiProxyHelper[models.SalesforceConnection]
var raScopeList *helper.DsRemoteApiScopeListHelper[models.SalesforceConnection, models.SalesforceScope, SalesforceRemotePagination]
var raScopeSearch *helper.DsRemoteApiScopeSearchHelper[models.SalesforceConnection, models.SalesforceScope]

func Init(br context.BasicRes, meta plugin.PluginMeta) {
	basicRes = br
	vld = validator.New()
	connectionHelper = helper.NewConnectionHelper(basicRes, vld, meta.Name())
	dsHelper = helper.NewDataSourceHelper[
		models.SalesforceConnection, models.SalesforceScope, models.SalesforceScopeConfig,
	](
		basicRes,
		meta.Name(),
		[]string{"id", "name"},
		func(c models.SalesforceConnection) models.SalesforceConnection {
			c.Normalize()
			return c.Sanitize()
		},
		func(s models.SalesforceScope) models.SalesforceScope { return s },
		nil,
	)
	raProxy = helper.NewDsRemoteApiProxyHelper[models.SalesforceConnection](dsHelper.ConnApi.ModelApiHelper)
	raScopeList = helper.NewDsRemoteApiScopeListHelper[models.SalesforceConnection, models.SalesforceScope, SalesforceRemotePagination](raProxy, listSalesforceRemoteScopes)
	raScopeSearch = helper.NewDsRemoteApiScopeSearchHelper[models.SalesforceConnection, models.SalesforceScope](raProxy, searchSalesforceRemoteScopes)
}
