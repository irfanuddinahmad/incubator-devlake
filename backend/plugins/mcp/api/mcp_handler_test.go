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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLRejectsWriteCTEs(t *testing.T) {
	err := validateSQL("WITH x AS (DELETE\nFROM issues RETURNING *) SELECT * FROM x")
	require.EqualError(t, err, "statement contains disallowed keyword: DELETE")
}

func TestValidateSQLRejectsMultipleStatements(t *testing.T) {
	err := validateSQL("SELECT * FROM issues; SELECT * FROM pull_requests")
	require.EqualError(t, err, "multiple statements are not permitted")
}

func TestValidateSQLRejectsCopy(t *testing.T) {
	err := validateSQL("COPY (SELECT * FROM issues) TO '/tmp/out.csv'")
	require.EqualError(t, err, "statement contains disallowed keyword: COPY")
}

func TestValidateSQLAllowsKeywordInStringLiteral(t *testing.T) {
	err := validateSQL("SELECT * FROM issues WHERE title = 'DELETE this'")
	require.NoError(t, err)
}

func TestValidateSQLAllowsKeywordInBlockComment(t *testing.T) {
	err := validateSQL("SELECT/*DELETE*/1")
	require.NoError(t, err)
}

func TestValidateSQLHandlesUTF8Literals(t *testing.T) {
	err := validateSQL("SELECT * FROM issues WHERE title = '关闭 DELETE'")
	require.NoError(t, err)
}

func TestBuildSchemaRegistryIncludesCurrentDomainTables(t *testing.T) {
	issues, ok := schemaRegistry["issues"]
	require.True(t, ok)
	require.Equal(t, "ticket", issues.Domain)
	require.Equal(t, "Issues, tickets, stories, bugs, tasks", issues.Description)
	require.Contains(t, issues.Columns, "fix_versions")

	deployments, ok := schemaRegistry["cicd_deployments"]
	require.True(t, ok)
	require.Equal(t, "devops", deployments.Domain)
	require.Equal(t, "CI/CD deployments", deployments.Description)
	require.Contains(t, deployments.Columns, "original_environment")

	userActivities, ok := schemaRegistry["user_activities"]
	require.True(t, ok)
	require.Equal(t, "crossdomain", userActivities.Domain)

	qaCases, ok := schemaRegistry["qa_test_cases"]
	require.True(t, ok)
	require.Equal(t, "qa", qaCases.Domain)

	pullRequests, ok := schemaRegistry["pull_requests"]
	require.True(t, ok)
	require.Equal(t, "code", pullRequests.Domain)
}

func TestDetectDomainReturnsUnknownForNil(t *testing.T) {
	require.Equal(t, "unknown", detectDomain(nil))
}

func TestRunListTablesIncludesQADomain(t *testing.T) {
	result := runListTables(map[string]interface{}{"domain": "qa"})
	require.False(t, result.IsError)
	require.Len(t, result.Content, 1)
	require.Contains(t, result.Content[0].Text, "qa_test_cases")
}

func TestRunGetSchemaReflectsCurrentColumns(t *testing.T) {
	result := runGetSchema(map[string]interface{}{"tables": []interface{}{"pull_requests"}})
	require.False(t, result.IsError)
	require.Len(t, result.Content, 1)
	require.Contains(t, result.Content[0].Text, "merged_by_name")
	require.Contains(t, result.Content[0].Text, "head_commit_sha")
	require.True(t, strings.Contains(result.Content[0].Text, "pull_requests"))
}
