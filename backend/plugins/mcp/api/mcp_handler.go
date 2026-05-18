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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

const mcpProtocolVersion = "2024-11-05"
const mcpServerName = "devlake-mcp"
const mcpServerVersion = "1.0.0"
const defaultRowLimit = 500

// ── JSON-RPC 2.0 wire types ──────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"` // string | number | null (omitted for notifications)
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── MCP protocol types ───────────────────────────────────────────────────────

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct {
		Tools struct{} `json:"tools"`
	} `json:"capabilities"`
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

type callToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ── Schema registry ──────────────────────────────────────────────────────────

type tableInfo struct {
	Domain      string
	Description string
	Columns     string
}

var schemaRegistry = map[string]tableInfo{
	// CODE
	"repos": {
		Domain:      "code",
		Description: "Git repositories",
		Columns:     "id, name, url, description, owner_id, language, created_date, updated_date, deleted",
	},
	"commits": {
		Domain:      "code",
		Description: "Git commits",
		Columns:     "sha, additions, deletions, message, author_name, author_email, authored_date, committer_name, committer_email, committed_date, author_id, committer_id",
	},
	"pull_requests": {
		Domain:      "code",
		Description: "Pull requests / merge requests",
		Columns:     "id, base_repo_id, head_repo_id, title, status, original_status, author_name, author_id, created_date, merged_date, closed_date, type, component, additions, deletions, is_draft, base_ref, head_ref, merge_commit_sha",
	},
	"pull_request_commits": {
		Domain:      "code",
		Description: "Join: pull requests ↔ commits",
		Columns:     "pull_request_id, commit_sha, commit_author_name, commit_author_email, commit_authored_date",
	},
	"pull_request_labels": {
		Domain:      "code",
		Description: "Labels attached to pull requests",
		Columns:     "pull_request_id, label_name",
	},
	"pull_request_reviewers": {
		Domain:      "code",
		Description: "Reviewers assigned to pull requests",
		Columns:     "pull_request_id, reviewer_id, name, user_name",
	},
	"pull_request_comments": {
		Domain:      "code",
		Description: "Comments on pull requests",
		Columns:     "id, pull_request_id, body, account_id, created_date, type, status",
	},
	"refs": {
		Domain:      "code",
		Description: "Git branches and tags",
		Columns:     "id, repo_id, name, commit_sha, is_default, ref_type, created_date",
	},
	"repo_commits": {
		Domain:      "code",
		Description: "Join: repos ↔ commits",
		Columns:     "repo_id, commit_sha",
	},
	"commit_files": {
		Domain:      "code",
		Description: "Files changed per commit",
		Columns:     "id, commit_sha, file_path, additions, deletions",
	},
	// TICKET
	"issues": {
		Domain:      "ticket",
		Description: "Issues, tickets, stories, bugs, tasks",
		Columns:     "id, title, type, original_type, status, original_status, priority, severity, story_point, created_date, updated_date, resolution_date, lead_time_minutes, assignee_id, assignee_name, creator_id, creator_name, parent_issue_id, is_subtask, component, epic_key, original_project, due_date",
	},
	"issue_labels": {
		Domain:      "ticket",
		Description: "Labels on issues",
		Columns:     "issue_id, label_name",
	},
	"issue_changelogs": {
		Domain:      "ticket",
		Description: "History of field changes on issues",
		Columns:     "id, issue_id, author_id, author_name, field_name, from_value, to_value, created_date",
	},
	"issue_worklogs": {
		Domain:      "ticket",
		Description: "Time logged against issues",
		Columns:     "id, issue_id, author_id, time_spent_minutes, logged_date, started_date",
	},
	"issue_comments": {
		Domain:      "ticket",
		Description: "Comments on issues",
		Columns:     "id, issue_id, body, account_id, created_date, updated_date",
	},
	"boards": {
		Domain:      "ticket",
		Description: "Issue boards / Kanban boards / projects",
		Columns:     "id, name, description, url, type, created_date",
	},
	"board_issues": {
		Domain:      "ticket",
		Description: "Join: boards ↔ issues",
		Columns:     "board_id, issue_id",
	},
	"sprints": {
		Domain:      "ticket",
		Description: "Agile sprints",
		Columns:     "id, name, url, status, started_date, ended_date, completed_date, original_board_id",
	},
	"sprint_issues": {
		Domain:      "ticket",
		Description: "Join: sprints ↔ issues",
		Columns:     "sprint_id, issue_id",
	},
	// DEVOPS
	"cicd_scopes": {
		Domain:      "devops",
		Description: "CI/CD project scopes (pipelines belong to a scope)",
		Columns:     "id, name, description, url, created_date, updated_date",
	},
	"cicd_pipelines": {
		Domain:      "devops",
		Description: "CI/CD pipeline runs",
		Columns:     "id, name, display_title, url, result, status, type, duration_sec, queued_duration_sec, environment, created_date, queued_date, started_date, finished_date, cicd_scope_id",
	},
	"cicd_tasks": {
		Domain:      "devops",
		Description: "Individual tasks/jobs within a pipeline",
		Columns:     "id, name, pipeline_id, result, status, type, duration_sec, environment, created_date, started_date, finished_date, cicd_scope_id",
	},
	"cicd_deployment_commits": {
		Domain:      "devops",
		Description: "Deployments linked to specific commits",
		Columns:     "id, commit_sha, cicd_scope_id, cicd_deployment_id, name, result, status, environment, started_date, finished_date, duration_sec, ref_name, repo_id, repo_url",
	},
	"cicd_pipeline_commits": {
		Domain:      "devops",
		Description: "Commits that triggered pipelines",
		Columns:     "pipeline_id, commit_sha, branch, repo_id, repo_url",
	},
	"cicd_releases": {
		Domain:      "devops",
		Description: "CI/CD releases / GitHub releases",
		Columns:     "id, name, tag_name, commit_sha, published_at, is_draft, is_latest, is_prerelease, cicd_scope_id, repo_id",
	},
	// CROSSDOMAIN
	"users": {
		Domain:      "crossdomain",
		Description: "Unified user identities across tools",
		Columns:     "id, email, name",
	},
	"accounts": {
		Domain:      "crossdomain",
		Description: "Tool-specific user accounts (GitHub user, Jira user, etc.)",
		Columns:     "id, email, full_name, user_name, avatar_url, organization, created_date, status",
	},
	"user_accounts": {
		Domain:      "crossdomain",
		Description: "Join: users ↔ accounts",
		Columns:     "user_id, account_id",
	},
	"teams": {
		Domain:      "crossdomain",
		Description: "Organizational teams",
		Columns:     "id, name, alias, parent_id, sorting_index",
	},
	"team_users": {
		Domain:      "crossdomain",
		Description: "Join: teams ↔ users",
		Columns:     "team_id, user_id",
	},
	"pull_request_issues": {
		Domain:      "crossdomain",
		Description: "Links pull requests to the issues they fix",
		Columns:     "pull_request_id, issue_id, pull_request_key, issue_key",
	},
	"issue_commits": {
		Domain:      "crossdomain",
		Description: "Links issues to related commits",
		Columns:     "issue_id, commit_sha",
	},
	"project_mapping": {
		Domain:      "crossdomain",
		Description: "Maps DevLake projects to data scopes (repos, boards, cicd scopes)",
		Columns:     "project_name, table, row_id",
	},
	// CODE QUALITY
	"cq_projects": {
		Domain:      "codequality",
		Description: "SonarQube / code quality analysis projects",
		Columns:     "id, name, qualifier, visibility, last_analysis_date, commit_sha",
	},
	"cq_issues": {
		Domain:      "codequality",
		Description: "Code quality issues / violations",
		Columns:     "id, rule, severity, component, project_key, type, scope, status, message, debt, effort, created_date, updated_date",
	},
	"cq_file_metrics": {
		Domain:      "codequality",
		Description: "File-level code quality metrics",
		Columns:     "id, project_key, file_name, (various metric columns)",
	},
	// AI
	"ai_activities": {
		Domain:      "ai",
		Description: "AI coding assistant usage metrics (GitHub Copilot, etc.)",
		Columns:     "id, provider, account_id, user_email, date, type, model, suggestions_count, acceptance_count, lines_added, lines_removed, commits_created, prs_created, input_tokens, output_tokens, estimated_cost_usd",
	},
}

