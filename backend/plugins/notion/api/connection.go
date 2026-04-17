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
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

func PostConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection := &models.NotionConnection{}
	if err := helper.Decode(input.Body, connection, vld); err != nil {
		return nil, err
	}
	connection.Normalize()
	if err := validateConnection(connection); err != nil {
		return nil, err
	}
	if err := connectionHelper.Create(connection, input); err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{Body: connection.Sanitize()}, nil
}

func PatchConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection := &models.NotionConnection{}
	if err := connectionHelper.First(connection, input.Params); err != nil {
		return nil, err
	}
	if err := (&models.NotionConnection{}).MergeFromRequest(connection, input.Body); err != nil {
		return nil, errors.Convert(err)
	}
	connection.Normalize()
	if err := validateConnection(connection); err != nil {
		return nil, err
	}
	if err := connectionHelper.SaveWithCreateOrUpdate(connection); err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{Body: connection.Sanitize()}, nil
}

func DeleteConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection := &models.NotionConnection{}
	output, err := connectionHelper.Delete(connection, input)
	if err != nil {
		return output, err
	}
	output.Body = connection.Sanitize()
	return output, nil
}

func ListConnections(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	_ = input
	var connections []models.NotionConnection
	if err := connectionHelper.List(&connections); err != nil {
		return nil, err
	}
	for i := range connections {
		connections[i] = connections[i].Sanitize()
	}
	return &plugin.ApiResourceOutput{Body: connections}, nil
}

func GetConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection := &models.NotionConnection{}
	if err := connectionHelper.First(connection, input.Params); err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{Body: connection.Sanitize()}, nil
}

func validateConnection(connection *models.NotionConnection) errors.Error {
	if connection == nil {
		return errors.BadInput.New("connection is required")
	}
	if strings.TrimSpace(connection.ApiToken) == "" {
		return errors.BadInput.New("token is required")
	}
	return nil
}
