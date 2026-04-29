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

package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/utils"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"gorm.io/gorm"
)

const DefaultEndpoint = "https://login.salesforce.com"
const DefaultApiVersion = "v61.0"

const (
	AuthModeAccessToken  = "access_token"
	AuthModeRefreshToken = "refresh_token"
)

type SalesforceConn struct {
	helper.RestConnection `mapstructure:",squash"`
	AuthMode              string `mapstructure:"authMode" json:"authMode" gorm:"column:auth_mode;type:varchar(32)"`
	AccessToken           string `mapstructure:"accessToken" json:"accessToken" gorm:"column:access_token;serializer:encdec"`
	RefreshToken          string `mapstructure:"refreshToken" json:"refreshToken" gorm:"column:refresh_token;serializer:encdec"`
	ClientId              string `mapstructure:"clientId" json:"clientId" gorm:"column:client_id;type:varchar(255)"`
	ClientSecret          string `mapstructure:"clientSecret" json:"clientSecret" gorm:"column:client_secret;serializer:encdec"`
	LoginUrl              string `mapstructure:"loginUrl" json:"loginUrl" gorm:"column:login_url;type:varchar(255)"`
	InstanceUrl           string `mapstructure:"instanceUrl" json:"instanceUrl" gorm:"column:instance_url;type:varchar(255)"`
	ApiVersion            string `mapstructure:"apiVersion" json:"apiVersion" gorm:"column:api_version;type:varchar(32)"`
	RateLimitPerHour      int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`

	tokenExpiresAt time.Time `gorm:"-" json:"-"`
}

func (conn *SalesforceConn) SetupAuthentication(request *http.Request) errors.Error {
	if conn == nil {
		return errors.BadInput.New("connection is required")
	}

	token, instanceURL, err := conn.resolveCredentials()
	if err != nil {
		return err
	}

	if request != nil && request.URL != nil && strings.TrimSpace(instanceURL) != "" {
		if parsed, parseErr := url.Parse(instanceURL); parseErr == nil {
			request.URL.Scheme = parsed.Scheme
			request.URL.Host = parsed.Host
		}
	}

	request.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (conn *SalesforceConn) resolveCredentials() (string, string, errors.Error) {
	if conn == nil {
		return "", "", errors.BadInput.New("connection is required")
	}

	authMode := conn.ResolveAuthMode()
	instanceURL := strings.TrimSpace(conn.InstanceUrl)
	if instanceURL == "" && strings.TrimSpace(conn.Endpoint) != "" && !strings.EqualFold(strings.TrimSpace(conn.Endpoint), strings.TrimSpace(conn.LoginUrl)) {
		instanceURL = strings.TrimSpace(conn.Endpoint)
	}

	switch authMode {
	case AuthModeRefreshToken:
		if strings.TrimSpace(conn.AccessToken) == "" || instanceURL == "" || conn.tokenExpiresAt.IsZero() || time.Now().After(conn.tokenExpiresAt) {
			if err := conn.refreshAccessToken(); err != nil {
				return "", "", err
			}
			instanceURL = strings.TrimSpace(conn.InstanceUrl)
		}
		if strings.TrimSpace(conn.AccessToken) == "" {
			return "", "", errors.BadInput.New("accessToken is required after refresh")
		}
		return strings.TrimSpace(conn.AccessToken), instanceURL, nil
	default:
		token := strings.TrimSpace(conn.AccessToken)
		if token == "" {
			return "", "", errors.BadInput.New("accessToken is required")
		}
		if instanceURL == "" {
			return "", "", errors.BadInput.New("instanceUrl is required")
		}
		return token, instanceURL, nil
	}
}

func (conn *SalesforceConn) refreshAccessToken() errors.Error {
	loginURL := strings.TrimRight(strings.TrimSpace(conn.LoginUrl), "/")
	if loginURL == "" {
		loginURL = DefaultEndpoint
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", strings.TrimSpace(conn.ClientId))
	form.Set("client_secret", strings.TrimSpace(conn.ClientSecret))
	form.Set("refresh_token", strings.TrimSpace(conn.RefreshToken))

	req, err := http.NewRequest(http.MethodPost, loginURL+"/services/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Convert(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return errors.Convert(err)
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return errors.Convert(readErr)
	}
	if res.StatusCode != http.StatusOK {
		return errors.Default.New(fmt.Sprintf("salesforce token refresh failed with status %d: %s", res.StatusCode, sanitizeOAuthError(body)))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		InstanceURL string `json:"instance_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return errors.Convert(err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return errors.Default.New("salesforce token refresh response did not include access_token")
	}

	conn.AccessToken = strings.TrimSpace(payload.AccessToken)
	if strings.TrimSpace(payload.InstanceURL) != "" {
		conn.InstanceUrl = strings.TrimSpace(payload.InstanceURL)
		conn.Endpoint = conn.InstanceUrl
	}
	conn.tokenExpiresAt = time.Now().Add(50 * time.Minute)
	return nil
}

func (conn SalesforceConn) ResolveAuthMode() string {
	authMode := strings.TrimSpace(strings.ToLower(conn.AuthMode))
	if authMode != "" {
		return authMode
	}
	if strings.TrimSpace(conn.RefreshToken) != "" || strings.TrimSpace(conn.ClientId) != "" || strings.TrimSpace(conn.ClientSecret) != "" {
		return AuthModeRefreshToken
	}
	return AuthModeAccessToken
}

