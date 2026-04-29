---
name: devlake-plugin
description: Complete guide for implementing a new datasource plugin in Apache DevLake. Covers architecture, all required interfaces, three-layer data model, subtask patterns, migration scripts, API registration, blueprint planning, and known pitfalls. Use whenever a new DevLake plugin is being added.
origin: custom
---

# DevLake Plugin Development Guide

## When to Activate

- Implementing a new datasource plugin
- Extending an existing plugin with new entity types
- Debugging a plugin's data pipeline
- Reviewing a plugin PR

---

## Repository Layout (backend only)

```
backend/
├── core/
│   ├── plugin/                    # All plugin interfaces (PluginMeta, SubTaskMeta, etc.)
│   ├── models/domainlayer/        # Standardised domain models
│   │   ├── code/                  # Repo, Commit, PullRequest, Ref
│   │   ├── ticket/                # Board, Issue, BoardIssue, Sprint
│   │   ├── devops/                # CicdPipeline, CicdTask, CicdDeployment
│   │   └── didgen/                # Deterministic domain ID generator
│   └── dal/                       # Database abstraction layer
├── helpers/pluginhelper/api/      # Reusable helpers: ApiCollector, ApiExtractor, DataConverter
├── plugins/
│   └── {plugin-name}/             # One directory per plugin (see structure below)
└── server/                        # HTTP server; plugins auto-discovered via plugin.RegisterPlugin
```

---

## Canonical Plugin Directory Structure

```
backend/plugins/{plugin}/
├── {plugin}.go                          # Entrypoint: calls plugin.Register at init()
├── impl/
│   └── impl.go                          # Main struct, implements ALL plugin interfaces
├── models/
│   ├── connection.go                    # Connection + auth model
│   ├── {scope}.go                       # Scope model (e.g. project, repo, board)
│   ├── scope_config.go                  # ScopeConfig model
│   ├── {entity}.go                      # Tool-layer entity models (one file per entity)
│   └── migrationscripts/
│       ├── register.go                  # All() returns []MigrationScript
│       └── {timestamp}_{description}.go # Individual migration
├── tasks/
│   ├── task_data.go                     # Options + TaskData structs; DecodeAndValidate
│   ├── api_client.go                    # NewXxxApiClient factory
│   ├── {entity}_collector.go           # RAW collection: API → _raw_*
│   ├── {entity}_extractor.go           # Extraction: _raw_* → _tool_*
│   └── {entity}_convertor.go          # Conversion: _tool_* → domain tables
└── api/
    ├── init.go                          # DsHelper, proxy, scope list init
    ├── connection_api.go                # Connection CRUD handlers
    ├── scope_api.go                     # Scope CRUD handlers
    ├── scope_config_api.go              # ScopeConfig CRUD handlers
    ├── remote_api.go                    # Remote scope discovery
    └── blueprint_v200.go               # Pipeline plan generation
```

> **Reference plugin**: `backend/plugins/taiga/` — a full, working plugin with all layers.

---

## Interface Checklist — impl/impl.go

Every full datasource plugin implements all of these. Use a compile-time assertion:

```go
var _ interface {
    plugin.PluginMeta
    plugin.PluginInit
    plugin.PluginTask
    plugin.PluginApi
    plugin.PluginModel
    plugin.PluginMigration
    plugin.DataSourcePluginBlueprintV200
    plugin.CloseablePluginTask
    plugin.PluginSource
} = (*MyPlugin)(nil)
```

### Interface Definitions

| Interface | Methods Required |
|---|---|
| `PluginMeta` | `Name() string`, `Description() string`, `RootPkgPath() string` |
| `PluginInit` | `Init(basicRes context.BasicRes) errors.Error` |
| `PluginTask` | `SubTaskMetas() []SubTaskMeta`, `PrepareTaskData(...)` |
| `CloseablePluginTask` | `Close(taskCtx plugin.TaskContext) errors.Error` |
| `PluginModel` | `GetTablesInfo() []dal.Tabler` |
| `PluginMigration` | `MigrationScripts() []MigrationScript` |
| `PluginSource` | `Connection() dal.Tabler`, `Scope() ToolLayerScope`, `ScopeConfig() dal.Tabler` |
| `PluginApi` | `ApiResources() map[string]map[string]ApiResourceHandler` |
| `DataSourcePluginBlueprintV200` | `MakeDataSourcePipelinePlanV200(...)` |

### RootPkgPath

Must exactly match the Go module import path:

```go
func (p MyPlugin) RootPkgPath() string {
    return "github.com/apache/incubator-devlake/plugins/myplugin"
}
```

