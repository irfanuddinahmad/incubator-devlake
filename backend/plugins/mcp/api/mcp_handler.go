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
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/domaininfo"
	"github.com/apache/incubator-devlake/core/plugin"
	"gorm.io/gorm/schema"
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

var schemaRegistry = buildSchemaRegistry()

var tableDescriptions = map[string]string{
	"repos":                    "Git repositories",
	"commits":                  "Git commits",
	"pull_requests":            "Pull requests / merge requests",
	"pull_request_commits":     "Join: pull requests and commits",
	"pull_request_labels":      "Labels attached to pull requests",
	"pull_request_reviewers":   "Reviewers assigned to pull requests",
	"pull_request_comments":    "Comments on pull requests",
	"refs":                     "Git branches and tags",
	"repo_commits":             "Join: repos and commits",
	"commit_files":             "Files changed per commit",
	"issues":                   "Issues, tickets, stories, bugs, tasks",
	"issue_labels":             "Labels on issues",
	"issue_changelogs":         "History of field changes on issues",
	"issue_worklogs":           "Time logged against issues",
	"issue_comments":           "Comments on issues",
	"boards":                   "Issue boards / Kanban boards / projects",
	"board_issues":             "Join: boards and issues",
	"sprints":                  "Agile sprints",
	"sprint_issues":            "Join: sprints and issues",
	"cicd_scopes":              "CI/CD project scopes",
	"cicd_pipelines":           "CI/CD pipeline runs",
	"cicd_tasks":               "Individual tasks/jobs within a pipeline",
	"cicd_deployments":         "CI/CD deployments",
	"cicd_deployment_commits":  "Deployments linked to specific commits",
	"cicd_pipeline_commits":    "Commits that triggered pipelines",
	"cicd_releases":            "CI/CD releases",
	"users":                    "Unified user identities across tools",
	"accounts":                 "Tool-specific user accounts",
	"user_accounts":            "Join: users and accounts",
	"user_activities":          "User activity events across tools",
	"teams":                    "Organizational teams",
	"team_users":               "Join: teams and users",
	"pull_request_issues":      "Links pull requests to the issues they fix",
	"issue_commits":            "Links issues to related commits",
	"project_mapping":          "Maps DevLake projects to data scopes",
	"incidents":                "Production incidents",
	"incident_assignees":       "Assignees on incidents",
	"cq_projects":              "SonarQube / code quality analysis projects",
	"cq_issues":                "Code quality issues / violations",
	"cq_file_metrics":          "File-level code quality metrics",
	"qa_apis":                  "QA API definitions",
	"qa_test_cases":            "QA test cases",
	"qa_test_case_executions":  "QA test case execution results",
	"qa_test_case_issue_links": "Links QA test cases to issues",
	"ai_activities":            "AI coding assistant usage metrics",
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
						"description": "Optional filter: 'code', 'ticket', 'devops', 'crossdomain', 'codequality', 'qa', 'ai', or 'unknown'. Omit to list all tables.",
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

	domains := []string{"code", "ticket", "devops", "crossdomain", "codequality", "qa", "ai", "unknown"}
	found := false
	for _, domain := range domains {
		tables, ok := byDomain[domain]
		if !ok {
			continue
		}
		found = true
		sort.Strings(tables)
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
	normalized := normalizeSQLForValidation(sql)
	tokens := strings.Fields(normalized)
	if len(tokens) == 0 {
		return fmt.Errorf("only SELECT (or WITH ... SELECT) statements are permitted")
	}
	for _, token := range tokens {
		if token == "INSERT" || token == "UPDATE" || token == "DELETE" || token == "DROP" ||
			token == "CREATE" || token == "ALTER" || token == "TRUNCATE" || token == "RENAME" ||
			token == "REPLACE" || token == "MERGE" || token == "EXEC" || token == "EXECUTE" ||
			token == "CALL" || token == "GRANT" || token == "REVOKE" || token == "LOAD" ||
			token == "COPY" {
			return fmt.Errorf("statement contains disallowed keyword: %s", token)
		}
	}
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i] == "INTO" && tokens[i+1] == "OUTFILE" {
			return fmt.Errorf("statement contains disallowed keyword: INTO OUTFILE")
		}
	}
	if tokens[0] != "SELECT" && tokens[0] != "WITH" {
		return fmt.Errorf("only SELECT (or WITH ... SELECT) statements are permitted")
	}
	if strings.ContainsRune(normalized, ';') {
		return fmt.Errorf("multiple statements are not permitted")
	}
	return nil
}