// ── Tool definitions ─────────────────────────────────────────────────────────

func toolList() []toolDef {
	return []toolDef{
		{
			Name:        "list_tables",
			Description: "List all DevLake normalized domain tables with their domain and description. Call this first to discover available data.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"domain": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter: 'code', 'ticket', 'devops', 'crossdomain', 'codequality', or 'ai'. Omit to list all tables.",
					},
				},
			},
		},
		{
			Name:        "get_schema",
			Description: "Get the column details for one or more tables. Use this to understand the exact columns before writing a query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tables": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Table names to describe, e.g. [\"pull_requests\", \"issues\"]",
					},
				},
				"required": []string{"tables"},
			},
		},
		{
			Name: "execute_query",
			Description: `Execute a read-only SQL SELECT query against DevLake's normalized domain layer.
Only SELECT statements are permitted. Results are capped at 500 rows.

Common patterns:
- PR cycle time:    SELECT avg(TIMESTAMPDIFF(HOUR, created_date, merged_date)) FROM pull_requests WHERE status='MERGED'
- Deploy frequency: SELECT DATE(started_date) as day, count(*) as deploys FROM cicd_deployment_commits WHERE result='SUCCESS' GROUP BY day ORDER BY day DESC
- Issue throughput: SELECT DATE_FORMAT(resolution_date,'%Y-%u') as week, count(*) FROM issues WHERE resolution_date IS NOT NULL GROUP BY week ORDER BY week DESC
- DORA lead time:   SELECT avg(TIMESTAMPDIFF(HOUR, c.authored_date, dc.finished_date)) FROM cicd_deployment_commits dc JOIN commit_files cf ON cf.commit_sha=dc.commit_sha JOIN commits c ON c.sha=cf.commit_sha WHERE dc.result='SUCCESS'`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "A read-only SQL SELECT statement to execute.",
					},
				},
				"required": []string{"sql"},
			},
		},
	}
}