---

## Three-Layer Data Model

Every entity flows through three layers. **Never skip a layer.**

```
Remote API → _raw_{plugin}_api_{entity} → _tool_{plugin}_{entity} → domain table
              (Collector)                   (Extractor)               (Converter)
```

| Layer | Table prefix | Populated by | Contains |
|---|---|---|---|
| Raw | `_raw_` | Collector | Verbatim JSON blobs from the API |
| Tool | `_tool_` | Extractor | Typed Go structs (plugin-specific) |
| Domain | (no prefix) | Converter | Standardised cross-plugin models |

---

## Domain Types

Set `DomainTypes` on every `SubTaskMeta`:

```go
plugin.DOMAIN_TYPE_CODE        // repositories, commits, branches
plugin.DOMAIN_TYPE_TICKET      // issues, boards, sprints
plugin.DOMAIN_TYPE_CODE_REVIEW // pull/merge requests
plugin.DOMAIN_TYPE_CROSS       // issue-PR links
plugin.DOMAIN_TYPE_CICD        // pipelines, deployments
plugin.DOMAIN_TYPE_CODE_QUALITY
```

---

## Step-by-Step: Build a New Plugin

### Step 1 — Connection Model

```go
// models/connection.go
type MyConn struct {
    helper.RestConnection `mapstructure:",squash"` // Endpoint, Proxy, RateLimitPerHour
    // Choose ONE auth approach:
    ApiKey string `mapstructure:"apiKey" json:"apiKey" gorm:"serializer:encdec"`
    // OR:
    helper.BasicAuth `mapstructure:",squash"` // Username + Password
    // OR:
    helper.MultiAuth `mapstructure:",squash"` // Multiple auth methods
}

func (c *MyConn) SetupAuthentication(req *http.Request) errors.Error {
    req.Header.Set("X-Api-Key", c.ApiKey)
    return nil
}

func (c *MyConn) Sanitize() MyConn {
    c.ApiKey = utils.SanitizeString(c.ApiKey)
    return *c
}

type MyConnection struct {
    helper.BaseConnection `mapstructure:",squash"` // ID, Name, CreatedAt, UpdatedAt
    MyConn                `mapstructure:",squash"`
}

func (MyConnection) TableName() string { return "_tool_myplugin_connections" }

// Preserve existing secrets on PATCH — do NOT trust empty string from client
func (connection *MyConnection) MergeFromRequest(target *MyConnection, body map[string]interface{}) error {
    existing := target.ApiKey
    if err := helper.DecodeMapStruct(body, target, true); err != nil {
        return err
    }
    if target.ApiKey == "" || target.ApiKey == utils.SanitizeString(existing) {
        target.ApiKey = existing
    }
    return nil
}

func (connection MyConnection) Sanitize() MyConnection {
    connection.MyConn = connection.MyConn.Sanitize()
    return connection
}
```

### Step 2 — Scope Model

```go
// models/my_project.go
type MyProject struct {
    common.Scope     `mapstructure:",squash"`  // ConnectionId, ScopeConfigId
    ProjectId        string    `json:"projectId" gorm:"primaryKey"`
    Name             string    `json:"name"`
    Description      string    `json:"description"`
    Url              string    `json:"url"`
}

func (MyProject) TableName() string { return "_tool_myplugin_projects" }

func (p *MyProject) ScopeId() string           { return p.ProjectId }
func (p *MyProject) ScopeName() string         { return p.Name }
func (p *MyProject) ScopeFullName() string     { return p.Name }
func (p *MyProject) ScopeParams() interface{}  {
    return &MyApiParams{ConnectionId: p.ConnectionId, ProjectId: p.ProjectId}
}
func (p *MyProject) ScopeConnectionId() uint64     { return p.ConnectionId }
func (p *MyProject) ScopeScopeConfigId() uint64    { return p.ScopeConfigId }
```

### Step 3 — ScopeConfig Model

```go
// models/scope_config.go
type MyScopeConfig struct {
    common.ScopeConfig `mapstructure:",squash"` // ID, ConnectionId, Name, Entities
    // Plugin-specific enrichment:
    TypeMappings   map[string]TypeMapping `json:"typeMappings" gorm:"serializer:json"`
    // Deployment/production patterns (for CICD domain):
    DeploymentPattern  string `json:"deploymentPattern"`
    ProductionPattern  string `json:"productionPattern"`
}

func (MyScopeConfig) TableName() string { return "_tool_myplugin_scope_configs" }
```

