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

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIVersion = "v61.0"
	defaultLoginURL   = "https://login.salesforce.com"
	defaultScope      = "api refresh_token"
)

type options struct {
	clientID      string
	clientSecret  string
	loginURL      string
	callbackHost  string
	callbackPort  int
	apiVersion    string
	scope         string
	objectTypes   string
	occurredAfter string
	outputPath    string
	timeout       time.Duration
	writeConfig   bool
	force         bool
	openBrowser   bool
	printTokens   bool
}

type callbackResult struct {
	code string
	err  error
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	InstanceURL  string `json:"instance_url"`
	ID           string `json:"id"`
	IssuedAt     string `json:"issued_at"`
	Signature    string `json:"signature"`
	TokenType    string `json:"token_type"`
}

type localConfig struct {
	AuthMode      string
	AccessToken   string
	RefreshToken  string
	ClientID      string
	ClientSecret  string
	LoginURL      string
	InstanceURL   string
	APIVersion    string
	ObjectTypes   []string
	OccurredAfter string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	opts, err := parseOptions(args, stderr)
	if err == flag.ErrHelp {
		return nil
	}
	if err != nil {
		return err
	}
	if err := validateOptions(opts); err != nil {
		return err
	}

	state, err := randomState()
	if err != nil {
		return fmt.Errorf("generate OAuth state: %w", err)
	}

	callbackURL := buildCallbackURL(opts)
	callbacks := make(chan callbackResult, 1)
	server, listener, err := startCallbackServer(opts.callbackHost, opts.callbackPort, state, callbacks)
	if err != nil {
		return err
	}
	defer shutdownServer(server)

	authURL := buildAuthorizeURL(opts.loginURL, opts.clientID, callbackURL, opts.scope, state)
	fmt.Fprintf(stdout, "Listening for Salesforce OAuth callback on %s\n", listener.Addr().String())
	fmt.Fprintf(stdout, "Callback URL configured in Salesforce must be: %s\n", callbackURL)

	if opts.openBrowser {
		if err := openBrowser(authURL); err != nil {
			fmt.Fprintf(stderr, "Could not open browser automatically: %v\n", err)
			fmt.Fprintf(stdout, "Open this URL manually:\n%s\n", authURL)
		} else {
			fmt.Fprintln(stdout, "Opened Salesforce authorization URL in your browser.")
		}
	} else {
		fmt.Fprintf(stdout, "Open this URL in your browser:\n%s\n", authURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	result, err := waitForCallback(ctx, callbacks)
	if err != nil {
		return err
	}
	if result.err != nil {
		return result.err
	}

	token, err := exchangeCodeForToken(ctx, opts.loginURL, result.code, opts.clientID, opts.clientSecret, callbackURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		fmt.Fprintln(stderr, "Salesforce did not return a refresh_token. Check that the OAuth app includes refresh_token/offline_access scope.")
	}

	if opts.writeConfig {
		configPath := opts.outputPath
		if strings.TrimSpace(configPath) == "" {
			configPath, err = defaultLocalConfigPath()
			if err != nil {
				return err
			}
		}
		if err := writeLocalTestConfig(configPath, opts, token); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Wrote ignored local Salesforce e2e config: %s\n", configPath)
	}

	fmt.Fprintf(stdout, "Salesforce instance URL: %s\n", token.InstanceURL)
	fmt.Fprintf(stdout, "Refresh token returned: %t\n", strings.TrimSpace(token.RefreshToken) != "")
	if opts.printTokens || !opts.writeConfig {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(token); err != nil {
			return fmt.Errorf("print token response: %w", err)
		}
	}
	return nil
}

func parseOptions(args []string, output io.Writer) (options, error) {
	opts := options{}
	flags := flag.NewFlagSet("salesforce-oauth-helper", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.clientID, "client-id", "", "Salesforce Consumer Key")
	flags.StringVar(&opts.clientSecret, "client-secret", "", "Salesforce Consumer Secret")
	flags.StringVar(&opts.loginURL, "login-url", defaultLoginURL, "Salesforce login URL, for example https://login.salesforce.com, https://test.salesforce.com, or your My Domain URL")
	flags.StringVar(&opts.callbackHost, "callback-host", "localhost", "Local callback host configured in Salesforce")
	flags.IntVar(&opts.callbackPort, "callback-port", 1717, "Local callback port configured in Salesforce")
	flags.StringVar(&opts.apiVersion, "api-version", defaultAPIVersion, "Salesforce REST API version for the generated e2e config")
	flags.StringVar(&opts.scope, "scope", defaultScope, "OAuth scopes requested from Salesforce")
	flags.StringVar(&opts.objectTypes, "object-types", "Lead,Opportunity,Case", "Comma-separated Salesforce object types for the generated e2e config")
	flags.StringVar(&opts.occurredAfter, "occurred-after", "", "Optional RFC3339 lower bound for the e2e sync window")
	flags.StringVar(&opts.outputPath, "output", "", "Path for generated salesforce_local_test.go; defaults to the Salesforce manual e2e folder")
	flags.DurationVar(&opts.timeout, "timeout", 10*time.Minute, "How long to wait for the browser callback")
	flags.BoolVar(&opts.writeConfig, "write-config", true, "Write an ignored salesforce_local_test.go file for manual e2e tests")
	flags.BoolVar(&opts.force, "force", false, "Overwrite the generated local config file if it already exists")
	flags.BoolVar(&opts.openBrowser, "open-browser", true, "Open the Salesforce authorization URL automatically")
	flags.BoolVar(&opts.printTokens, "print-tokens", false, "Print the full token response to stdout")

	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}

func validateOptions(opts options) error {
	if strings.TrimSpace(opts.clientID) == "" {
		return fmt.Errorf("client-id is required")
	}
	if strings.TrimSpace(opts.clientSecret) == "" {
		return fmt.Errorf("client-secret is required")
	}
	if _, err := normalizeURL(opts.loginURL); err != nil {
		return fmt.Errorf("login-url is invalid: %w", err)
	}
	if opts.callbackPort <= 0 || opts.callbackPort > 65535 {
		return fmt.Errorf("callback-port must be between 1 and 65535")
	}
	if opts.timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if _, err := parseObjectTypes(opts.objectTypes); err != nil {
		return err
	}
	if err := validateRFC3339("occurred-after", opts.occurredAfter); err != nil {
		return err
	}
	return nil
}

func validateRFC3339(name string, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%s must be an RFC3339 timestamp: %w", name, err)
	}
	return nil
}

func randomState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func buildCallbackURL(opts options) string {
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(opts.callbackHost, strconv.Itoa(opts.callbackPort)),
		Path:   "/callback",
	}).String()
}