func (conn SalesforceConn) GetVersion() string {
	if strings.TrimSpace(conn.ApiVersion) == "" {
		return DefaultApiVersion
	}
	return strings.TrimSpace(conn.ApiVersion)
}

func (conn SalesforceConn) Sanitize() SalesforceConn {
	conn.AccessToken = utils.SanitizeString(conn.AccessToken)
	conn.RefreshToken = utils.SanitizeString(conn.RefreshToken)
	conn.ClientSecret = utils.SanitizeString(conn.ClientSecret)
	return conn
}

type SalesforceConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	SalesforceConn        `mapstructure:",squash"`
}

func (SalesforceConnection) TableName() string {
	return "_tool_salesforce_connections"
}

func (connection SalesforceConnection) Sanitize() SalesforceConnection {
	connection.SalesforceConn = connection.SalesforceConn.Sanitize()
	return connection
}

func (connection *SalesforceConnection) MergeFromRequest(target *SalesforceConnection, body map[string]interface{}) error {
	if target == nil {
		return nil
	}

	originalAccessToken := target.AccessToken
	originalRefreshToken := target.RefreshToken
	originalClientSecret := target.ClientSecret

	if err := helper.DecodeMapStruct(body, target, true); err != nil {
		return err
	}

	if target.AccessToken == "" || target.AccessToken == utils.SanitizeString(originalAccessToken) {
		target.AccessToken = originalAccessToken
	}
	if target.RefreshToken == "" || target.RefreshToken == utils.SanitizeString(originalRefreshToken) {
		target.RefreshToken = originalRefreshToken
	}
	if target.ClientSecret == "" || target.ClientSecret == utils.SanitizeString(originalClientSecret) {
		target.ClientSecret = originalClientSecret
	}

	target.Normalize()
	return target.validateForSave()
}

func (connection *SalesforceConnection) Normalize() {
	if connection == nil {
		return
	}

	connection.AuthMode = connection.ResolveAuthMode()
	connection.LoginUrl = strings.TrimSpace(connection.LoginUrl)
	if connection.LoginUrl == "" {
		connection.LoginUrl = DefaultEndpoint
	}
	connection.InstanceUrl = strings.TrimSpace(connection.InstanceUrl)
	connection.ApiVersion = connection.GetVersion()
	if connection.RateLimitPerHour <= 0 {
		connection.RateLimitPerHour = 5000
	}

	switch connection.ResolveAuthMode() {
	case AuthModeRefreshToken:
		if connection.InstanceUrl != "" {
			connection.Endpoint = connection.InstanceUrl
		} else {
			connection.Endpoint = connection.LoginUrl
		}
	default:
		if connection.InstanceUrl != "" {
			connection.Endpoint = connection.InstanceUrl
		}
	}
}

func (connection *SalesforceConnection) validateForSave() errors.Error {
	if connection == nil {
		return nil
	}

	switch connection.ResolveAuthMode() {
	case AuthModeRefreshToken:
		if err := validateHttpsURL("loginUrl", connection.LoginUrl); err != nil {
			return err
		}
		if strings.TrimSpace(connection.InstanceUrl) != "" {
			if err := validateHttpsURL("instanceUrl", connection.InstanceUrl); err != nil {
				return err
			}
		}
		if strings.TrimSpace(connection.RefreshToken) == "" {
			return errors.BadInput.New("refreshToken is required when authMode is refresh_token")
		}
		if strings.TrimSpace(connection.ClientId) == "" {
			return errors.BadInput.New("clientId is required when authMode is refresh_token")
		}
		if strings.TrimSpace(connection.ClientSecret) == "" {
			return errors.BadInput.New("clientSecret is required when authMode is refresh_token")
		}
	default:
		if strings.TrimSpace(connection.AccessToken) == "" {
			return errors.BadInput.New("accessToken is required when authMode is access_token")
		}
		if strings.TrimSpace(connection.InstanceUrl) == "" {
			return errors.BadInput.New("instanceUrl is required when authMode is access_token")
		}
		if err := validateHttpsURL("instanceUrl", connection.InstanceUrl); err != nil {
			return err
		}
	}
	return nil
}

func (connection *SalesforceConnection) Validate() errors.Error {
	if connection == nil {
		return nil
	}
	connection.Normalize()
	return connection.validateForSave()
}

func sanitizeOAuthError(body []byte) string {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error) != "" {
		if desc := strings.TrimSpace(payload.ErrorDescription); desc != "" {
			return fmt.Sprintf("%s: %s", payload.Error, desc)
		}
		return payload.Error
	}
	return "unexpected response from salesforce oauth endpoint"
}

func validateHttpsURL(fieldName string, rawURL string) errors.Error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return errors.BadInput.New(fmt.Sprintf("%s must be a valid https URL", fieldName))
	}
	if parsed.Scheme != "https" {
		return errors.BadInput.New(fmt.Sprintf("%s must start with https://", fieldName))
	}
	if parsed.Host == "" {
		return errors.BadInput.New(fmt.Sprintf("%s must be a valid https URL", fieldName))
	}
	return nil
}

func (connection *SalesforceConnection) BeforeSave(_ *gorm.DB) error {
	if connection == nil {
		return nil
	}
	if err := connection.Validate(); err != nil {
		return err
	}
	return nil
}
