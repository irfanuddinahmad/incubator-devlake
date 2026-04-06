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

// CursorDailyUsage stores per-user daily usage metrics from POST /teams/daily-usage-data.
type CursorDailyUsage struct {
	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Day          time.Time `gorm:"primaryKey;type:date" json:"day"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`

	// Tab completion metrics
	TotalTabsShown    int `json:"totalTabsShown"`
	TotalTabsAccepted int `json:"totalTabsAccepted"`

	// Code change metrics
	TotalLinesAdded      int `json:"totalLinesAdded"`
	TotalLinesDeleted    int `json:"totalLinesDeleted"`
	AcceptedLinesAdded   int `json:"acceptedLinesAdded"`
	AcceptedLinesDeleted int `json:"acceptedLinesDeleted"`

	// Apply/accept actions
	TotalApplies int `json:"totalApplies"`
	TotalAccepts int `json:"totalAccepts"`
	TotalRejects int `json:"totalRejects"`

	// AI request type breakdown
	ComposerRequests int `json:"composerRequests"`
	ChatRequests     int `json:"chatRequests"`
	AgentRequests    int `json:"agentRequests"`
	CmdkUsages       int `json:"cmdkUsages"`

	// Billing request type breakdown
	SubscriptionIncludedReqs int `json:"subscriptionIncludedReqs"`
	ApiKeyReqs               int `json:"apiKeyReqs"`
	UsageBasedReqs           int `json:"usageBasedReqs"`

	// Misc
	BugbotUsages  int    `json:"bugbotUsages"`
	MostUsedModel string `gorm:"type:varchar(100)" json:"mostUsedModel"`
	ClientVersion string `gorm:"type:varchar(50)" json:"clientVersion"`

	common.NoPKModel
}

func (CursorDailyUsage) TableName() string {
	return "_tool_cursor_daily_usage"
}

// CursorUsageEvent stores individual AI request events from POST /teams/filtered-usage-events.
// Each record represents one API call made by a team member.
type CursorUsageEvent struct {
	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	// Timestamp in milliseconds treated as primary key granularity (ms precision).
	Timestamp time.Time `gorm:"primaryKey;type:datetime(3)" json:"timestamp"`
	UserEmail string    `gorm:"primaryKey;type:varchar(255)" json:"userEmail"`
	Model     string    `gorm:"primaryKey;type:varchar(100)" json:"model"`

	// Billing category: "Usage-based", "Included in Business", etc.
	Kind string `gorm:"type:varchar(100)" json:"kind"`

	MaxMode          bool    `json:"maxMode"`
	RequestsCosts    float64 `json:"requestsCosts"`
	IsTokenBasedCall bool    `json:"isTokenBasedCall"`
	IsChargeable     bool    `json:"isChargeable"`
	IsHeadless       bool    `json:"isHeadless"`

	// Token usage (populated when IsTokenBasedCall is true)
	InputTokens      int64   `json:"inputTokens"`
	OutputTokens     int64   `json:"outputTokens"`
	CacheWriteTokens int64   `json:"cacheWriteTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens"`
	TotalCents       float64 `json:"totalCents"`

	// Total amount charged in cents (model cost + Cursor Token Fee if applicable).
	ChargedCents float64 `json:"chargedCents"`

	common.NoPKModel
}

func (CursorUsageEvent) TableName() string {
	return "_tool_cursor_usage_events"
}

// CursorCommitAiShare stores per-commit AI code attribution from GET /analytics/ai-code/commits.
// This endpoint is only available on Enterprise plans.
type CursorCommitAiShare struct {
	ConnectionId uint64 `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	RepoName     string `gorm:"primaryKey;type:varchar(255)" json:"repoName"`
	CommitSha    string `gorm:"primaryKey;type:varchar(64)" json:"commitSha"`

	CommitDate         time.Time `gorm:"type:date" json:"commitDate"`
	TabLinesAdded      int       `json:"tabLinesAdded"`
	ComposerLinesAdded int       `json:"composerLinesAdded"`
	ManualLinesAdded   int       `json:"manualLinesAdded"`

	common.NoPKModel
}

func (CursorCommitAiShare) TableName() string {
	return "_tool_cursor_commit_ai_share"
}