// ── HTTP handler entry point ─────────────────────────────────────────────────

func McpHandler(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	// Re-marshal the pre-parsed body map back to JSON so we can unmarshal it
	// into the typed rpcRequest struct (including json.RawMessage for params).
	bodyBytes, err := json.Marshal(input.Body)
	if err != nil {
		return jsonOut(400, errResp(nil, -32700, "parse error")), nil
	}

	var req rpcRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		return jsonOut(400, errResp(nil, -32700, "parse error")), nil
	}

	// Notifications have no id; handle without returning a JSON-RPC result body.
	if req.Method == "notifications/initialized" || req.Method == "notifications/cancelled" {
		return &plugin.ApiResourceOutput{Status: 202, Body: nil}, nil
	}

	resp := dispatch(&req)
	return jsonOut(200, resp), nil
}

// ── Dispatcher ───────────────────────────────────────────────────────────────

func dispatch(req *rpcRequest) *rpcResponse {
	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "tools/list":
		return handleToolsList(req)
	case "tools/call":
		return handleToolsCall(req)
	default:
		return errResp(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func handleInitialize(req *rpcRequest) *rpcResponse {
	var result initializeResult
	result.ProtocolVersion = mcpProtocolVersion
	result.ServerInfo.Name = mcpServerName
	result.ServerInfo.Version = mcpServerVersion
	return okResp(req.ID, result)
}

func handleToolsList(req *rpcRequest) *rpcResponse {
	return okResp(req.ID, toolsListResult{Tools: toolList()})
}

func handleToolsCall(req *rpcRequest) *rpcResponse {
	var p callToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return errResp(req.ID, -32602, "invalid params")
	}

	switch p.Name {
	case "list_tables":
		return okResp(req.ID, runListTables(p.Arguments))
	case "get_schema":
		return okResp(req.ID, runGetSchema(p.Arguments))
	case "execute_query":
		return okResp(req.ID, runExecuteQuery(p.Arguments))
	default:
		return okResp(req.ID, callToolResult{
			IsError: true,
			Content: textContent(fmt.Sprintf("unknown tool: %s", p.Name)),
		})
	}
}

// ── Tool implementations ─────────────────────────────────────────────────────

func runListTables(args map[string]interface{}) callToolResult {
	domainFilter := ""
	if d, ok := args["domain"].(string); ok {
		domainFilter = strings.ToLower(d)
	}

	var sb strings.Builder
	if domainFilter != "" {
		sb.WriteString(fmt.Sprintf("Tables in domain '%s':\n\n", domainFilter))
	} else {
		sb.WriteString("All DevLake normalized domain tables:\n\n")
	}

	byDomain := map[string][]string{}
	for name, info := range schemaRegistry {
		if domainFilter == "" || info.Domain == domainFilter {
			byDomain[info.Domain] = append(byDomain[info.Domain], name)
		}
	}

	domains := []string{"code", "ticket", "devops", "crossdomain", "codequality", "ai"}
	found := false
	for _, domain := range domains {
		tables, ok := byDomain[domain]
		if !ok {
			continue
		}
		found = true
		sb.WriteString(fmt.Sprintf("## %s\n", domain))
		for _, t := range tables {
			sb.WriteString(fmt.Sprintf("  %-35s %s\n", t, schemaRegistry[t].Description))
		}
		sb.WriteString("\n")
	}
	if !found {
		return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("no tables found for domain '%s'", domainFilter))}
	}

	return callToolResult{Content: textContent(sb.String())}
}

