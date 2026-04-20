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
	"context"
	"net/http"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
	"github.com/apache/incubator-devlake/server/api/shared"
)

func testSalesforceConnection(connection *models.SalesforceConnection) errors.Error {
	apiClient, err := helper.NewApiClientFromConnection(context.Background(), basicRes, connection)
	if err != nil {
		return err
	}

	scope, err := querySalesforceOrganization(apiClient, connection.GetVersion())
	if err != nil {
		return err
	}
	if scope == nil || scope.Id == "" {
		return errors.Default.New("salesforce connection test did not resolve organization details")
	}

	return nil
}

func TestConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection, err := decodeConnectionBody(input.Body)
	if err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	if err := testSalesforceConnection(connection); err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}

	return &plugin.ApiResourceOutput{
		Body:   shared.ApiBody{Success: true, Message: "success"},
		Status: http.StatusOK,
	}, nil
}

func TestExistingConnection(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	connection := &models.SalesforceConnection{}
	if err := connectionHelper.First(connection, input.Params); err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, errors.BadInput.Wrap(err, "find connection from db"))
	}
	if err := helper.DecodeMapStruct(input.Body, connection, false); err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}

	connection.Normalize()
	if err := validateConnection(connection); err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}
	if err := testSalesforceConnection(connection); err != nil {
		return nil, plugin.WrapTestConnectionErrResp(basicRes, err)
	}

	return &plugin.ApiResourceOutput{
		Body:   shared.ApiBody{Success: true, Message: "success"},
		Status: http.StatusOK,
	}, nil
}
