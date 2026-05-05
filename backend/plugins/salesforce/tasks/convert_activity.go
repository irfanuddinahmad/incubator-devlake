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
	"fmt"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/salesforce/models"
)

const salesforceDebounceWindow = 5 * time.Minute

var _ plugin.SubTaskEntryPoint = ConvertActivity

var ConvertActivityMeta = plugin.SubTaskMeta{
	Name:             "convertActivity",
	EntryPoint:       ConvertActivity,
	EnabledByDefault: true,
	Description:      "Convert Salesforce tool-layer activity events into domain activity records",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	Dependencies:     []*plugin.SubTaskMeta{&ExtractActivityMeta, &CollectUsersMeta},
}

func ConvertActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*SalesforceTaskData)
	if !ok {
		return errors.Default.New("task data is not SalesforceTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId
	scopeId := data.Options.ScopeId

	deleteClauses := []dal.Clause{
		dal.Where("source_system = ? AND connection_id = ? AND scope_id = ?", "salesforce", connectionId, scopeId),
	}
	cursorClauses := []dal.Clause{
		dal.From(&models.SalesforceActivityEvent{}),
		dal.Where("connection_id = ? AND scope_id = ?", connectionId, scopeId),
		dal.Orderby("occurred_at ASC"),
	}
	if data.Options.OccurredAfter != nil {
		bound := data.Options.OccurredAfter.UTC()
		deleteClauses = append(deleteClauses, dal.Where("action_time >= ?", bound))
		cursorClauses = append(cursorClauses, dal.Where("occurred_at >= ?", bound))
	}
	if data.Options.OccurredBefore != nil {
		bound := data.Options.OccurredBefore.UTC()
		deleteClauses = append(deleteClauses, dal.Where("action_time < ?", bound))
		cursorClauses = append(cursorClauses, dal.Where("occurred_at < ?", bound))
	}

	if err := db.Delete(&crossdomain.UserActivity{}, deleteClauses...); err != nil {
		return err
	}

	cursor, err := db.Cursor(cursorClauses...)
	if err != nil {
		return err
	}
	defer cursor.Close()

	idGen := didgen.NewDomainIdGenerator(&models.SalesforceActivityEvent{})
	userMap, err := loadSalesforceUserMap(db, connectionId)
	if err != nil {
		return err
	}
	activities, err := buildSalesforceActivities(db, cursor, idGen.Generate, userMap)
	if err != nil {
		return err
	}

	return createUserActivitiesInBatches(db, activities)
}

type salesforceActivityIdFunc func(pks ...interface{}) string

type salesforceGroupedActivity struct {
	groupId    string
	accountId  string
	userEmail  string
	userName   string
	nativeId   string
	actionType string
	objectType string
	objectId   string
	lastTime   time.Time
	lastEvent  string
	origin     models.SalesforceActivityEvent
	count      int
}

func buildSalesforceActivities(
	db dal.Dal,
	cursor dal.Rows,
	generateId salesforceActivityIdFunc,
	userMap map[string]models.SalesforceUser,
) ([]*crossdomain.UserActivity, errors.Error) {
	events := make([]models.SalesforceActivityEvent, 0)
	for cursor.Next() {
		event := &models.SalesforceActivityEvent{}
		if err := db.Fetch(cursor, event); err != nil {
			return nil, errors.Default.Wrap(err, "error fetching Salesforce activity event")
		}
		events = append(events, *event)
	}

	return buildSalesforceActivitiesFromEvents(events, generateId, func(email string) string {
		return resolveAccountIdByEmail(db, email)
	}, userMap), nil
}

func buildSalesforceActivitiesFromEvents(
	events []models.SalesforceActivityEvent,
	generateId salesforceActivityIdFunc,
	resolveAccountId func(email string) string,
	userMap map[string]models.SalesforceUser,
) []*crossdomain.UserActivity {
	if resolveAccountId == nil {
		resolveAccountId = func(string) string { return "" }
	}
	if userMap == nil {
		userMap = map[string]models.SalesforceUser{}
	}

	grouped := map[string]*salesforceGroupedActivity{}
	orderedKeys := make([]string, 0)

	for _, event := range events {
		eventCopy := event
		if strings.EqualFold(strings.TrimSpace(eventCopy.ActionType), "ignored") {
			continue
		}

		bucket := floorToDebounceWindow(eventCopy.OccurredAt.UTC(), salesforceDebounceWindow)
		normalizedAction := normalizeActionType(eventCopy.ActionType, "updated")
		normalizedObject := normalizeObjectType(eventCopy.ObjectType, eventCopy.SourceObjectType)
		userKey := strings.TrimSpace(eventCopy.ActingUserEmail)
		if userKey == "" {
			userKey = strings.TrimSpace(eventCopy.ActingUserId)
		}
		if userKey == "" {
			userKey = "unknown"
		}

		groupId := fmt.Sprintf("%s:%s:%s:%s:%d", userKey, normalizedAction, normalizedObject, eventCopy.ObjectId, bucket.Unix())
		group := grouped[groupId]
		if group == nil {
			resolvedEmail := strings.TrimSpace(eventCopy.ActingUserEmail)
			resolvedName := ""
			if user, ok := userMap[strings.TrimSpace(eventCopy.ActingUserId)]; ok {
				if strings.TrimSpace(user.Email) != "" {
					resolvedEmail = strings.TrimSpace(user.Email)
				}
				resolvedName = strings.TrimSpace(user.Name)
			}
			group = &salesforceGroupedActivity{
				groupId:    groupId,
				accountId:  resolveAccountId(resolvedEmail),
				userEmail:  resolvedEmail,
				userName:   fallbackDisplay(resolvedName, resolvedEmail, strings.TrimSpace(eventCopy.ActingUserId), "Salesforce user"),
				nativeId:   strings.TrimSpace(eventCopy.ActingUserId),
				actionType: normalizedAction,
				objectType: normalizedObject,
				objectId:   strings.TrimSpace(eventCopy.ObjectId),
				lastTime:   eventCopy.OccurredAt.UTC(),
				lastEvent:  eventCopy.EventId,
				origin:     eventCopy,
				count:      1,
			}
			grouped[groupId] = group
			orderedKeys = append(orderedKeys, groupId)
			continue
		}

		group.count++
		if eventCopy.OccurredAt.After(group.lastTime) {
			group.lastTime = eventCopy.OccurredAt.UTC()
			group.lastEvent = eventCopy.EventId
			group.origin = eventCopy
		}
	}

	activities := make([]*crossdomain.UserActivity, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		group := grouped[key]
		summary := fmt.Sprintf("Salesforce %s %s", group.objectType, group.actionType)
		if group.count > 1 {
			summary = fmt.Sprintf("Salesforce %s %s x%d", group.objectType, group.actionType, group.count)
		}

		activity := &crossdomain.UserActivity{
			DomainEntity:  crossdomain.UserActivity{}.DomainEntity,
			ConnectionId:  group.origin.ConnectionId,
			ScopeId:       group.origin.ScopeId,
			SourceSystem:  "salesforce",
			SourceEventId: group.lastEvent,
			AccountId:     group.accountId,
			UserEmail:     group.userEmail,
			UserDisplay:   group.userName,
			NativeUserId:  group.nativeId,
			ActionType:    group.actionType,
			ObjectType:    group.objectType,
			ObjectId:      group.objectId,
			ObjectRef:     fmt.Sprintf("%s:%s", group.objectType, group.objectId),
			ActionTime:    group.lastTime,
			ActionDay:     utcDay(group.lastTime),
			Summary:       summary,
		}
		if generateId != nil {
			activity.Id = generateId(group.origin.ConnectionId, group.origin.ScopeId, group.groupId)
		}
		activity.RawDataOrigin = group.origin.RawDataOrigin
		activities = append(activities, activity)
	}

	return activities
}
