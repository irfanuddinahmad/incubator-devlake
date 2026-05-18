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
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/mcp/api"
)

var _ interface {
	plugin.PluginMeta
	plugin.PluginInit
	plugin.PluginApi
} = (*Mcp)(nil)

type Mcp struct{}

func (p Mcp) Name() string { return "mcp" }

func (p Mcp) Description() string {
	return "MCP server exposing DevLake normalized data via read-only SQL queries"
}

func (p Mcp) RootPkgPath() string {
	return "github.com/apache/incubator-devlake/plugins/mcp"
}

func (p Mcp) Init(basicRes context.BasicRes) errors.Error {
	api.Init(basicRes)
	return nil
}

func (p Mcp) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
	return map[string]map[string]plugin.ApiResourceHandler{
		// MCP Streamable HTTP endpoint (stateless JSON-RPC)
		"mcp": {
			"POST": api.McpHandler,
			"GET":  api.McpHandler,
		},
	}
}
