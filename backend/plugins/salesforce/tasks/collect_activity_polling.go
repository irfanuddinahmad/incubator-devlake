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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var _ plugin.SubTaskEntryPoint = CollectActivityPolling

var CollectActivityPollingMeta = plugin.SubTaskMeta{
	Name:             "collectActivityPolling",
	EntryPoint:       CollectActivityPolling,
	EnabledByDefault: true,
	Description:      "Collect Salesforce activity records via SOQL polling",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CROSS},
}

func CollectActivityPolling(taskCtx plugin.SubTaskContext) errors.Error {
	data, ok := taskCtx.TaskContext().GetData().(*SalesforceTaskData)
	if !ok {
		return errors.Default.New("task data is not SalesforceTaskData")
	}

	apiClient, err := CreateApiClient(taskCtx.TaskContext(), data.Connection)
	if err != nil {
		return err
	}

	until := data.Options.OccurredBefore

	for _, objectType := range ResolveObjectTypes(data.Options.ObjectTypes) {
		params := salesforceRawParams{
			ConnectionId: data.Options.ConnectionId,
			ScopeId:      data.Options.ScopeId,
			ObjectType:   objectType,
		}
		rawArgs := helper.RawDataSubTaskArgs{
			Ctx:     taskCtx,
			Table:   rawSalesforceObjectTableSuffix(objectType),
			Options: params,
			Params:  params,
		}
		collector, err := helper.NewStatefulApiCollector(rawArgs)
		if err != nil {
			return err
		}

		since := resolveSalesforceSince(collector.GetSince(), data.Options.OccurredAfter, time.Now())
		tableName, paramsValue, err := prepareSalesforceRawTable(taskCtx, rawSalesforceObjectTableSuffix(objectType), params, collector.IsIncremental())
		if err != nil {
			return err
		}

		soql := buildSalesforceActivityQuery(objectType, since, until)
		next := ""
		for {
			response, requestURL, err := executeSalesforceQuery(apiClient, data.Connection.GetVersion(), soql, next)
			if err != nil {
				return err
			}
			if err := insertSalesforceRawRows(taskCtx.GetDal(), tableName, paramsValue, requestURL, response.Records); err != nil {
				return err
			}
			next = strings.TrimSpace(response.NextRecordsURL)
			if next == "" {
				break
			}
		}
		if err := closeSalesforceActivityCollector(collector, since, until); err != nil {
			return err
		}
	}

	return nil
}

func closeSalesforceActivityCollector(collector *helper.StatefulApiCollector, since *time.Time, until *time.Time) errors.Error {
	if until == nil {
		return collector.Close()
	}

	return collector.CloseWithUntil(resolveSalesforceActivityCheckpoint(since, until, collector.GetUntil()))
}

func resolveSalesforceActivityCheckpoint(since *time.Time, until *time.Time, runUntil *time.Time) time.Time {
	checkpoint := until.UTC()
	if runUntil != nil && runUntil.Before(checkpoint) {
		checkpoint = runUntil.UTC()
	}
	if since != nil {
		normalizedSince := since.UTC()
		if checkpoint.Before(normalizedSince) {
			checkpoint = normalizedSince
		}
	}
	return checkpoint
}

func buildSalesforceActivityQuery(objectType string, since *time.Time, until *time.Time) string {
	filters := make([]string, 0, 2)
	if since != nil {
		filters = append(filters, fmt.Sprintf("SystemModstamp >= %s", formatSalesforceTimeLiteral(*since)))
	}
	if until != nil {
		filters = append(filters, fmt.Sprintf("SystemModstamp < %s", formatSalesforceTimeLiteral(until.UTC())))
	}

	query := fmt.Sprintf(
		"SELECT Id, CreatedDate, CreatedById, LastModifiedDate, LastModifiedById, SystemModstamp FROM %s",
		objectType,
	)
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY SystemModstamp ASC, Id ASC"
	return query
}
