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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/plane/models"
	"github.com/go-playground/validator/v10"
)

var basicRes context.BasicRes
var vld *validator.Validate
var connHelper *helper.ConnectionApiHelper
var connApi *helper.ModelApiHelper[models.PlaneConnection]
var dsHelper *helper.DsHelper[models.PlaneConnection, models.PlaneProject, models.PlaneScopeConfig]
var raProxy *helper.DsRemoteApiProxyHelper[models.PlaneConnection]
var raScopeList *helper.DsRemoteApiScopeListHelper[models.PlaneConnection, models.PlaneProject, PlaneRemotePagination]

func Init(br context.BasicRes, p plugin.PluginMeta) {
	basicRes = br
	vld = validator.New()
	connHelper = helper.NewConnectionHelper(br, vld, p.Name())
	dsHelper = helper.NewDataSourceHelper[
		models.PlaneConnection, models.PlaneProject, models.PlaneScopeConfig,
	](
		br,
		p.Name(),
		[]string{"name"},
		func(c models.PlaneConnection) models.PlaneConnection {
			return c.Sanitize()
		},
		func(s models.PlaneProject) models.PlaneProject {
			return s
		},
		nil,
	)
	raProxy = helper.NewDsRemoteApiProxyHelper[models.PlaneConnection](dsHelper.ConnApi.ModelApiHelper)
	raScopeList = helper.NewDsRemoteApiScopeListHelper[models.PlaneConnection, models.PlaneProject, PlaneRemotePagination](raProxy, listPlaneRemoteScopes)
	connApi = dsHelper.ConnApi.ModelApiHelper
}