### Step 4 — Tool-layer Entity Model

```go
// models/my_issue.go
type MyIssue struct {
    common.NoPKModel                          // CreatedAt, UpdatedAt, RawDataOrigin
    ConnectionId   uint64    `gorm:"primaryKey"`
    ProjectId      string    `gorm:"primaryKey"`
    IssueId        string    `gorm:"primaryKey"`
    Title          string
    Description    string
    Status         string
    IssueType      string
    Priority       string
    AssigneeId     string
    AssigneeName   string
    CreatedDate    *time.Time
    UpdatedDate    *time.Time
    ClosedDate     *time.Time
}

func (MyIssue) TableName() string { return "_tool_myplugin_issues" }
```

### Step 5 — Task Data & Options

```go
// tasks/task_data.go
type MyApiParams struct {
    ConnectionId uint64 `json:"connectionId"`
    ProjectId    string `json:"projectId"`
}

type MyOptions struct {
    ConnectionId  uint64           `json:"connectionId"  mapstructure:"connectionId"`
    ProjectId     string           `json:"projectId"     mapstructure:"projectId"`
    ScopeConfig   *models.MyScopeConfig `json:"scopeConfig" mapstructure:"scopeConfig"`
    ScopeConfigId uint64           `json:"scopeConfigId" mapstructure:"scopeConfigId"`
    PageSize      int              `json:"pageSize"      mapstructure:"pageSize"`
}

type MyTaskData struct {
    Options   *MyOptions
    ApiClient *api.ApiAsyncClient
}
```

### Step 6 — API Client Factory

```go
// tasks/api_client.go
func NewMyApiClient(taskCtx plugin.TaskContext, connection *models.MyConnection) (*api.ApiAsyncClient, errors.Error) {
    syncApiClient, err := api.NewApiClientFromConnection(context.TODO(), taskCtx, connection)
    if err != nil {
        return nil, err
    }
    asyncApiClient, err := api.CreateAsyncApiClient(taskCtx, syncApiClient, nil)
    if err != nil {
        return nil, err
    }
    return asyncApiClient, nil
}
```

### Step 7 — Collector (API → Raw)

```go
// tasks/issue_collector.go
const RAW_ISSUE_TABLE = "myplugin_api_issues"

var CollectIssuesMeta = plugin.SubTaskMeta{
    Name:             "collectIssues",
    EntryPoint:       CollectIssues,
    EnabledByDefault: true,
    Description:      "collect MyPlugin issues from remote API",
    DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectIssues(taskCtx plugin.SubTaskContext) errors.Error {
    data := taskCtx.GetData().(*MyTaskData)

    collector, err := api.NewApiCollector(api.ApiCollectorArgs{
        RawDataSubTaskArgs: api.RawDataSubTaskArgs{
            Ctx: taskCtx,
            Params: MyApiParams{
                ConnectionId: data.Options.ConnectionId,
                ProjectId:    data.Options.ProjectId,
            },
            Table: RAW_ISSUE_TABLE,
        },
        ApiClient:   data.ApiClient,
        PageSize:    100,
        // IMPORTANT: Implement real pagination. Do NOT rely on large page sizes.
        // For cursor-based APIs:
        UrlTemplate: "api/v1/workspaces/{{ .Params.WorkspaceSlug }}/projects/{{ .Params.ProjectId }}/issues/",
        Query: func(reqData *api.RequestData) (url.Values, errors.Error) {
            query := url.Values{}
            query.Set("per_page", strconv.Itoa(reqData.Pager.Size))
            query.Set("page", strconv.Itoa(reqData.Pager.Page))
            return query, nil
        },
        GetTotalPages: func(res *http.Response, args *api.ApiCollectorArgs) (int, errors.Error) {
            // Parse total from response headers or body
            body := &struct{ Count int `json:"count"` }{}
            api.UnmarshalResponse(res, body)
            return int(math.Ceil(float64(body.Count) / float64(args.PageSize))), nil
        },
        ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
            body := &struct{ Results []json.RawMessage `json:"results"` }{}
            err := api.UnmarshalResponse(res, body)
            return body.Results, err
        },
    })
    if err != nil {
        return err
    }
    return collector.Execute()
}
```

### Step 8 — Extractor (Raw → Tool Layer)

