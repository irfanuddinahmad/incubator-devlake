package service

import (
	stdctx "context"
	"net/http"

	corectx "github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/codex/models"
)

type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func TestConnection(ctx stdctx.Context, br corectx.BasicRes, connection *models.CodexConnection) (*TestConnectionResult, errors.Error) {
	if connection == nil {
		return nil, errors.BadInput.New("connection is required")
	}

	connection.Normalize()

	apiClient, err := helper.NewApiClientFromConnection(ctx, br, connection)
	if err != nil {
		return nil, err
	}

	resp, err := apiClient.Get("models", nil, nil)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to reach Codex API")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.Unauthorized.New("invalid API key")
	}
	if resp.StatusCode >= 400 {
		return nil, errors.Default.New("Codex API returned an unexpected error")
	}

	return &TestConnectionResult{
		Success: true,
		Message: "Connected to Codex API successfully",
	}, nil
}