func buildAuthorizeURL(loginURL string, clientID string, callbackURL string, scope string, state string) string {
	base := strings.TrimRight(strings.TrimSpace(loginURL), "/") + "/services/oauth2/authorize"
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", strings.TrimSpace(clientID))
	query.Set("redirect_uri", callbackURL)
	query.Set("scope", strings.TrimSpace(scope))
	query.Set("state", state)
	return base + "?" + query.Encode()
}

func startCallbackServer(host string, port int, state string, callbacks chan<- callbackResult) (*http.Server, net.Listener, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		if returnedState := query.Get("state"); returnedState != state {
			writeCallbackError(w, http.StatusBadRequest, "Invalid OAuth state.")
			sendCallback(callbacks, callbackResult{err: fmt.Errorf("invalid OAuth state")})
			return
		}
		if oauthErr := query.Get("error"); oauthErr != "" {
			description := query.Get("error_description")
			writeCallbackError(w, http.StatusBadRequest, "Salesforce returned an OAuth error.")
			sendCallback(callbacks, callbackResult{err: fmt.Errorf("salesforce OAuth error: %s %s", oauthErr, description)})
			return
		}
		code := strings.TrimSpace(query.Get("code"))
		if code == "" {
			writeCallbackError(w, http.StatusBadRequest, "Missing OAuth code.")
			sendCallback(callbacks, callbackResult{err: fmt.Errorf("callback did not include code")})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<html><body><h1>Salesforce connected.</h1><p>You can return to the terminal.</p></body></html>")
		sendCallback(callbacks, callbackResult{code: code})
	})

	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("listen on %s: %w", address, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			sendCallback(callbacks, callbackResult{err: fmt.Errorf("callback server failed: %w", err)})
		}
	}()
	return server, listener, nil
}

func sendCallback(callbacks chan<- callbackResult, result callbackResult) {
	select {
	case callbacks <- result:
	default:
	}
}

func writeCallbackError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<html><body><h1>%s</h1><p>You can return to the terminal.</p></body></html>", html.EscapeString(message))
}

func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

