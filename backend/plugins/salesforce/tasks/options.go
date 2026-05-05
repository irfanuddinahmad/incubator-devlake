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
)

var defaultSalesforceObjectTypes = []string{
	"Account",
	"Contact",
	"Lead",
	"Opportunity",
	"Case",
	"Task",
	"Event",
}

var salesforceObjectAliases = map[string]string{
	"account":       "Account",
	"accounts":      "Account",
	"contact":       "Contact",
	"contacts":      "Contact",
	"lead":          "Lead",
	"leads":         "Lead",
	"opportunity":   "Opportunity",
	"opportunities": "Opportunity",
	"case":          "Case",
	"cases":         "Case",
	"task":          "Task",
	"tasks":         "Task",
	"event":         "Event",
	"events":        "Event",
}

type SalesforceOptions struct {
	ConnectionId uint64   `json:"connectionId" mapstructure:"connectionId"`
	ScopeId      string   `json:"scopeId" mapstructure:"scopeId"`
	ObjectTypes  []string `json:"objectTypes" mapstructure:"objectTypes"`
	UseCdc       bool     `json:"useCdc" mapstructure:"useCdc"`
	// MaxUsers caps user-collection pagination. 0 means no cap (use the default).
	MaxUsers int `json:"maxUsers" mapstructure:"maxUsers"`

	OccurredAfter  *time.Time `json:"occurredAfter" mapstructure:"occurredAfter"`
	OccurredBefore *time.Time `json:"occurredBefore" mapstructure:"occurredBefore"`
}

const DefaultMaxUsers = 50000

type salesforceRawParams struct {
	ConnectionId uint64 `json:"connectionId"`
	ScopeId      string `json:"scopeId"`
	ObjectType   string `json:"objectType"`
}

func (p salesforceRawParams) GetParams() any { return p }

func DefaultObjectTypes() []string {
	result := make([]string, len(defaultSalesforceObjectTypes))
	copy(result, defaultSalesforceObjectTypes)
	return result
}

func IsCanonicalSalesforceObjectType(objectType string) bool {
	trimmed := strings.TrimSpace(objectType)
	if trimmed == "" {
		return false
	}
	for _, canonical := range defaultSalesforceObjectTypes {
		if canonical == trimmed {
			return true
		}
	}
	return false
}

func ResolveObjectTypes(requested []string) []string {
	selected := requested
	if len(selected) == 0 {
		return DefaultObjectTypes()
	}

	result := make([]string, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, raw := range selected {
		key := strings.TrimSpace(strings.ToLower(raw))
		canonical, ok := salesforceObjectAliases[key]
		if !ok {
			continue
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		result = append(result, canonical)
	}
	if len(result) == 0 {
		return DefaultObjectTypes()
	}
	return result
}
