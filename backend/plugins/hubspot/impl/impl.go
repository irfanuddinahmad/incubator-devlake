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

package impl

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/hubspot/api"
	"github.com/apache/incubator-devlake/plugins/hubspot/models"
	"github.com/apache/incubator-devlake/plugins/hubspot/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/hubspot/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginApi
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginSource
	plugin.PluginMigration
} = (*Hubspot)(nil)

type Hubspot struct{}

func (p Hubspot) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes)
	return nil
}

func (p Hubspot) Description() string {
	return "Collect HubSpot activity events for daily user reporting"
}

func (p Hubspot) Name() string {
	return "hubspot"
}

func (p Hubspot) Connection() dal.Tabler {
	return &models.HubspotConnection{}
}

func (p Hubspot) Scope() plugin.ToolLayerScope {
	return &models.HubspotScope{}
}

func (p Hubspot) ScopeConfig() dal.Tabler {
	return &models.HubspotScopeConfig{}
}

func (p Hubspot) GetTablesInfo() []dal.Tabler {
	return models.GetTablesInfo()
}

func (p Hubspot) SubTaskMetas() []plugin.SubTaskMeta {
	return tasks.GetSubTaskMetas()
}

func (p Hubspot) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.HubspotOptions
	if err := helper.Decode(options, &op, nil); err != nil {
		return nil, err
	}

	connectionHelper := helper.NewConnectionHelper(taskCtx, nil, p.Name())
	connection := &models.HubspotConnection{}
	if err := connectionHelper.FirstById(connection, op.ConnectionId); err != nil {
		return nil, err
	}

	connection.Normalize()

	return &tasks.HubspotTaskData{
		Options:    &op,
		Connection: connection,
	}, nil
}

func (p Hubspot) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/hubspot"
}

func (p Hubspot) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}

func (p Hubspot) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		"connections/:connectionId/scopes/:scopeId/webhook": {
			"POST": api.PostWebhookEvents,
		},
	}
}