func buildSchemaRegistry() map[string]tableInfo {
	registry := make(map[string]tableInfo)
	schemaCache := &sync.Map{}
	for _, table := range domaininfo.GetDomainTablesInfo() {
		parsedSchema, err := schema.Parse(table, schemaCache, schema.NamingStrategy{})
		if err != nil {
			panic(fmt.Sprintf("buildSchemaRegistry: failed to parse schema for %T: %v", table, err))
		}
		columns := make([]string, 0, len(parsedSchema.Fields))
		seen := make(map[string]struct{}, len(parsedSchema.Fields))
		for _, field := range parsedSchema.Fields {
			if field.DBName == "" {
				continue
			}
			if _, exists := seen[field.DBName]; exists {
				continue
			}
			seen[field.DBName] = struct{}{}
			columns = append(columns, field.DBName)
		}
		description := tableDescriptions[table.TableName()]
		if description == "" {
			description = fmt.Sprintf("Domain layer table %s", table.TableName())
		}
		registry[table.TableName()] = tableInfo{
			Domain:      detectDomain(table),
			Description: description,
			Columns:     strings.Join(columns, ", "),
		}
	}
	return registry
}

func detectDomain(table interface{}) string {
	if table == nil {
		return "unknown"
	}
	tableType := reflect.TypeOf(table)
	if tableType.Kind() == reflect.Ptr {
		tableType = tableType.Elem()
	}
	pkgPath := tableType.PkgPath()
	switch {
	case strings.Contains(pkgPath, "/codequality"):
		return "codequality"
	case strings.Contains(pkgPath, "/crossdomain"):
		return "crossdomain"
	case strings.Contains(pkgPath, "/devops"):
		return "devops"
	case strings.Contains(pkgPath, "/ticket"):
		return "ticket"
	case strings.Contains(pkgPath, "/qa"):
		return "qa"
	case strings.Contains(pkgPath, "/ai"):
		return "ai"
	case strings.Contains(pkgPath, "/code"):
		return "code"
	default:
		return "unknown"
	}
}

func normalizeSQLForValidation(sql string) string {
	var sb strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				sb.WriteByte(' ')
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
				sb.WriteByte(' ')
			}
			continue
		}
		if inSingleQuote {
			if ch == '\'' {
				if next == '\'' {
					i++
					continue
				}
				inSingleQuote = false
				sb.WriteByte(' ')
			}
			continue
		}
		if inDoubleQuote {
			if ch == '"' {
				if next == '"' {
					i++
					continue
				}
				inDoubleQuote = false
				sb.WriteByte(' ')
			}
			continue
		}
		if inBacktick {
			if ch == '`' {
				inBacktick = false
				sb.WriteByte(' ')
			}
			continue
		}
		if ch == '-' && next == '-' {
			inLineComment = true
			i++
			sb.WriteByte(' ')
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			sb.WriteByte(' ')
			continue
		}
		if ch == '\'' {
			inSingleQuote = true
			sb.WriteByte(' ')
			continue
		}
		if ch == '"' {
			inDoubleQuote = true
			sb.WriteByte(' ')
			continue
		}
		if ch == '`' {
			inBacktick = true
			sb.WriteByte(' ')
			continue
		}
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == ';' {
			sb.WriteRune(unicode.ToUpper(ch))
			continue
		}
		sb.WriteByte(' ')
	}
	return sb.String()
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
