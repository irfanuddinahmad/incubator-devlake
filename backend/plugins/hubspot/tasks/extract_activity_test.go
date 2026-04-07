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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHubspotOwnerId(t *testing.T) {
	assert.Equal(t, "123", extractHubspotOwnerId(map[string]interface{}{"hubspot_owner_id": "123"}))
	assert.Equal(t, "123", extractHubspotOwnerId(map[string]interface{}{"hubspot_owner_id": float64(123)}))
	assert.Equal(t, "", extractHubspotOwnerId(map[string]interface{}{"hubspot_owner_id": nil}))
	assert.Equal(t, "", extractHubspotOwnerId(nil))
}

func TestExtractHubspotOwnerEmail(t *testing.T) {
	assert.Equal(t, "owner@example.com", extractHubspotOwnerEmail(map[string]interface{}{
		"hubspot_owner_email": "owner@example.com",
	}))
	assert.Equal(t, "sender@example.com", extractHubspotOwnerEmail(map[string]interface{}{
		"hubspot_owner_email":   "",
		"hs_email_sender_email": "sender@example.com",
	}))
	assert.Equal(t, "", extractHubspotOwnerEmail(map[string]interface{}{
		"owner_email": nil,
	}))
	assert.Equal(t, "", extractHubspotOwnerEmail(nil))
}
