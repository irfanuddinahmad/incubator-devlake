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

// Package ai holds domain-layer models for AI coding assistant activity.
package ai

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/domainlayer"
)

// AiActivity records a single day of AI-assisted coding activity for one developer.
// It is the normalised domain-layer representation shared by all AI tool plugins
// (Claude, GitHub Copilot, Cursor, Codex, etc.).
type AiActivity struct {
	domainlayer.DomainEntity

	// Provider identifies the AI tool (e.g. "claude", "gh-copilot", "cursor", "codex").
	Provider string `gorm:"type:varchar(100);index" json:"provider"`

	// AccountId is the global DevLake account resolved from UserEmail.
	// May be empty if no matching account was found.
	AccountId string `gorm:"type:varchar(255);index" json:"accountId"`

	// UserEmail is the raw email (or login) as returned by the provider.
	UserEmail string `gorm:"type:varchar(255)" json:"userEmail"`

	// Date is the calendar day the activity was recorded.
	Date time.Time `gorm:"type:date;index" json:"date"`

	// Type categorises the activity (e.g. "CODE_EDIT", "CHAT").
	Type string `gorm:"type:varchar(100)" json:"type"`

	// Model is the AI model variant used (e.g. "claude-sonnet-4-5", "gpt-4o").
	Model string `gorm:"type:varchar(100)" json:"model"`

	// InterfaceType describes how the developer interacts with the tool.
	// Known values: "cli", "ide_plugin", "web_ui".
	InterfaceType string `gorm:"type:varchar(50)" json:"interfaceType"`

	// Volume & autocomplete metrics (Copilot / Cursor style)
	NumSessions      int `json:"numSessions"`
	SuggestionsCount int `json:"suggestionsCount"` // suggestions shown to the developer
	AcceptanceCount  int `json:"acceptanceCount"`  // suggestions accepted by the developer

	// Code change metrics
	LinesAdded   int `json:"linesAdded"`
	LinesRemoved int `json:"linesRemoved"`

	// Agentic outcome metrics (Claude / Codex style)
	CommitsCreated int `json:"commitsCreated"`
	PrsCreated     int `json:"prsCreated"`

	// Cost & token metrics
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	EstimatedCostUsd float64 `json:"estimatedCostUsd"`
}

func (AiActivity) TableName() string {
	return "ai_activities"
}