```go
// tasks/issue_extractor.go
var ExtractIssuesMeta = plugin.SubTaskMeta{
    Name:             "extractIssues",
    EntryPoint:       ExtractIssues,
    EnabledByDefault: true,
    Description:      "extract MyPlugin issues from raw data",
    DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ExtractIssues(taskCtx plugin.SubTaskContext) errors.Error {
    data := taskCtx.GetData().(*MyTaskData)

    extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
        RawDataSubTaskArgs: api.RawDataSubTaskArgs{
            Ctx: taskCtx,
            Params: MyApiParams{
                ConnectionId: data.Options.ConnectionId,
                ProjectId:    data.Options.ProjectId,
            },
            Table: RAW_ISSUE_TABLE,
        },
        Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
            // Define inline API struct matching remote JSON shape
            var apiIssue struct {
                Id          string     `json:"id"`
                Title       string     `json:"title"`
                Description string     `json:"description"`
                State       struct {
                    Name string `json:"name"`
                } `json:"state"`
                Priority    string     `json:"priority"`
                CreatedAt   *time.Time `json:"created_at"`
                UpdatedAt   *time.Time `json:"updated_at"`
                CompletedAt *time.Time `json:"completed_at"`
                Assignees   []struct {
                    Id          string `json:"id"`
                    DisplayName string `json:"display_name"`
                } `json:"assignees"`
            }
            if err := json.Unmarshal(row.Data, &apiIssue); err != nil {
                return nil, errors.Default.Wrap(err, "unmarshalling issue")
            }

            issue := &models.MyIssue{
                ConnectionId: data.Options.ConnectionId,
                ProjectId:    data.Options.ProjectId,
                IssueId:      apiIssue.Id,
                Title:        apiIssue.Title,
                Description:  apiIssue.Description,
                Status:       apiIssue.State.Name,
                Priority:     apiIssue.Priority,
                CreatedDate:  apiIssue.CreatedAt,
                UpdatedDate:  apiIssue.UpdatedAt,
                ClosedDate:   apiIssue.CompletedAt,
            }
            // Map first assignee (multi-assignee not supported in v1)
            if len(apiIssue.Assignees) > 0 {
                issue.AssigneeId = apiIssue.Assignees[0].Id
                issue.AssigneeName = apiIssue.Assignees[0].DisplayName
            }

            return []interface{}{issue}, nil
        },
    })
    if err != nil {
        return err
    }
    return extractor.Execute()
}
```

### Step 9 — Converter (Tool Layer → Domain)

```go
// tasks/issue_convertor.go
var ConvertIssuesMeta = plugin.SubTaskMeta{
    Name:             "convertIssues",
    EntryPoint:       ConvertIssues,
    EnabledByDefault: true,
    Description:      "convert MyPlugin issues to DevLake domain model",
    DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func ConvertIssues(subtaskCtx plugin.SubTaskContext) errors.Error {
    data := subtaskCtx.GetData().(*MyTaskData)
    db := subtaskCtx.GetDal()

    issueIdGen := didgen.NewDomainIdGenerator(&models.MyIssue{})
    boardIdGen := didgen.NewDomainIdGenerator(&models.MyProject{})
    boardId := boardIdGen.Generate(data.Options.ConnectionId, data.Options.ProjectId)

    converter, err := api.NewStatefulDataConverter(&api.StatefulDataConverterArgs[models.MyIssue]{
        SubtaskCommonArgs: &api.SubtaskCommonArgs{
            SubTaskContext: subtaskCtx,
            Table:          RAW_ISSUE_TABLE,
            Params: MyApiParams{
                ConnectionId: data.Options.ConnectionId,
                ProjectId:    data.Options.ProjectId,
            },
        },
        Input: func(stateManager *api.SubtaskStateManager) (dal.Rows, errors.Error) {
            clauses := []dal.Clause{
                dal.Select("*"),
                dal.From(&models.MyIssue{}),
                // CRITICAL: Filter by BOTH connection_id AND project_id.
                // Filtering only by connection_id causes cross-project data leakage
                // when multiple projects exist under one connection.
                dal.Where("connection_id = ? AND project_id = ?",
                    data.Options.ConnectionId, data.Options.ProjectId),
            }
            if stateManager.IsIncremental() {
                if since := stateManager.GetSince(); since != nil {
                    clauses = append(clauses, dal.Where("updated_at >= ?", since))
                }
            }
            return db.Cursor(clauses...)
        },
        Convert: func(issue *models.MyIssue) ([]interface{}, errors.Error) {
            domainIssue := &ticket.Issue{
                DomainEntity: domainlayer.DomainEntity{
                    Id: issueIdGen.Generate(issue.ConnectionId, issue.IssueId),
                },
                IssueKey:       issue.IssueId,
                Title:          issue.Title,
                Type:           mapIssueType(issue.IssueType),
                OriginalType:   issue.IssueType,
                Status:         mapIssueStatus(issue.Status),
                OriginalStatus: issue.Status,
                Priority:       issue.Priority,
                AssigneeId:     issue.AssigneeId,
                AssigneeName:   issue.AssigneeName,
                CreatedDate:    issue.CreatedDate,
                UpdatedDate:    issue.UpdatedDate,
                ResolutionDate: issue.ClosedDate,
            }

            // Lead time: only when actually closed
            if issue.ClosedDate != nil && issue.CreatedDate != nil {
                mins := uint(issue.ClosedDate.Sub(*issue.CreatedDate).Minutes())
                if mins > 0 {
                    domainIssue.LeadTimeMinutes = &mins
                }
            }

            boardIssue := &ticket.BoardIssue{
                BoardId: boardId,
                IssueId: domainIssue.Id,
            }
            return []interface{}{domainIssue, boardIssue}, nil
        },
    })
    if err != nil {
        return err
    }
    return converter.Execute()
}

// mapIssueStatus normalises source status to DevLake standard values.
// DevLake recognises: TODO, IN_PROGRESS, DONE, OTHER
func mapIssueStatus(status string) string {
    switch strings.ToLower(status) {
    case "done", "closed", "completed", "resolved":
        return "DONE"
    case "in progress", "in_progress", "started":
        return "IN_PROGRESS"
    default:
        return "TODO"
    }
}

// mapIssueType normalises source type to DevLake standard values.
// DevLake recognises: BUG, REQUIREMENT, INCIDENT, QUESTION, EPIC, USER_STORY, TASK
func mapIssueType(issueType string) string {
    switch strings.ToLower(issueType) {
    case "bug", "defect":
        return "BUG"
    case "epic":
        return "EPIC"
    case "story", "user story":
        return "USER_STORY"
    case "task":
        return "TASK"
    default:
        return "REQUIREMENT"
    }
}
```

