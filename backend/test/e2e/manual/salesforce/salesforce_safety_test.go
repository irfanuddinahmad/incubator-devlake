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

package salesforce

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSafeE2EDatabaseURL(t *testing.T) {
	require.NoError(t, validateSafeE2EDatabaseURL("mysql://merico:merico@127.0.0.1:3306/lake_test?parseTime=true", ""))
	require.NoError(t, validateSafeE2EDatabaseURL("mysql://merico:merico@127.0.0.1:3306/devlake-salesforce-test", ""))
	require.NoError(t, validateSafeE2EDatabaseURL("mysql://merico:merico@127.0.0.1:3306/salesforce_test_20260421", ""))

	require.ErrorContains(t, validateSafeE2EDatabaseURL("", ""), "E2E_DB_URL")
	require.ErrorContains(t, validateSafeE2EDatabaseURL("mysql://merico:merico@127.0.0.1:3306/lake?parseTime=true", ""), "refusing to run")
	require.NoError(t, validateSafeE2EDatabaseURL("mysql://merico:merico@127.0.0.1:3306/lake?parseTime=true", "true"))
}

func TestDatabaseNameFromURL(t *testing.T) {
	dbName, err := databaseNameFromURL("mysql://user:pass@localhost:3306/lake_test?parseTime=true")
	require.NoError(t, err)
	require.Equal(t, "lake_test", dbName)

	_, err = databaseNameFromURL("mysql://user:pass@localhost:3306/?parseTime=true")
	require.ErrorContains(t, err, "database name")
}
