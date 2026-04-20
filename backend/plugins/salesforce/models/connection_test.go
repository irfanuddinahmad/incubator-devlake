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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSalesforceConnection_BeforeSaveAccessTokenValidation(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeAccessToken, AccessToken: "  ", InstanceUrl: "https://example.my.salesforce.com"}}
	err := c.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accessToken")

	c = &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeAccessToken, AccessToken: "token", InstanceUrl: "https://example.my.salesforce.com"}}
	assert.NoError(t, c.BeforeSave(nil))
}

func TestSalesforceConnection_BeforeSaveRefreshTokenValidation(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeRefreshToken, RefreshToken: "refresh", ClientId: "", ClientSecret: "secret"}}
	err := c.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clientId")

	c = &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeRefreshToken, RefreshToken: "refresh", ClientId: "client", ClientSecret: "secret"}}
	assert.NoError(t, c.BeforeSave(nil))
}
