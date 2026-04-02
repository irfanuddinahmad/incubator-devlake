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

// CodexCodeReviewResponse stores one user-engagement record from the Analytics API
// endpoint GET /analytics/codex/workspaces/{workspace_id}/code_review_responses.
//
// Each record captures how a single user interacted with Codex code-review comments
// on a given day: replies, upvotes, and downvotes.
type CodexCodeReviewResponse struct {
	common.NoPKModel

	ConnectionId uint64    `gorm:"primaryKey" json:"connectionId"`
	ScopeId      string    `gorm:"primaryKey;type:varchar(255)" json:"scopeId"`
	Date         time.Time `gorm:"primaryKey;type:date" json:"date"`
	UserEmail    string    `gorm:"primaryKey;type:varchar(255);default:''" json:"userEmail"`

	// Replies is the number of comments the user posted in response to Codex feedback.
	Replies int64 `json:"replies"`
	// Upvotes is the count of positive reactions (thumbs-up / agree) given to Codex comments.
	Upvotes int64 `json:"upvotes"`
	// Downvotes is the count of negative reactions (thumbs-down / disagree) given to Codex comments.
	Downvotes int64 `json:"downvotes"`
}

func (CodexCodeReviewResponse) TableName() string {
	return "_tool_codex_code_review_responses"
}
