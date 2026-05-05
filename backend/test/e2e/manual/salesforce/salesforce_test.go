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

package salesforce

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apache/incubator-devlake/core/config"
	"github.com/apache/incubator-devlake/core/dal"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	salesforceimpl "github.com/apache/incubator-devlake/plugins/salesforce/impl"
	salesforcemodels "github.com/apache/incubator-devlake/plugins/salesforce/models"
	"github.com/apache/incubator-devlake/test/helper"
	"github.com/stretchr/testify/require"
)

const (
	pluginName               = "salesforce"
	allowUnsafeDatabaseEnv   = "SALESFORCE_E2E_ALLOW_UNSAFE_DB"
	defaultSalesforceTestDb  = "lake_test"
	unsafeDatabaseOverrideOn = "true"
)

func TestSalesforcePlugin(t *testing.T) {
	cfg := withDefaults(helper.GetTestConfig[TestConfig]())
	require.NotEmpty(t, cfg.AuthMode, "AuthMode must be provided via helper.SetTestConfig(TestConfig{...})")
	require.NotEmpty(t, cfg.ApiVersion, "ApiVersion must be provided via helper.SetTestConfig(TestConfig{...})")
	validateAuthConfig(t, cfg)

	config.GetConfig().Set("DISABLED_REMOTE_PLUGINS", true)
	dbURL := config.GetConfig().GetString("E2E_DB_URL")
	require.NoError(t, validateSafeE2EDatabaseURL(dbURL, os.Getenv(allowUnsafeDatabaseEnv)))

	client := helper.ConnectLocalServer(t, &helper.LocalClientConfig{
		ServerPort:   8093,
		DbURL:        dbURL,
		CreateServer: true,
		DropDb:       true,
		TruncateDb:   true,
		Plugins: []plugin.PluginMeta{
			salesforceimpl.Salesforce{},
		},
		Timeout:         30 * time.Second,
		PipelineTimeout: 10 * time.Minute,
	})

	connection := createConnection(t, cfg, client)
	var (
		scopeConfig  salesforcemodels.SalesforceScopeConfig
		createdScope *salesforcemodels.SalesforceScope
		projectName  string
		blueprint    coreModels.Blueprint
	)
	defer func() {
		if projectName != "" {
			client.DeleteProject(projectName)
			blueprint.ID = 0
		}
		if blueprint.ID != 0 {
			client.DeleteBlueprint(blueprint.ID)
		}
		if createdScope != nil {
			client.DeleteScope(pluginName, connection.ID, createdScope.ScopeId(), false)
		}
		if scopeConfig.ID != 0 {
			client.DeleteScopeConfig(pluginName, connection.ID, scopeConfig.ID)
		}
		if connection != nil {
			client.DeleteConnection(pluginName, connection.ID)
		}
	}()

	scopeConfig = helper.Cast[salesforcemodels.SalesforceScopeConfig](client.CreateScopeConfig(pluginName, connection.ID,
		salesforcemodels.SalesforceScopeConfig{
			ScopeConfig: common.ScopeConfig{
				Name:     "salesforce-default",
				Entities: []string{plugin.DOMAIN_TYPE_CROSS},
			},
			Name:        "salesforce-default",
			ObjectTypes: cfg.ObjectTypes,
			UseCdc:      false,
		},
	))

	remoteScopes := client.RemoteScopes(helper.RemoteScopesQuery{
		PluginName:   pluginName,
		ConnectionId: connection.ID,
	})
	require.NotEmpty(t, remoteScopes.Children, "expected Salesforce remote scope discovery to return an org scope")

	scopePayload := make([]any, 0, 1)
	for _, remoteScope := range remoteScopes.Children {
		if remoteScope.Type != "scope" {
			continue
		}
		scope := helper.Cast[salesforcemodels.SalesforceScope](remoteScope.Data)
		scope.ConnectionId = connection.ID
		scope.ScopeConfigId = scopeConfig.ID
		scopePayload = append(scopePayload, scope)
		break
	}
	require.NotEmpty(t, scopePayload, "expected at least one Salesforce org scope")

	createdScopes := helper.Cast[[]*salesforcemodels.SalesforceScope](client.CreateScopes(pluginName, connection.ID, scopePayload...))
	require.Len(t, createdScopes, 1)
	createdScope = createdScopes[0]

	project := client.CreateProject(&helper.ProjectConfig{
		ProjectName: fmt.Sprintf("project-%s-%d", pluginName, time.Now().Unix()),
		EnableDora:  false,
	})
	projectName = project.Name

	timeAfter := parseOptionalTime(t, cfg.OccurredAfter, "OccurredAfter")
	require.NotNil(t, project.Blueprint, "project should create a default blueprint")
	blueprint = client.PatchBasicBlueprintV2(project.Blueprint.ID, connection.Name, &helper.BlueprintV2Config{
		Connection: &coreModels.BlueprintConnection{
			PluginName:   pluginName,
			ConnectionId: connection.ID,
			Scopes: []*coreModels.BlueprintScope{
				{ScopeId: createdScope.ScopeId()},
			},
		},
		TimeAfter:   timeAfter,
		SkipOnFail:  false,
		ProjectName: project.Name,
	})

	project = client.GetProject(project.Name)
	require.NotNil(t, project.Blueprint, "project should reference the created blueprint")
	require.Equal(t, blueprint.ID, project.Blueprint.ID)

	pipeline := client.TriggerBlueprint(blueprint.ID)
	require.Equal(t, coreModels.TASK_COMPLETED, pipeline.Status)

	assertSalesforceActivities(t, client, connection.ID, createdScope.ScopeId())
}

