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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	corePlugin "github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var salesforceTimeLayouts = []string{
	"2006-01-02T15:04:05.000-0700",
	time.RFC3339,
	"2006-01-02T15:04:05Z",
}

type salesforceQueryResponse struct {
	TotalSize      int               `json:"totalSize"`
	Done           bool              `json:"done"`
	NextRecordsURL string            `json:"nextRecordsUrl"`
	Records        []json.RawMessage `json:"records"`
}

func executeSalesforceQuery(
	apiClient *helper.ApiAsyncClient,
	apiVersion string,
	soql string,
	nextRecordsURL string,
) (*salesforceQueryResponse, string, errors.Error) {
	var (
		res *http.Response
		err errors.Error
	)

	if strings.TrimSpace(nextRecordsURL) != "" {
		res, err = apiClient.Get(normalizeSalesforcePath(nextRecordsURL), nil, nil)
	} else {
		res, err = apiClient.Get(
			fmt.Sprintf("services/data/%s/query", apiVersion),
			url.Values{"q": []string{soql}},
			nil,
		)
	}
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, "", errors.Convert(readErr)
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", errors.Default.New(fmt.Sprintf("salesforce query failed with status %d: %s", res.StatusCode, string(body)))
	}

	var response salesforceQueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", errors.Convert(err)
	}
	return &response, res.Request.URL.String(), nil
}

func normalizeSalesforcePath(path string) string {
	return strings.TrimPrefix(strings.TrimSpace(path), "/")
}

func rawSalesforceObjectTableSuffix(objectType string) string {
	return "salesforce_" + strings.ToLower(strings.TrimSpace(objectType))
}

func prepareSalesforceRawTable(
	taskCtx corePlugin.SubTaskContext,
	tableSuffix string,
	params salesforceRawParams,
	isIncremental bool,
) (string, string, errors.Error) {
	db := taskCtx.GetDal()
	tableName := "_raw_" + tableSuffix
	paramsValue := corePlugin.MarshalScopeParams(params)

	if err := db.AutoMigrate(&helper.RawData{}, dal.From(tableName)); err != nil {
		return "", "", err
	}
	if isIncremental {
		return tableName, paramsValue, nil
	}
	if err := db.Delete(&helper.RawData{}, dal.From(tableName), dal.Where("params = ?", paramsValue)); err != nil {
		return "", "", err
	}

	return tableName, paramsValue, nil
}

func insertSalesforceRawRows(
	db dal.Dal,
	tableName string,
	params string,
	requestURL string,
	records []json.RawMessage,
) errors.Error {
	if len(records) == 0 {
		return nil
	}

	rows := make([]*helper.RawData, len(records))
	for i, record := range records {
		rows[i] = &helper.RawData{
			Params: params,
			Data:   record,
			Url:    requestURL,
		}
	}
	return db.Create(rows, dal.From(tableName))
}

func parseSalesforceTime(raw string) (time.Time, errors.Error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range salesforceTimeLayouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.Default.New(fmt.Sprintf("failed to parse Salesforce time %q", raw))
}

func resolveSalesforceSince(collectedSince *time.Time, occurredAfter *time.Time, now time.Time) *time.Time {
	if collectedSince != nil && !collectedSince.IsZero() {
		t := collectedSince.UTC()
		return &t
	}
	if occurredAfter != nil {
		t := occurredAfter.UTC()
		return &t
	}
	t := now.UTC().AddDate(0, 0, -30)
	return &t
}

func formatSalesforceTimeLiteral(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
