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

package helper

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// ToJson FIXME
func ToJson(x any) json.RawMessage {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return b
}

func ToMap(x any) map[string]any {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	m := map[string]any{}
	if err = json.Unmarshal(b, &m); err != nil {
		panic(err)
	}
	return m
}

func FromMap[T any](x map[string]any) *T {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	t := new(T)
	if err = json.Unmarshal(b, &t); err != nil {
		panic(err)
	}
	return t
}

// ToCleanJson FIXME
func ToCleanJson(inline bool, x any) json.RawMessage {
	j, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	var m any
	if err = json.Unmarshal(j, &m); err != nil {
		panic(err)
	}
	m = cleanJsonValue(m)
	var b []byte
	if inline {
		b, err = json.Marshal(m)
	} else {
		b, err = json.MarshalIndent(m, "", "    ")
	}
	if err != nil {
		panic(err)
	}
	return b
}

func cleanJsonValue(value any) any {
	switch valueCasted := value.(type) {
	case map[string]any:
		removeNullsFromMap(valueCasted)
		redactSensitiveFields(valueCasted)
		return valueCasted
	case []any:
		for i, arrayValue := range valueCasted {
			valueCasted[i] = cleanJsonValue(arrayValue)
		}
		return valueCasted
	default:
		return value
	}
}

func removeNullsFromMap(m map[string]any) {
	refMap := reflect.ValueOf(m)
	for _, refKey := range refMap.MapKeys() {
		key := refKey.String()
		refValue := refMap.MapIndex(refKey)
		if refValue.IsNil() || refValue.Elem().IsZero() {
			delete(m, key)
			continue
		}
		value := refValue.Interface()
		if isNullTime(value) {
			delete(m, key)
			continue
		}
		switch valueCasted := value.(type) {
		case map[string]any:
			removeNullsFromMap(valueCasted)
		case []any:
			for _, arrayValue := range valueCasted {
				if m, ok := arrayValue.(map[string]any); ok {
					removeNullsFromMap(m)
				}
			}
		}
	}
}

func redactSensitiveFields(m map[string]any) {
	for key, value := range m {
		if isSensitiveKey(key) {
			m[key] = redactValue(value)
			continue
		}
		switch valueCasted := value.(type) {
		case map[string]any:
			redactSensitiveFields(valueCasted)
		case []any:
			for _, arrayValue := range valueCasted {
				if item, ok := arrayValue.(map[string]any); ok {
					redactSensitiveFields(item)
				}
			}
		}
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	return strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "privatekey")
}

func redactValue(value any) any {
	if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
		return str
	}
	return "******"
}

func isNullTime(value any) bool {
	if str, ok := value.(string); ok {
		if t, err := time.Parse("2006-01-02T15:04:05Z", str); err == nil {
			zeroTime := time.Time{}
			if t.Equal(zeroTime) {
				return true
			}
		}
	}
	return false
}
