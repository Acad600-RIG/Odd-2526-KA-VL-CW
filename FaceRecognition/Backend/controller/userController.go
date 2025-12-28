package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kennethandrew67/go-backend/model"
)

// GetBearerToken authenticates against the Bluejack LAPI and returns the bearer token.
func GetBearerToken() (string, error) {
	baseURL := "https://bluejack.binus.ac.id/lapi/api/Account/LogOn"

	reqBody := model.LoginRequest{
		Username: "KA24-1",
		Password: "Darwin123",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal login request: %w", err)
	}

	// Create POST request with JSON body
	req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read login response: %w", err)
	}

	var loginResp model.LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("parse login response: %w", err)
	}

	if loginResp.AccessToken == "" {
		return "", fmt.Errorf("missing access token in response")
	}

	return loginResp.AccessToken, nil
}