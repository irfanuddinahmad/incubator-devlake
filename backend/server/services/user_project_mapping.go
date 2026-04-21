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

package services

import (
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models"
)

// GetUserProjectMappings returns all project mappings for a given Grafana user login.
func GetUserProjectMappings(userLogin string) ([]*models.UserProjectMapping, errors.Error) {
	var mappings []*models.UserProjectMapping
	err := db.All(&mappings, dal.Where("user_login = ?", userLogin))
	if err != nil {
		return nil, errors.Default.Wrap(err, "error getting user project mappings")
	}
	return mappings, nil
}

// GetAllUserProjectMappings returns all mappings (for admin view).
func GetAllUserProjectMappings() ([]*models.UserProjectMapping, errors.Error) {
	var mappings []*models.UserProjectMapping
	err := db.All(&mappings, dal.Orderby("user_login, project_name"))
	if err != nil {
		return nil, errors.Default.Wrap(err, "error getting all user project mappings")
	}
	return mappings, nil
}

// CreateUserProjectMapping creates or updates a user→project mapping.
func CreateUserProjectMapping(mapping *models.UserProjectMapping) errors.Error {
	// verify the project exists
	if _, err := GetProject(mapping.ProjectName); err != nil {
		return errors.BadInput.New("project not found: " + mapping.ProjectName)
	}
	if err := db.CreateOrUpdate(mapping); err != nil {
		return errors.Default.Wrap(err, "error creating user project mapping")
	}
	return nil
}

// DeleteUserProjectMapping deletes a specific user→project mapping.
func DeleteUserProjectMapping(userLogin, projectName string) errors.Error {
	err := db.Delete(
		&models.UserProjectMapping{},
		dal.Where("user_login = ? AND project_name = ?", userLogin, projectName),
	)
	if err != nil {
		return errors.Default.Wrap(err, "error deleting user project mapping")
	}
	return nil
}

// DeleteAllMappingsForUser deletes all mappings for a given user.
func DeleteAllMappingsForUser(userLogin string) errors.Error {
	err := db.Delete(
		&models.UserProjectMapping{},
		dal.Where("user_login = ?", userLogin),
	)
	if err != nil {
		return errors.Default.Wrap(err, "error deleting user project mappings")
	}
	return nil
}