func createConnection(t *testing.T, cfg TestConfig, client *helper.DevlakeClient) *helper.Connection {
	t.Helper()

	conn := salesforcemodels.SalesforceConnection{
		BaseConnection: api.BaseConnection{
			Name: "salesforce-conn",
		},
		SalesforceConn: salesforcemodels.SalesforceConn{
			RestConnection: api.RestConnection{
				Endpoint:         connectionEndpoint(cfg),
				Proxy:            "",
				RateLimitPerHour: 5000,
			},
			AuthMode:     cfg.AuthMode,
			AccessToken:  cfg.AccessToken,
			RefreshToken: cfg.RefreshToken,
			ClientId:     cfg.ClientId,
			ClientSecret: cfg.ClientSecret,
			LoginUrl:     cfg.LoginUrl,
			InstanceUrl:  cfg.InstanceUrl,
			ApiVersion:   cfg.ApiVersion,
		},
	}
	client.TestConnection(pluginName, conn)
	return client.CreateConnection(pluginName, conn)
}

func assertSalesforceActivities(t *testing.T, client *helper.DevlakeClient, connectionId uint64, scopeId string) {
	t.Helper()

	var activities []crossdomain.UserActivity
	err := client.GetDal().All(
		&activities,
		dal.Where("source_system = ? AND connection_id = ? AND scope_id = ?", pluginName, connectionId, scopeId),
		dal.Orderby("action_time ASC"),
	)
	require.NoError(t, err)
	require.NotEmpty(t, activities, "expected Salesforce pipeline to write rows into user_activities")

	hasRequiredActivityFields := false
	hasActorIdentity := false
	for _, activity := range activities {
		if activity.ActionType != "" &&
			activity.ObjectType != "" &&
			activity.ObjectId != "" &&
			!activity.ActionTime.IsZero() &&
			activity.Summary != "" {
			hasRequiredActivityFields = true
		}
		if activity.NativeUserId != "" || activity.UserEmail != "" {
			hasActorIdentity = true
		}
	}
	require.True(t, hasRequiredActivityFields, "expected at least one Salesforce activity with action/object/time/summary fields")
	require.True(t, hasActorIdentity, "expected at least one Salesforce activity with native_user_id or user_email")
}

func withDefaults(cfg TestConfig) TestConfig {
	cfg.AuthMode = strings.TrimSpace(strings.ToLower(cfg.AuthMode))
	if cfg.AuthMode == "" {
		cfg.AuthMode = salesforcemodels.AuthModeAccessToken
	}
	if strings.TrimSpace(cfg.ApiVersion) == "" {
		cfg.ApiVersion = salesforcemodels.DefaultApiVersion
	}
	if strings.TrimSpace(cfg.LoginUrl) == "" {
		cfg.LoginUrl = salesforcemodels.DefaultEndpoint
	}
	if len(cfg.ObjectTypes) == 0 {
		cfg.ObjectTypes = []string{"Lead", "Opportunity", "Case"}
	}
	return cfg
}

func validateAuthConfig(t *testing.T, cfg TestConfig) {
	t.Helper()

	switch cfg.AuthMode {
	case salesforcemodels.AuthModeRefreshToken:
		require.NotEmpty(t, cfg.RefreshToken, "RefreshToken must be provided for refresh_token auth")
		require.NotEmpty(t, cfg.ClientId, "ClientId must be provided for refresh_token auth")
		require.NotEmpty(t, cfg.ClientSecret, "ClientSecret must be provided for refresh_token auth")
	case salesforcemodels.AuthModeAccessToken:
		require.NotEmpty(t, cfg.AccessToken, "AccessToken must be provided for access_token auth")
		require.NotEmpty(t, cfg.InstanceUrl, "InstanceUrl must be provided for access_token auth")
	default:
		require.Failf(t, "unsupported AuthMode", "AuthMode must be %q or %q, got %q",
			salesforcemodels.AuthModeAccessToken,
			salesforcemodels.AuthModeRefreshToken,
			cfg.AuthMode,
		)
	}
}

func connectionEndpoint(cfg TestConfig) string {
	if strings.TrimSpace(cfg.InstanceUrl) != "" {
		return strings.TrimSpace(cfg.InstanceUrl)
	}
	return strings.TrimSpace(cfg.LoginUrl)
}

func parseOptionalTime(t *testing.T, raw string, fieldName string) *time.Time {
	t.Helper()

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	require.NoErrorf(t, err, "%s must be an RFC3339 timestamp", fieldName)
	return &parsed
}

func validateSafeE2EDatabaseURL(rawURL string, allowUnsafe string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("E2E_DB_URL must point to a disposable test database")
	}
	if strings.EqualFold(strings.TrimSpace(allowUnsafe), unsafeDatabaseOverrideOn) {
		return nil
	}

	dbName, err := databaseNameFromURL(rawURL)
	if err != nil {
		return err
	}
	if !isSafeE2EDatabaseName(dbName) {
		return fmt.Errorf(
			"refusing to run destructive Salesforce e2e test against database %q; use %q or set %s=%s to override",
			dbName,
			defaultSalesforceTestDb,
			allowUnsafeDatabaseEnv,
			unsafeDatabaseOverrideOn,
		)
	}
	return nil
}

func databaseNameFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("E2E_DB_URL is invalid: %w", err)
	}
	dbName := strings.Trim(parsed.Path, "/")
	if dbName == "" {
		return "", fmt.Errorf("E2E_DB_URL must include a database name")
	}
	parts := strings.Split(dbName, "/")
	return parts[len(parts)-1], nil
}

func isSafeE2EDatabaseName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized == defaultSalesforceTestDb ||
		strings.HasSuffix(normalized, "_test") ||
		strings.HasPrefix(normalized, "test_") ||
		strings.Contains(normalized, "_test_")
}