### Step 10 — PrepareTaskData (impl/impl.go)

```go
func (p MyPlugin) PrepareTaskData(taskCtx plugin.TaskContext, options map[string]interface{}) (interface{}, errors.Error) {
    var op tasks.MyOptions
    if err := helper.Decode(options, &op, nil); err != nil {
        return nil, errors.Default.Wrap(err, "could not decode options")
    }
    if op.ConnectionId == 0 {
        return nil, errors.BadInput.New("connectionId is required")
    }

    connection := &models.MyConnection{}
    connectionHelper := helper.NewConnectionHelper(taskCtx, nil, p.Name())
    if err := connectionHelper.FirstById(connection, op.ConnectionId); err != nil {
        return nil, errors.Default.Wrap(err, "failed to load connection")
    }

    apiClient, err := tasks.NewMyApiClient(taskCtx, connection)
    if err != nil {
        return nil, errors.Default.Wrap(err, "failed to create API client")
    }

    // Load scope if provided
    if op.ProjectId != "" {
        var scope models.MyProject
        db := taskCtx.GetDal()
        err = db.First(&scope, dal.Where("connection_id = ? AND project_id = ?", op.ConnectionId, op.ProjectId))
        if err != nil && db.IsErrorNotFound(err) {
            // Best practice: fetch and save missing scope from remote rather than erroring.
            // Taiga left a TODO here — avoid that gap.
            return nil, errors.Default.Wrap(err, fmt.Sprintf("project %s not found; import it first", op.ProjectId))
        }
        if err != nil {
            return nil, errors.Default.Wrap(err, "failed to load project")
        }
        if op.ScopeConfigId == 0 && scope.ScopeConfigId != 0 {
            op.ScopeConfigId = scope.ScopeConfigId
        }
    }

    // Load scope config
    if op.ScopeConfig == nil && op.ScopeConfigId != 0 {
        var sc models.MyScopeConfig
        if err := taskCtx.GetDal().First(&sc, dal.Where("id = ?", op.ScopeConfigId)); err != nil {
            return nil, errors.BadInput.Wrap(err, "failed to load scope config")
        }
        op.ScopeConfig = &sc
    }
    if op.ScopeConfig == nil {
        op.ScopeConfig = new(models.MyScopeConfig)
    }

    if op.PageSize <= 0 || op.PageSize > 100 {
        op.PageSize = 100
    }

    return &tasks.MyTaskData{Options: &op, ApiClient: apiClient}, nil
}
```

### Step 11 — Migration Scripts

