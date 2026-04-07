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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/notion/models"
	"github.com/apache/incubator-devlake/plugins/notion/models/migrationscripts"
	"github.com/apache/incubator-devlake/plugins/notion/tasks"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginTask
	plugin.PluginModel
	plugin.PluginSource
	plugin.PluginMigration
} = (*Notion)(nil)

type Notion struct{}

func (p Notion) Description() string {
	return "Collect Notion activity events for daily user reporting"
}

func (p Notion) Name() string {
	return "notion"
}

func (p Notion) Connection() dal.Tabler {
	return &models.NotionConnection{}
}

func (p Notion) Scope() plugin.ToolLayerScope {
	return &models.NotionScope{}
}

func (p Notion) ScopeConfig() dal.Tabler {
	return &models.NotionScopeConfig{}
}

func (p Notion) GetTablesInfo() []dal.Tabler {
	return models.GetTablesInfo()
}

func (p Notion) SubTaskMetas() []plugin.SubTaskMeta {
	return tasks.GetSubTaskMetas()
}

func (p Notion) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
	var op tasks.NotionOptions
	if err := helper.Decode(options, &op, nil); err != nil {
		return nil, err
	}

	connectionHelper := helper.NewConnectionHelper(taskCtx, nil, p.Name())
	connection := &models.NotionConnection{}
	if err := connectionHelper.FirstById(connection, op.ConnectionId); err != nil {
		return nil, err
	}

	connection.Normalize()

	return &tasks.NotionTaskData{
		Options:    &op,
		Connection: connection,
	}, nil
}

func (p Notion) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/notion"
}

func (p Notion) MigrationScripts() []plugin.MigrationScript {
	return migrationscripts.All()
}
