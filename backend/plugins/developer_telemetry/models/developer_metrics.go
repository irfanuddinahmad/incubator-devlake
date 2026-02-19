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
	"time"
)

// GitActivity represents git activity metrics for a developer
type GitActivity struct {
	TotalCommits      int          `json:"total_commits"`
	TotalLinesAdded   int          `json:"total_lines_added"`
	TotalLinesDeleted int          `json:"total_lines_deleted"`
	TotalFilesChanged int          `json:"total_files_changed"`
	Repositories      []Repository `json:"repositories"`
}

// Repository represents per-repository git metrics
type Repository struct {
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Commits        int      `json:"commits"`
	LinesAdded     int      `json:"lines_added"`
	LinesDeleted   int      `json:"lines_deleted"`
	FilesChanged   int      `json:"files_changed"`
	BranchesWorked []string `json:"branches_worked"`
}

// DevelopmentActivity represents detected development activity patterns
type DevelopmentActivity struct {
	TestRunsDetected int `json:"test_runs_detected"`
	BuildsDetected   int `json:"build_commands_detected"`
}

// DeveloperMetrics represents the tool layer table for developer telemetry data
type DeveloperMetrics struct {
	ConnectionId        uint64    `gorm:"primaryKey;type:BIGINT;column:connection_id" json:"connection_id"`
	DeveloperId         string    `gorm:"primaryKey;type:varchar(255);column:developer_id" json:"developer_id"`
	Date                time.Time `gorm:"primaryKey;type:date;column:date" json:"date"`
	Email               string    `gorm:"type:varchar(255);index;column:email" json:"email"`
	Name                string    `gorm:"type:varchar(255);column:name" json:"name"`
	Hostname            string    `gorm:"type:varchar(255);column:hostname" json:"hostname"`
	ActiveHours         int       `gorm:"column:active_hours" json:"active_hours"`
	ToolsUsed           string    `gorm:"type:text;column:tools_used" json:"tools_used"`                     // JSON array stored as text
	ProjectContext      string    `gorm:"type:text;column:project_context" json:"project_context"`           // JSON array stored as text
	GitActivity         string    `gorm:"type:text;column:git_activity" json:"git_activity"`                 // JSON object stored as text
	DevelopmentActivity string    `gorm:"type:text;column:development_activity" json:"development_activity"` // JSON object stored as text
	OsInfo              string    `gorm:"type:varchar(255);column:os_info" json:"os_info"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (DeveloperMetrics) TableName() string {
	return "_tool_developer_metrics"
}
