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

package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAuthorizeURL(t *testing.T) {
	authURL := buildAuthorizeURL(
		"https://example.my.salesforce.com/",
		"client-id",
		"http://localhost:1717/callback",
		"api refresh_token",
		"state-value",
	)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	require.Equal(t, "https", parsed.Scheme)
	require.Equal(t, "example.my.salesforce.com", parsed.Host)
	require.Equal(t, "/services/oauth2/authorize", parsed.Path)
	require.Equal(t, "code", parsed.Query().Get("response_type"))
	require.Equal(t, "client-id", parsed.Query().Get("client_id"))
	require.Equal(t, "http://localhost:1717/callback", parsed.Query().Get("redirect_uri"))
	require.Equal(t, "api refresh_token", parsed.Query().Get("scope"))
	require.Equal(t, "state-value", parsed.Query().Get("state"))
}

func TestRenderLocalConfig(t *testing.T) {
	content, err := renderLocalConfig(localConfig{
		AuthMode:      "refresh_token",
		AccessToken:   "access-token",
		RefreshToken:  "refresh-token",
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		LoginURL:      "https://login.salesforce.com",
		InstanceURL:   "https://example.my.salesforce.com",
		APIVersion:    "v61.0",
		ObjectTypes:   []string{"Lead", "Opportunity"},
		OccurredAfter: "2026-04-01T00:00:00Z",
	})
	require.NoError(t, err)

	rendered := string(content)
	require.Contains(t, rendered, "package salesforce")
	require.Contains(t, rendered, "helper.SetTestConfig(TestConfig{")
	require.Contains(t, rendered, `AuthMode:      "refresh_token"`)
	require.Contains(t, rendered, `RefreshToken:  "refresh-token"`)
	require.Contains(t, rendered, `ClientSecret:  "client-secret"`)
	require.Contains(t, rendered, `InstanceUrl:   "https://example.my.salesforce.com"`)
	require.Contains(t, rendered, `ObjectTypes:   []string{"Lead", "Opportunity"}`)
	require.Contains(t, rendered, `OccurredAfter: "2026-04-01T00:00:00Z"`)
	require.NotContains(t, rendered, "OccurredBefore")
}

func TestParseObjectTypesTrimsAndDeduplicates(t *testing.T) {
	objectTypes, err := parseObjectTypes("Lead, Opportunity,Lead, Case")
	require.NoError(t, err)
	require.Equal(t, []string{"Lead", "Opportunity", "Case"}, objectTypes)
}

func TestValidateOptionsRequiresCredentials(t *testing.T) {
	err := validateOptions(options{
		loginURL:     defaultLoginURL,
		callbackPort: 1717,
		timeout:      10,
		objectTypes:  "Lead",
	})

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "client-id"))
}

func TestNormalizeURLRequiresHTTPS(t *testing.T) {
	_, err := normalizeURL("https://login.salesforce.com")
	require.NoError(t, err)

	_, err = normalizeURL("http://login.salesforce.com")
	require.ErrorContains(t, err, "https")
}