```go
// models/migrationscripts/20260101000001_add_init_tables.go
type addInitTables20260101 struct{}

func (m *addInitTables20260101) Up(basicRes context.BasicRes) errors.Error {
    db := basicRes.GetDal()
    return db.AutoMigrate(
        &models.MyConnection{},
        &models.MyProject{},
        &models.MyScopeConfig{},
        &models.MyIssue{},
    )
}

func (m *addInitTables20260101) Version() uint64 { return 20260101000001 }
func (m *addInitTables20260101) Name() string    { return "myplugin init tables" }

// models/migrationscripts/register.go
func All() []plugin.MigrationScript {
    return []plugin.MigrationScript{
        new(addInitTables20260101),
        // Add new migrations here chronologically
    }
}
```

**Migration rules:**
- Version is a `uint64` timestamp: `YYYYMMDDHHMMSS`
- Never modify an existing migration — only add new ones
- Each migration is additive; never drop columns or tables in migrations
- Register all migrations in `All()` in `register.go`

### Step 12 — API Layer Init

```go
// api/init.go
var dsHelper *api.DsHelper[models.MyConnection, models.MyProject, models.MyScopeConfig]
var raProxy *api.DsRemoteApiProxyHelper[models.MyConnection]
var raScopeList *api.DsRemoteApiScopeListHelper[models.MyConnection, models.MyProject, MyRemotePagination]

func Init(br context.BasicRes, p plugin.PluginMeta) {
    basicRes = br
    vld = validator.New()
    dsHelper = api.NewDataSourceHelper[
        models.MyConnection,
        models.MyProject,
        models.MyScopeConfig,
    ](
        br,
        p.Name(),
        []string{"name"},                            // scope search fields
        func(c models.MyConnection) models.MyConnection { return c.Sanitize() },
        nil,
        nil,
    )
    raProxy = api.NewDsRemoteApiProxyHelper[models.MyConnection](dsHelper.ConnApi.ModelApiHelper)
    raScopeList = api.NewDsRemoteApiScopeListHelper[models.MyConnection, models.MyProject, MyRemotePagination](raProxy, listMyRemoteScopes)
}
```

### Step 13 — Connection API Handlers

```go
// api/connection_api.go — all delegated to dsHelper
var TestConnection      = dsHelper.ConnApi.TestConnection
var PostConnections     = dsHelper.ConnApi.PostConnections
var ListConnections     = dsHelper.ConnApi.ListConnections
var GetConnection       = dsHelper.ConnApi.GetConnection
var PatchConnection     = dsHelper.ConnApi.PatchConnection
var DeleteConnection    = dsHelper.ConnApi.DeleteConnection
var TestExistingConnection = dsHelper.ConnApi.TestExistingConnection
```

For `TestConnection`, the helper calls `SetupAuthentication` and makes a test request. Ensure the connection model implements `ApiAuthenticator`.

### Step 14 — ApiResources Map

```go
func (p MyPlugin) ApiResources() map[string]map[string]plugin.ApiResourceHandler {
    return map[string]map[string]plugin.ApiResourceHandler{
        "test":                                            {"POST": api.TestConnection},
        "connections":                                     {"POST": api.PostConnections, "GET": api.ListConnections},
        "connections/:connectionId":                       {"GET": api.GetConnection, "PATCH": api.PatchConnection, "DELETE": api.DeleteConnection},
        "connections/:connectionId/test":                  {"POST": api.TestExistingConnection},
        "connections/:connectionId/remote-scopes":         {"GET": api.RemoteScopes},
        "connections/:connectionId/scopes":                {"GET": api.GetScopeList, "PUT": api.PutScope},
        "connections/:connectionId/scopes/:scopeId":       {"GET": api.GetScope, "PATCH": api.UpdateScope, "DELETE": api.DeleteScope},
        "connections/:connectionId/scope-configs":         {"POST": api.CreateScopeConfig, "GET": api.GetScopeConfigList},
        "connections/:connectionId/scope-configs/:scopeConfigId": {
            "PATCH": api.UpdateScopeConfig, "GET": api.GetScopeConfig, "DELETE": api.DeleteScopeConfig,
        },
        // Optional: reverse lookup
        "scope-config/:scopeConfigId/projects":            {"GET": api.GetProjectsByScopeConfig},
    }
}
```

### Step 15 — Blueprint V200

