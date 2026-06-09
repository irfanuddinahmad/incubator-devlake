<!--
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
-->

##### Q1. Which types of GitHub data are collected?

The data collected includes: `issues`, `repos`, `commits`, `branches`, `pull requests`, `pr comments`, `workflow runs`, `job runs`, `deployments`, `users`, etc.

For a comprehensive scope of data collection, refer to the [Supported Data Sources documentation](https://devlake.apache.org/docs/Overview/SupportedDataSources/#data-collection-scope-by-each-plugin).

##### Q2. What time range is covered in the data collection?

DevLake collects all available GitHub history by default unless you configure a start date. The time range can be adjusted on the project details page at any point.

##### Q3. How do I backfill an existing GitHub project?

Existing GitHub projects keep their configured start date. To backfill older data, open the project details page, clear the Time Range start date, and run a full sync. If the project had already synchronized with an old cutoff, DevLake may also have saved that cutoff in collector state; clear historical data for the affected data scope before rerunning if older records are still skipped.

##### Q4. Is it possible to transform the collected data?

Yes, data transformations can be applied by setting up a Scope Config for the selected repositories afterward.

##### Q5. How often is the data synchronized?

Data synchronization occurs daily. This frequency can be modified on the project details page as needed.
