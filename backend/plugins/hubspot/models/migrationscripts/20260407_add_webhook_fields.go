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

package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

type hubspotConnectionWebhookFields20260407 struct {
	EnableWebhook    bool   `gorm:"column:enable_webhook"`
	WebhookSharedKey string `gorm:"column:webhook_shared_key;type:varchar(255)"`
}

func (hubspotConnectionWebhookFields20260407) TableName() string {
	return "_tool_hubspot_connections"
}

type addHubspotWebhookFields struct{}

func (*addHubspotWebhookFields) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(basicRes, &hubspotConnectionWebhookFields20260407{})
}

func (*addHubspotWebhookFields) Version() uint64 {
	return 20260407000001
}

func (*addHubspotWebhookFields) Name() string {
	return "add webhook fields to HubSpot connection"
}
