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
// Anthropic Admin API (/v1/organizations/{org_id}/usage_report/claude_code).
type ClaudeUsage struct {
	common.NoPKModel

	// Primary key fields
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`

	// Core productivity metrics
	NumSessions     int `json:"numSessions"`
	LinesAdded      int `json:"linesAdded"`
	LinesRemoved    int `json:"linesRemoved"`
	CommitsByClaude int `json:"commitsByClaude"`
	PrsByClaude     int `json:"prsByClaude"`

	// Model & cost breakdown
	Model            string  `gorm:"type:varchar(100)" json:"model"`
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	EstimatedCostUsd float64 `json:"estimatedCostUsd"`
}

func (ClaudeUsage) TableName() string {
	return "_tool_claude_usage"
}
