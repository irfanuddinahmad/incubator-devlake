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

// CodexCodeReview stores one code-review activity record from the Analytics API
// endpoint GET /analytics/codex/workspaces/{workspace_id}/code_reviews.
//
// Each record represents the Codex-generated review activity for a single pull
// request on a given day.
type CodexCodeReview struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	// PrUrl is the canonical pull-request URL used as a stable identifier.
	PrUrl string `gorm:"primaryKey;type:varchar(500);default:''" json:"prUrl"`

	// ReviewsCompleted is the number of Codex code reviews finished for this PR.
	ReviewsCompleted int64 `json:"reviewsCompleted"`
	// CommentsGenerated is the total number of review comments Codex posted.
	CommentsGenerated int64 `json:"commentsGenerated"`

	// Severity breakdown of generated comments.
	SeverityLow      int64 `json:"severityLow"`
	SeverityMedium   int64 `json:"severityMedium"`
	SeverityHigh     int64 `json:"severityHigh"`
	SeverityCritical int64 `json:"severityCritical"`
}

func (CodexCodeReview) TableName() string {
	return "_tool_codex_code_reviews"
}
