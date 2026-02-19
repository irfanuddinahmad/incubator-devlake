## Overview
This PR adds comprehensive developer telemetry support to DevLake with git activity tracking and development pattern analysis. This plugin enables automatic collection of meaningful developer metrics including code contributions, repository activity, and development workflows.

## Key Features

### Git Activity Metrics
- **Commit Tracking**: Total commits per developer per day
- **Code Churn**: Lines added, deleted, and files changed
- **Repository Breakdown**: Per-repository statistics with branch information
- **Accurate Time Attribution**: Metrics properly timestamped to activity date

### Development Activity Metrics
- **Test Runs**: Automatic detection of test execution patterns
- **Build Commands**: Tracking of build and compilation activities
- **Pattern-Based Analysis**: Intelligent recognition of development workflows

### Database Schema
- New table: `_tool_developer_metrics`
- JSON fields for flexible metric storage
- Proper indexing on developer_id and date
- Migration script: `20260219_add_git_activity_fields.go`

## API Endpoints

### Report Submission
```
POST /plugins/developer_telemetry/report
POST /plugins/developer_telemetry/connections/:connectionId/report
```

### Connection Management
- GET, POST, PATCH, DELETE for connections
- Webhook-style configuration
- API key authentication support

## Critical Bug Fix: Mapstructure Serialization

**Problem**: Nested struct fields (GitActivity, Repository, DevelopmentActivity) were not being properly deserialized from JSON. The `api.Decode` function uses the mapstructure library which requires explicit struct tags.

**Symptoms**: 
- Commit counts stored correctly
- Line counts and other nested fields defaulted to 0
- Arrays like `branches_worked` were empty

**Solution**: Added `mapstructure` tags to ALL nested struct fields alongside existing `json` tags.

**Before**:
```go
type GitActivity struct {
    TotalCommits int `json:"total_commits"`
    TotalLinesAdded int `json:"total_lines_added"`
    // ... missing mapstructure tags
}
```

**After**:
```go
type GitActivity struct {
    TotalCommits int `json:"total_commits" mapstructure:"total_commits"`
    TotalLinesAdded int `json:"total_lines_added" mapstructure:"total_lines_added"`
    // ... all fields now have mapstructure tags
}
```

**Impact**: Complete data now properly stored in database with all metrics accurate.

## UI Integration
- New plugin registration in config-ui
- Connection dialogs for configuration
- View/Edit/Delete operations
- Developer-telemetry icon and branding

## Testing

### End-to-End Validation
- ✅ Built plugin successfully
- ✅ Created database connection (ID 17)
- ✅ Received telemetry data via API
- ✅ Verified complete JSON deserialization
- ✅ Confirmed all metrics stored accurately

### Example Verified Data
```json
{
  "git_activity": {
    "total_commits": 4,
    "total_lines_added": 390,
    "total_lines_deleted": 157,
    "total_files_changed": 7,
    "repositories": [
      {
        "name": "incubator-devlake",
        "commits": 1,
        "lines_added": 145,
        "lines_deleted": 28,
        "files_changed": 4,
        "branches_worked": ["feature/enhanced-telemetry-backend"]
      },
      {
        "name": "mosyle-dev-telemetry",
        "commits": 3,
        "lines_added": 245,
        "lines_deleted": 129,
        "files_changed": 3,
        "branches_worked": ["feature/enhanced-metrics-git-activity"]
      }
    ]
  },
  "development_activity": {
    "test_runs_detected": 5,
    "build_commands_detected": 3
  }
}
```

## Files Changed

### Backend Plugin
- `plugins/developer_telemetry/api/report_api.go`: API handlers with mapstructure tags
- `plugins/developer_telemetry/models/developer_metrics.go`: Database models
- `plugins/developer_telemetry/models/migrationscripts/20260219_add_git_activity_fields.go`: Migration
- `plugins/developer_telemetry/impl/impl.go`: Plugin registration

### Config UI
- `config-ui/src/plugins/register/developer-telemetry/`: Complete UI component suite
  - Connection creation/editing dialogs
  - View and delete operations
  - Configuration utilities
  - Icon and styling

## Migration Path
1. Database migration runs automatically on plugin load
2. Creates `_tool_developer_metrics` table if not exists
3. Backward compatible - existing connections unaffected
4. No data migration needed (new plugin)

## Deployment Notes
- **Plugin Build**: `make build-plugin plugin=developer_telemetry`
- **Database**: MySQL 8+ required
- **API Authentication**: Supports API key authentication
- **Data Retention**: Configurable per connection
- **Performance**: Optimized JSON storage with proper indexing

## Related PR
- Collector: Enhanced Developer Telemetry with Git Activity & Pattern-Based Analysis

## Commits
- feat: Add git activity and development activity metrics to developer_telemetry plugin
- fix: Add mapstructure tags to nested structs for proper JSON decoding
