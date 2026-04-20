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
	"testing"

	"github.com/apache/incubator-devlake/plugins/salesforce/models"
	"github.com/stretchr/testify/require"
	"github.com/go-playground/validator/v10"
)

func TestDecodeConnectionBodyDerivesEndpointForAccessTokenMode(t *testing.T) {
	vld = validator.New()

	connection, err := decodeConnectionBody(map[string]interface{}{
		"name":        "sf-access",
		"authMode":    "access_token",
		"accessToken": "token",
		"instanceUrl": "https://org.example.my.salesforce.com",
	})
	require.NoError(t, err)
	require.Equal(t, models.AuthModeAccessToken, connection.AuthMode)
	require.Equal(t, "https://org.example.my.salesforce.com", connection.Endpoint)
}

func TestDecodeConnectionBodyDerivesEndpointForRefreshTokenMode(t *testing.T) {
	vld = validator.New()

	connection, err := decodeConnectionBody(map[string]interface{}{
		"name":         "sf-refresh",
		"authMode":     "refresh_token",
		"loginUrl":     "https://login.salesforce.com",
		"clientId":     "client-id",
		"clientSecret": "client-secret",
		"refreshToken": "refresh-token",
	})
	require.NoError(t, err)
	require.Equal(t, models.AuthModeRefreshToken, connection.AuthMode)
	require.Equal(t, "https://login.salesforce.com", connection.Endpoint)
}