```go
// api/blueprint_v200.go
func MakeDataSourcePipelinePlanV200(
    subtaskMetas []plugin.SubTaskMeta,
    connectionId uint64,
    bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
    connection, err := dsHelper.ConnSrv.FindByPk(connectionId)
    if err != nil {
        return nil, nil, err
    }
    scopeDetails, err := dsHelper.ScopeSrv.MapScopeDetails(connectionId, bpScopes)
    if err != nil {
        return nil, nil, err
    }
    _, err = helper.NewApiClientFromConnection(context.TODO(), basicRes, connection)
    if err != nil {
        return nil, nil, err
    }

    plan := make(coreModels.PipelinePlan, len(scopeDetails))
    scopes := make([]plugin.Scope, 0)
    idGen := didgen.NewDomainIdGenerator(&models.MyProject{})

    for i, sd := range scopeDetails {
        scope, scopeConfig := sd.Scope, sd.ScopeConfig
        task, err := helper.MakePipelinePlanTask(
            "myplugin",
            subtaskMetas,
            scopeConfig.Entities,
            MyTaskOptions{ConnectionId: scope.ConnectionId, ProjectId: scope.ProjectId},
        )
        if err != nil {
            return nil, nil, err
        }
        plan[i] = coreModels.PipelineStage{task}

        // Add domain-layer board if TICKET domain requested
        for _, entity := range scopeConfig.Entities {
            if entity == plugin.DOMAIN_TYPE_TICKET {
                scopes = append(scopes, &ticket.Board{
                    DomainEntity: domainlayer.DomainEntity{
                        Id: idGen.Generate(connection.ID, scope.ProjectId),
                    },
                    Name: scope.Name,
                })
                break
            }
        }
    }

    return plan, scopes, nil
}
```

---

## Domain ID Generation

Domain IDs must be deterministic and stable across syncs.

```go
import "github.com/apache/incubator-devlake/core/models/domainlayer/didgen"

issueIdGen := didgen.NewDomainIdGenerator(&models.MyIssue{})
id := issueIdGen.Generate(connectionId, issueId)
// Result: "myplugin:MyIssue:1:abc-123"

boardIdGen := didgen.NewDomainIdGenerator(&models.MyProject{})
boardId := boardIdGen.Generate(connectionId, projectId)
// Result: "myplugin:MyProject:1:proj-456"
```

**Rules:**
- Always pass `connectionId` as the first argument
- Use the native source ID (string or numeric) as the second argument
- The generator derives the plugin name from the type's package path
- IDs are stable — same inputs always produce the same output

---

## SubTaskMeta Best Practices

```go
var CollectIssuesMeta = plugin.SubTaskMeta{
    Name:             "collectIssues",          // camelCase, unique within plugin
    EntryPoint:       CollectIssues,            // matches var _ plugin.SubTaskEntryPoint = CollectIssues
    EnabledByDefault: true,
    Description:      "collect issues from remote API",
    DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
    // Optional — set if this subtask reads output of another:
    Dependencies:     []*plugin.SubTaskMeta{&ExtractIssuesMeta},
    DependencyTables: []string{RAW_ISSUE_TABLE},
    ProductTables:    []string{"_tool_myplugin_issues"},
}
```

Register all metas in `SubTaskMetas()` in `impl.go`. Order matters — collectors before extractors before converters.

---

## Known Pitfalls (Lessons from Taiga plugin)

### 1. Cross-project data leakage — CRITICAL

```go
// WRONG: Only filters connection_id — leaks rows from other projects
dal.Where("connection_id = ?", data.Options.ConnectionId)

// CORRECT: Always filter BOTH
dal.Where("connection_id = ? AND project_id = ?",
    data.Options.ConnectionId, data.Options.ProjectId)
```

### 2. Weak pagination

```go
// WRONG: Large page size is not pagination
PageSize: 1000 // Will break for large projects

// CORRECT: Implement GetTotalPages + proper Query
GetTotalPages: func(res *http.Response, args *api.ApiCollectorArgs) (int, errors.Error) { ... }
```

### 3. Scope config mappings must be implemented, not stubbed

If you define `TypeMappings` on `ScopeConfig`, you must apply them in converters. Unused config fields mislead users.

### 4. Partial field mapping

Map every extracted field to the domain model. Fields extracted but not converted are silently lost and will confuse users.

### 5. Inconsistent status normalisation

All entity types in the same plugin should normalise status using the same helper function. Mixing closed/open for some types and raw status for others breaks dashboard queries.

### 6. Missing secret preservation in PATCH

If `MergeFromRequest` is not implemented on the connection model, a PATCH request that omits the token/password will clear the stored credential.

### 7. Table naming

All `_tool_*` tables must match `TableName()` on the model struct. Mismatches cause silent runtime errors.

---

## Testing Strategy

### Unit Tests (extractors and converters)

