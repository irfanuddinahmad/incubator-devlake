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

	"github.com/apache/incubator-devlake/core/models/common"
)

// ClaudeUsage stores daily per-user Claude Code usage metrics ingested from the
// Anthropic Admin API (/v1/organizations/usage_report/claude_code).
type ClaudeUsage struct {
	common.NoPKModel

	// Primary key fields
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`
	Model        string    `gorm:"primaryKey;type:varchar(100)" json:"model"`

	// Core productivity metrics
	NumSessions     int `json:"numSessions"`
	LinesAdded      int `json:"linesAdded"`
	LinesRemoved    int `json:"linesRemoved"`
	CommitsByClaude int `json:"commitsByClaude"`
	PrsByClaude     int `json:"prsByClaude"`

	// Tool action counts — totals for the day (same value across all model rows
	// for the same user/date since tool_actions is not per-model in the API).
	EditToolAccepted         int `json:"editToolAccepted"`
	EditToolRejected         int `json:"editToolRejected"`
	MultiEditToolAccepted    int `json:"multiEditToolAccepted"`
	MultiEditToolRejected    int `json:"multiEditToolRejected"`
	WriteToolAccepted        int `json:"writeToolAccepted"`
	WriteToolRejected        int `json:"writeToolRejected"`
	NotebookEditToolAccepted int `json:"notebookEditToolAccepted"`
	NotebookEditToolRejected int `json:"notebookEditToolRejected"`

	// Token usage & cost (per model, from model_breakdown)
	InputTokens         int64   `json:"inputTokens"`
	OutputTokens        int64   `json:"outputTokens"`
	CacheReadTokens     int64   `json:"cacheReadTokens"`
	CacheCreationTokens int64   `json:"cacheCreationTokens"`
	EstimatedCostUsd    float64 `json:"estimatedCostUsd"`
}

func (ClaudeUsage) TableName() string {
	return "_tool_claude_usage"
}
