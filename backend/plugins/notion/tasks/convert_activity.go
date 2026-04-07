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
	"github.com/apache/incubator-devlake/plugins/notion/models"
)

const notionDebounceWindow = 5 * time.Minute

var _ plugin.SubTaskEntryPoint = ConvertActivity

var ConvertActivityMeta = plugin.SubTaskMeta{
	Name:             "convertActivity",
	EntryPoint:       ConvertActivity,
	EnabledByDefault: true,
	Description:      "Convert Notion tool-layer activity events into domain activity records",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
	Dependencies:     []*plugin.SubTaskMeta{&ExtractActivityMeta},
}

func ConvertActivity(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*NotionTaskData)
	if !ok {
		return errors.Default.New("task data is not NotionTaskData")
	}

	db := taskCtx.GetDal()
	connectionId := data.Options.ConnectionId
	scopeId := data.Options.ScopeId

	if err := db.Delete(
		&crossdomain.UserActivity{},
		dal.Where("source_system = ? AND connection_id = ? AND scope_id = ?", "notion", connectionId, scopeId),
	); err != nil {
		return err
	}

	cursor, err := db.Cursor(
		dal.From(&models.NotionActivityEvent{}),
		dal.Where("connection_id = ? AND scope_id = ?", connectionId, scopeId),
		dal.Orderby("occurred_at ASC"),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	idGen := didgen.NewDomainIdGenerator(&models.NotionActivityEvent{})
	activities, err := buildNotionActivities(db, cursor, idGen)
	if err != nil {
		return err
	}

	return createUserActivitiesInBatches(db, activities)
}

type notionGroupedActivity struct {
	groupId    string
	accountId  string
	userEmail  string
	userName   string
	nativeId   string
	actionType string
	objectType string
	objectId   string
	bucket     time.Time
	lastTime   time.Time
	lastEvent  string
	origin     models.NotionActivityEvent
	count      int
}

func buildNotionActivities(db dal.Dal, cursor dal.Rows, idGen *didgen.DomainIdGenerator) ([]*crossdomain.UserActivity, errors.Error) {
	events := make([]models.NotionActivityEvent, 0)

	for cursor.Next() {
		event := &models.NotionActivityEvent{}
		if err := db.Fetch(cursor, event); err != nil {
			return nil, errors.Default.Wrap(err, "error fetching Notion activity event")
		}
		events = append(events, *event)
	}

	activities := buildNotionActivitiesFromEvents(events, idGen, func(email string) string {
		return resolveAccountIdByEmail(db, email)
	})
	return activities, nil
}

func buildNotionActivitiesFromEvents(
	events []models.NotionActivityEvent,
	idGen *didgen.DomainIdGenerator,
	resolveAccountId func(email string) string,
) []*crossdomain.UserActivity {
	if resolveAccountId == nil {
		resolveAccountId = func(string) string { return "" }
	}

	grouped := map[string]*notionGroupedActivity{}
	orderedKeys := make([]string, 0)

	for _, event := range events {
		eventCopy := event

		bucket := floorToDebounceWindow(eventCopy.OccurredAt.UTC(), notionDebounceWindow)
		normalizedAction := normalizeActionType(eventCopy.ActionType, "edited")
		normalizedObject := normalizeObjectType(eventCopy.ObjectType, eventCopy.SourceObjectType)
		userKey := strings.TrimSpace(eventCopy.EditorUserEmail)
		if userKey == "" {
			userKey = strings.TrimSpace(eventCopy.EditorUserId)
		}
		if userKey == "" {
			userKey = "unknown"
		}

		groupId := fmt.Sprintf("%s:%s:%s:%s:%d", userKey, normalizedAction, normalizedObject, eventCopy.ObjectId, bucket.Unix())
		group := grouped[groupId]
		if group == nil {
			group = &notionGroupedActivity{
				groupId:    groupId,
				accountId:  resolveAccountId(eventCopy.EditorUserEmail),
				userEmail:  strings.TrimSpace(eventCopy.EditorUserEmail),
				userName:   fallbackDisplay(strings.TrimSpace(eventCopy.EditorUserEmail), strings.TrimSpace(eventCopy.EditorUserId), "Notion user"),
				nativeId:   strings.TrimSpace(eventCopy.EditorUserId),
				actionType: normalizedAction,
				objectType: normalizedObject,
				objectId:   strings.TrimSpace(eventCopy.ObjectId),
				bucket:     bucket,
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
		actionDay := utcDay(group.lastTime)
		objectRef := fmt.Sprintf("%s:%s", group.objectType, group.objectId)
		summary := fmt.Sprintf("Notion %s %s", group.objectType, group.actionType)
		if group.count > 1 {
			summary = fmt.Sprintf("Notion %s %s x%d", group.objectType, group.actionType, group.count)
		}

		activity := &crossdomain.UserActivity{
			DomainEntity:  crossdomain.UserActivity{}.DomainEntity,
			ConnectionId:  group.origin.ConnectionId,
			ScopeId:       group.origin.ScopeId,
			SourceSystem:  "notion",
			SourceEventId: group.lastEvent,
			AccountId:     group.accountId,
			UserEmail:     group.userEmail,
			UserDisplay:   group.userName,
			NativeUserId:  group.nativeId,
			ActionType:    group.actionType,
			ObjectType:    group.objectType,
			ObjectId:      group.objectId,
			ObjectRef:     objectRef,
			ActionTime:    group.lastTime,
			ActionDay:     actionDay,
			Summary:       summary,
		}
		activity.Id = idGen.Generate(group.origin.ConnectionId, group.origin.ScopeId, group.groupId)
		activity.RawDataOrigin = group.origin.RawDataOrigin
		activities = append(activities, activity)
	}

	return activities
}