Test that JSON fixtures produce exactly the expected tool-layer and domain-layer rows.

```go
func TestExtractIssues(t *testing.T) {
    rawData := `{"id":"abc","title":"Fix crash","state":{"name":"In Progress"}}`
    rows, err := extractIssue([]byte(rawData), 1, "proj-1")
    require.NoError(t, err)
    require.Len(t, rows, 1)
    issue := rows[0].(*models.MyIssue)
    assert.Equal(t, "Fix crash", issue.Title)
    assert.Equal(t, "In Progress", issue.Status)
}
```

### E2E Snapshot Tests (full pipeline)

Located in `{plugin}/e2e/`. These are the most valuable tests:

1. Import CSV fixture into `_raw_*` table
2. Run extractor subtask
3. Compare `_tool_*` against golden snapshot
4. Run converter subtask
5. Compare domain tables against golden snapshot

```go
func TestIssues(t *testing.T) {
    var testIssue e2ehelper.DataFlowTester
    testIssue.ImportCsvIntoRawTable("./snapshot_tables/_raw_myplugin_api_issues.csv",
        "_raw_myplugin_api_issues")
    testIssue.Subtask(tasks.ExtractIssuesMeta, taskData)
    testIssue.VerifyTableWithOptions(models.MyIssue{}, e2ehelper.TableOptions{
        CSVRelPath: "./snapshot_tables/_tool_myplugin_issues.csv",
    })
    testIssue.Subtask(tasks.ConvertIssuesMeta, taskData)
    testIssue.VerifyTableWithOptions(ticket.Issue{}, e2ehelper.TableOptions{
        CSVRelPath: "./snapshot_tables/issues.csv",
    })
}
```

Write snapshot E2E tests for every entity before shipping.

---

## Checklist Before Submitting a Plugin PR

- [ ] All plugin interfaces implemented with compile-time assertion in `impl.go`
- [ ] `RootPkgPath()` matches actual Go module path
- [ ] `TableName()` defined on every model
- [ ] Every tool-layer model has `NoPKModel` or `Model` embedded (for timestamps + raw data origin)
- [ ] Connection sanitises secrets before returning (API/GET never leaks credentials)
- [ ] `MergeFromRequest` preserves existing secrets on PATCH
- [ ] Collectors implement real pagination (not just large page sizes)
- [ ] Converters filter on both `connection_id` AND `project_id` (or equivalent scope ID)
- [ ] Status normalisation is consistent across all entity types
- [ ] All extracted fields are mapped to domain models (no silent data loss)
- [ ] Domain IDs generated via `didgen` (not hand-crafted strings)
- [ ] Migration scripts added for every new table; `All()` updated
- [ ] `GetTablesInfo()` lists every model
- [ ] `SubTaskMetas()` lists every subtask in collect → extract → convert order
- [ ] E2E snapshot tests written for every entity
- [ ] `Close()` releases the async API client
- [ ] Apache license header in every `.go` file

---

## Domain Layer Quick Reference

| Domain | Type | Key Fields |
|---|---|---|
| `ticket.Board` | Board/project | `Id`, `Name`, `Description`, `Url` |
| `ticket.Issue` | Work item | `Id`, `IssueKey`, `Title`, `Type`, `Status`, `Priority`, `AssigneeId`, `CreatedDate`, `UpdatedDate`, `ResolutionDate`, `LeadTimeMinutes`, `StoryPoint` |
| `ticket.BoardIssue` | Board-issue link | `BoardId`, `IssueId` |
| `ticket.Sprint` | Sprint/iteration | `Id`, `Name`, `StartedDate`, `CompletedDate`, `State` |
| `code.Repo` | Repository | `Id`, `Name`, `HttpUrlToRepo`, `CreatedDate` |
| `code.PullRequest` | PR/MR | `Id`, `Title`, `Status`, `MergedDate`, `AuthorId` |
| `code.Commit` | Commit | `Sha`, `Message`, `AuthorName`, `AuthoredDate` |
| `devops.CicdPipeline` | Pipeline run | `Id`, `Name`, `Result`, `Status`, `StartedDate`, `FinishedDate` |

Import paths:
```go
"github.com/apache/incubator-devlake/core/models/domainlayer"        // DomainEntity
"github.com/apache/incubator-devlake/core/models/domainlayer/ticket"
"github.com/apache/incubator-devlake/core/models/domainlayer/code"
"github.com/apache/incubator-devlake/core/models/domainlayer/devops"
"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
```

---

## Apache License Header

Every `.go` file must start with:

```go
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
```