func waitForCallback(ctx context.Context, callbacks <-chan callbackResult) (callbackResult, error) {
	select {
	case result := <-callbacks:
		return result, nil
	case <-ctx.Done():
		return callbackResult{}, fmt.Errorf("timed out waiting for Salesforce callback")
	}
}

func exchangeCodeForToken(ctx context.Context, loginURL string, code string, clientID string, clientSecret string, callbackURL string) (tokenResponse, error) {
	loginURL = strings.TrimRight(strings.TrimSpace(loginURL), "/")
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", strings.TrimSpace(clientID))
	form.Set("client_secret", strings.TrimSpace(clientSecret))
	form.Set("redirect_uri", callbackURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL+"/services/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return tokenResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		return tokenResponse{}, fmt.Errorf("token exchange failed with status %d: %s", res.StatusCode, string(body))
	}

	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return tokenResponse{}, err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return tokenResponse{}, fmt.Errorf("token response did not include access_token")
	}
	if strings.TrimSpace(token.InstanceURL) == "" {
		return tokenResponse{}, fmt.Errorf("token response did not include instance_url")
	}
	return token, nil
}

func writeLocalTestConfig(path string, opts options, token tokenResponse) error {
	if _, err := os.Stat(path); err == nil && !opts.force {
		return fmt.Errorf("%s already exists; pass -force=true to overwrite it", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	objectTypes, err := parseObjectTypes(opts.objectTypes)
	if err != nil {
		return err
	}

	authMode := "access_token"
	if strings.TrimSpace(token.RefreshToken) != "" {
		authMode = "refresh_token"
	}
	content, err := renderLocalConfig(localConfig{
		AuthMode:      authMode,
		AccessToken:   token.AccessToken,
		RefreshToken:  token.RefreshToken,
		ClientID:      opts.clientID,
		ClientSecret:  opts.clientSecret,
		LoginURL:      opts.loginURL,
		InstanceURL:   token.InstanceURL,
		APIVersion:    opts.apiVersion,
		ObjectTypes:   objectTypes,
		OccurredAfter: opts.occurredAfter,
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0600)
}

func renderLocalConfig(cfg localConfig) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("package salesforce\n\n")
	builder.WriteString("import \"github.com/apache/incubator-devlake/test/helper\"\n\n")
	builder.WriteString("func init() {\n")
	builder.WriteString("\thelper.SetTestConfig(TestConfig{\n")
	writeConfigString(&builder, "AuthMode", cfg.AuthMode)
	writeConfigString(&builder, "AccessToken", cfg.AccessToken)
	writeConfigString(&builder, "RefreshToken", cfg.RefreshToken)
	writeConfigString(&builder, "ClientId", cfg.ClientID)
	writeConfigString(&builder, "ClientSecret", cfg.ClientSecret)
	writeConfigString(&builder, "LoginUrl", cfg.LoginURL)
	writeConfigString(&builder, "InstanceUrl", cfg.InstanceURL)
	writeConfigString(&builder, "ApiVersion", cfg.APIVersion)
	if len(cfg.ObjectTypes) > 0 {
		builder.WriteString("\t\tObjectTypes: []string{")
		for i, objectType := range cfg.ObjectTypes {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Quote(objectType))
		}
		builder.WriteString("},\n")
	}
	writeConfigString(&builder, "OccurredAfter", cfg.OccurredAfter)
	builder.WriteString("\t})\n")
	builder.WriteString("}\n")

	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func writeConfigString(builder *strings.Builder, field string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	builder.WriteString("\t\t")
	builder.WriteString(field)
	builder.WriteString(": ")
	builder.WriteString(strconv.Quote(strings.TrimSpace(value)))
	builder.WriteString(",\n")
}

func parseObjectTypes(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	objectTypes := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		objectType := strings.TrimSpace(part)
		if objectType == "" {
			continue
		}
		if _, ok := seen[objectType]; ok {
			continue
		}
		seen[objectType] = struct{}{}
		objectTypes = append(objectTypes, objectType)
	}
	if len(objectTypes) == 0 {
		return nil, fmt.Errorf("object-types must include at least one Salesforce object")
	}
	return objectTypes, nil
}

func defaultLocalConfigPath() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot resolve oauth helper source path")
	}
	return filepath.Join(filepath.Dir(filepath.Dir(sourceFile)), "salesforce_local_test.go"), nil
}

func normalizeURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	return parsed, nil
}