func runGetSchema(args map[string]interface{}) callToolResult {
	tablesRaw, ok := args["tables"]
	if !ok {
		return callToolResult{IsError: true, Content: textContent("'tables' argument required")}
	}
	tablesList, ok := tablesRaw.([]interface{})
	if !ok {
		return callToolResult{IsError: true, Content: textContent("'tables' must be an array")}
	}

	var sb strings.Builder
	for _, raw := range tablesList {
		name, ok := raw.(string)
		if !ok {
			continue
		}
		info, exists := schemaRegistry[name]
		if !exists {
			sb.WriteString(fmt.Sprintf("Table '%s': not found in schema registry\n\n", name))
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s  (domain: %s)\n", name, info.Domain))
		sb.WriteString(fmt.Sprintf("%s\n\n", info.Description))
		sb.WriteString(fmt.Sprintf("Columns: %s\n\n", info.Columns))
	}

	return callToolResult{Content: textContent(sb.String())}
}

func runExecuteQuery(args map[string]interface{}) callToolResult {
	sqlRaw, ok := args["sql"]
	if !ok {
		return callToolResult{IsError: true, Content: textContent("'sql' argument required")}
	}
	sqlStr, ok := sqlRaw.(string)
	if !ok {
		return callToolResult{IsError: true, Content: textContent("'sql' must be a string")}
	}

	sqlStr = strings.TrimSpace(sqlStr)
	if err := validateSQL(sqlStr); err != nil {
		return callToolResult{IsError: true, Content: textContent(err.Error())}
	}

	// Append LIMIT if the query doesn't already include one.
	if !strings.Contains(strings.ToUpper(sqlStr), " LIMIT ") {
		sqlStr = fmt.Sprintf("SELECT * FROM (%s) _q LIMIT %d", sqlStr, defaultRowLimit)
	}

	rows, dbErr := db.RawCursor(sqlStr)
	if dbErr != nil {
		return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("query error: %s", dbErr.Error()))}
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("failed to read columns: %s", err))}
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("scan error: %s", err))}
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			v := values[i]
			// Convert []byte (e.g. MySQL text/blob) to string for clean JSON output.
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[col] = v
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("row iteration error: %s", err))}
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return callToolResult{IsError: true, Content: textContent(fmt.Sprintf("JSON marshal error: %s", err))}
	}

	summary := fmt.Sprintf("// %d row(s) returned\n", len(results))
	return callToolResult{Content: textContent(summary + string(out))}
}

// validateSQL ensures only read-only SELECT statements are executed.
func validateSQL(sql string) error {
	upper := strings.ToUpper(sql)

	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("only SELECT (or WITH ... SELECT) statements are permitted")
	}

	// Block any write or DDL keywords.
	blocked := []string{
		"INSERT ", "UPDATE ", "DELETE ", "DROP ", "CREATE ", "ALTER ",
		"TRUNCATE ", "RENAME ", "REPLACE ", "MERGE ", "EXEC ", "EXECUTE ",
		"CALL ", "GRANT ", "REVOKE ", "LOAD ", "INTO OUTFILE",
	}
	for _, kw := range blocked {
		if strings.Contains(upper, kw) {
			return fmt.Errorf("statement contains disallowed keyword: %s", strings.TrimSpace(kw))
		}
	}

	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func okResp(id interface{}, result interface{}) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errResp(id interface{}, code int, message string) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func textContent(text string) []toolContent {
	return []toolContent{{Type: "text", Text: text}}
}

func jsonOut(status int, body interface{}) *plugin.ApiResourceOutput {
	return &plugin.ApiResourceOutput{Status: status, Body: body}
}
