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

package tasks

import (
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

func createUserActivitiesInBatches(db dal.Dal, activities []*crossdomain.UserActivity) errors.Error {
	const batchSize = 200
	for start := 0; start < len(activities); start += batchSize {
		end := start + batchSize
		if end > len(activities) {
			end = len(activities)
		}
		if err := db.Create(activities[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func resolveAccountIdByEmail(db dal.Dal, email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	var account crossdomain.Account
	if err := db.First(&account, dal.Where("email = ?", email)); err != nil {
		return ""
	}
	return account.Id
}

func floorToDebounceWindow(t time.Time, window time.Duration) time.Time {
	if window <= 0 {
		return t.UTC()
	}
	utc := t.UTC()
	return utc.Truncate(window)
}

func utcDay(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeActionType(actionType, fallback string) string {
	actionType = strings.TrimSpace(strings.ToLower(actionType))
	if actionType == "" {
		return fallback
	}
	return actionType
}

func normalizeObjectType(objectType, fallback string) string {
	objectType = strings.TrimSpace(strings.ToLower(objectType))
	if objectType == "" {
		objectType = strings.TrimSpace(strings.ToLower(fallback))
	}
	if objectType == "" {
		return "object"
	}
	return objectType
}

func fallbackDisplay(name, email, nativeId, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if strings.TrimSpace(email) != "" {
		return strings.TrimSpace(email)
	}
	if strings.TrimSpace(nativeId) != "" {
		return strings.TrimSpace(nativeId)
	}
	return fallback
}

// loadNotionUserMap loads all NotionUser records for the given connectionId into a
// map keyed by UserId for fast lookup during conversion.
func loadNotionUserMap(db dal.Dal, connectionId uint64) map[string]models.NotionUser {
	userMap := map[string]models.NotionUser{}
	var users []models.NotionUser
	if err := db.All(&users, dal.Where("connection_id = ?", connectionId)); err != nil {
		return userMap
	}
	for _, u := range users {
		userMap[u.UserId] = u
	}
	return userMap
}
