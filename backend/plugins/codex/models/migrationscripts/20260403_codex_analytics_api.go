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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/migrationscripts/archived"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

// migrateToAnalyticsAPI migrates the Codex plugin to the Codex Enterprise
// Analytics API (https://api.chatgpt.com/v1/analytics/codex).
//
// Changes:
//   - _tool_codex_connections: adds workspace_id column for the ChatGPT Enterprise
//     workspace ID required by the Analytics API path parameter.
//   - _tool_codex_usage: drops the old completions-API schema (input_tokens,
//     output_tokens, model) and replaces it with Analytics-API fields (threads,
//     turns, credits, client_surface, user_email).
//   - _tool_codex_code_reviews: new table for per-PR code-review activity.
//   - _tool_codex_code_review_responses: new table for per-user engagement data.
type migrateToAnalyticsAPI struct{}

// codexConnection20260403 adds workspace_id to the connections table.
type codexConnection20260403 struct {
	archived.Model
	Name             string `gorm:"type:varchar(100);uniqueIndex"`
	Endpoint         string `gorm:"type:varchar(255)"`
	Proxy            string `gorm:"type:varchar(255)"`
	RateLimitPerHour int
	ApiKey           string
	// WorkspaceId is the ChatGPT Enterprise workspace ID (new field).
	WorkspaceId string `gorm:"type:varchar(255)"`
}

func (codexConnection20260403) TableName() string { return "_tool_codex_connections" }

// codexUsage20260403 replaces the old completions-API schema.
// The old PK was (connection_id, scope_id, date, model).
// The new PK is  (connection_id, scope_id, date, client_surface, user_email).
type codexUsage20260403 struct {
	archived.NoPKModel
	ConnectionId  uint64    `gorm:"primaryKey"`
	ScopeId       string    `gorm:"primaryKey;type:varchar(255)"`
	Date          time.Time `gorm:"primaryKey;type:date"`
	ClientSurface string    `gorm:"primaryKey;type:varchar(50);default:''"`
	UserEmail     string    `gorm:"primaryKey;type:varchar(255);default:''"`
	Threads       int64
	Turns         int64
	Credits       float64
}

func (codexUsage20260403) TableName() string { return "_tool_codex_usage" }

type codexCodeReview20260403 struct {
	archived.NoPKModel
	ConnectionId      uint64    `gorm:"primaryKey"`
	ScopeId           string    `gorm:"primaryKey;type:varchar(255)"`
	Date              time.Time `gorm:"primaryKey;type:date"`
	PrUrl             string    `gorm:"primaryKey;type:varchar(500);default:''"`
	ReviewsCompleted  int64
	CommentsGenerated int64
	SeverityLow       int64
	SeverityMedium    int64
	SeverityHigh      int64
	SeverityCritical  int64
}

func (codexCodeReview20260403) TableName() string { return "_tool_codex_code_reviews" }

type codexCodeReviewResponse20260403 struct {
	archived.NoPKModel
	ConnectionId uint64    `gorm:"primaryKey"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)"`
	Date         time.Time `gorm:"primaryKey;type:date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255);default:''"`
	Replies      int64
	Upvotes      int64
	Downvotes    int64
}

func (codexCodeReviewResponse20260403) TableName() string {
	return "_tool_codex_code_review_responses"
}

func (script *migrateToAnalyticsAPI) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Drop the old usage table so it can be recreated with the new Analytics API schema.
	// The old schema (input_tokens / output_tokens / model) is incompatible with the new
	// Analytics API schema (threads / turns / credits / client_surface / user_email).
	if err := db.DropTables(&codexUsage20260403{}); err != nil {
		return err
	}

	// AutoMigrate the updated/new tables.
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&codexConnection20260403{},         // adds workspace_id column
		&codexUsage20260403{},              // recreates usage table with new schema
		&codexCodeReview20260403{},         // new table
		&codexCodeReviewResponse20260403{}, // new table
	)
}

func (*migrateToAnalyticsAPI) Version() uint64 { return 20260403000001 }
func (*migrateToAnalyticsAPI) Name() string {
	return "migrate Codex plugin to the Codex Enterprise Analytics API"
}
