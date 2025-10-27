package aura

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type Client struct {
	APIKey  string
	BaseURL string
	Version string
}

type VerifyRequest struct {
	AgentID        string          `json:"agent_id"`
	RequestContext json.RawMessage `json:"request_context"`
}

type VerifyResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func NewClient(apiKey, baseURL, version string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Client{APIKey: apiKey, BaseURL: baseURL, Version: version}
}

func (c *Client) Verify(agentID string, requestContext any) (*VerifyResponse, error) {
	b, err := json.Marshal(requestContext)
	if err != nil {
		return nil, err
	}
	reqBody, _ := json.Marshal(VerifyRequest{AgentID: agentID, RequestContext: b})
	req, _ := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/verify", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)
	if c.Version != "" {
		req.Header.Set("AURA-Version", c.Version)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("verify failed: status " + resp.Status)
	}
	var out VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
