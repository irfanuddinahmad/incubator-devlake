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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTokenExpiry_UsesExpiresInWithSafetyBuffer(t *testing.T) {
	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)

	expiry := computeTokenExpiry([]byte(`3600`), now)
	assert.Equal(t, now.Add(3600*time.Second-60*time.Second), expiry)

	// Salesforce sometimes returns expires_in as a JSON string.
	expiry = computeTokenExpiry([]byte(`"7200"`), now)
	assert.Equal(t, now.Add(7200*time.Second-60*time.Second), expiry)
}

func TestComputeTokenExpiry_FallsBackOnMissingOrInvalid(t *testing.T) {
	now := time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)
	fallback := now.Add(25 * time.Minute)

	assert.Equal(t, fallback, computeTokenExpiry(nil, now))
	assert.Equal(t, fallback, computeTokenExpiry([]byte(``), now))
	assert.Equal(t, fallback, computeTokenExpiry([]byte(`"not-a-number"`), now))
	assert.Equal(t, fallback, computeTokenExpiry([]byte(`-1`), now))
}

func TestSanitizeOAuthError_PrefersStructuredFields(t *testing.T) {
	body := []byte(`{"error":"invalid_grant","error_description":"expired authorization code","refresh_token":"sneaky"}`)
	got := sanitizeOAuthError(body)
	assert.Equal(t, "invalid_grant: expired authorization code", got)
	assert.NotContains(t, got, "sneaky", "raw body fields must not leak through")
}

func TestSanitizeOAuthError_FallsBackForUnparseable(t *testing.T) {
	got := sanitizeOAuthError([]byte("<html>something exploded</html>"))
	assert.Equal(t, "unexpected response from salesforce oauth endpoint", got)
}

func TestSalesforceConnection_BeforeSaveAccessTokenValidation(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeAccessToken, AccessToken: "  ", InstanceUrl: "https://example.my.salesforce.com"}}
	err := c.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accessToken")

	c = &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeAccessToken, AccessToken: "token", InstanceUrl: "https://example.my.salesforce.com"}}
	assert.NoError(t, c.BeforeSave(nil))
}

func TestSalesforceConnection_BeforeSaveAccessTokenRejectsInstanceUrlWithoutScheme(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{
		AuthMode:    AuthModeAccessToken,
		AccessToken: "token",
		InstanceUrl: "example.my.salesforce.com",
	}}

	err := c.BeforeSave(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceUrl must start with https://")
}

func TestSalesforceConnection_BeforeSaveRefreshTokenValidation(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeRefreshToken, RefreshToken: "refresh", ClientId: "", ClientSecret: "secret"}}
	err := c.BeforeSave(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clientId")

	c = &SalesforceConnection{SalesforceConn: SalesforceConn{AuthMode: AuthModeRefreshToken, RefreshToken: "refresh", ClientId: "client", ClientSecret: "secret"}}
	assert.NoError(t, c.BeforeSave(nil))
}

func TestSalesforceConnection_BeforeSaveRefreshTokenRejectsLoginUrlWithoutScheme(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{
		AuthMode:     AuthModeRefreshToken,
		RefreshToken: "refresh",
		ClientId:     "client",
		ClientSecret: "secret",
		LoginUrl:     "login.salesforce.com",
	}}

	err := c.BeforeSave(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loginUrl must start with https://")
}

func TestSalesforceConnection_BeforeSaveRefreshTokenRejectsInstanceUrlWithoutHttps(t *testing.T) {
	c := &SalesforceConnection{SalesforceConn: SalesforceConn{
		AuthMode:     AuthModeRefreshToken,
		RefreshToken: "refresh",
		ClientId:     "client",
		ClientSecret: "secret",
		LoginUrl:     "https://login.salesforce.com",
		InstanceUrl:  "http://example.my.salesforce.com",
	}}

	err := c.BeforeSave(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instanceUrl must start with https://")
}

func TestSalesforceConn_SetupAuthenticationRefreshesOnFirstUse(t *testing.T) {
	refreshCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/services/oauth2/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		require.Equal(t, "client", r.Form.Get("client_id"))
		require.Equal(t, "secret", r.Form.Get("client_secret"))
		require.Equal(t, "refresh", r.Form.Get("refresh_token"))
		refreshCalled = true
		_, _ = fmt.Fprint(w, `{"access_token":"fresh-token","instance_url":"https://fresh.my.salesforce.com"}`)
	}))
	defer server.Close()

	conn := &SalesforceConn{
		AuthMode:     AuthModeRefreshToken,
		AccessToken:  "stale-token",
		RefreshToken: "refresh",
		ClientId:     "client",
		ClientSecret: "secret",
		LoginUrl:     server.URL,
		InstanceUrl:  "https://old.my.salesforce.com",
	}
	req, err := http.NewRequest(http.MethodGet, "https://old.my.salesforce.com/services/data/v61.0/query", nil)
	require.NoError(t, err)

	require.NoError(t, conn.SetupAuthentication(req))
	require.True(t, refreshCalled)
	require.Equal(t, "Bearer fresh-token", req.Header.Get("Authorization"))
	require.Equal(t, "fresh.my.salesforce.com", req.URL.Host)
	require.False(t, conn.tokenExpiresAt.IsZero())
}

func TestSalesforceConn_SetupAuthenticationReusesUnexpiredRefreshTokenModeAccessToken(t *testing.T) {
	conn := &SalesforceConn{
		AuthMode:       AuthModeRefreshToken,
		AccessToken:    "fresh-token",
		RefreshToken:   "refresh",
		ClientId:       "client",
		ClientSecret:   "secret",
		LoginUrl:       "http://127.0.0.1:1",
		InstanceUrl:    "https://fresh.my.salesforce.com",
		tokenExpiresAt: time.Now().Add(time.Hour),
	}
	req, err := http.NewRequest(http.MethodGet, "https://fresh.my.salesforce.com/services/data/v61.0/query", nil)
	require.NoError(t, err)

	require.NoError(t, conn.SetupAuthentication(req))
	require.Equal(t, "Bearer fresh-token", req.Header.Get("Authorization"))
}
