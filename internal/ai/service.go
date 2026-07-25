package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Service handles AI proxy requests.
type Service struct {
	client *http.Client
}

// NewService creates a new AI proxy service.
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// ProxyStream forwards a streaming request to an OpenAI-compatible provider
// and returns the response body as an io.ReadCloser for SSE streaming.
func (s *Service) ProxyStream(ctx context.Context, req *ProxyRequest) (io.ReadCloser, error) {
	if err := validateSSRF(req.BaseURL); err != nil {
		return nil, &SSRFError{Message: err.Error()}
	}

	upstreamReq := buildUpstreamRequest(req)
	upstreamReq["stream"] = true

	body, err := json.Marshal(upstreamReq)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream request: %w", err)
	}

	chatURL := strings.TrimRight(req.BaseURL, "/") + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Use a dedicated client without timeout for streaming
	streamClient := &http.Client{
		Transport: s.client.Transport,
		Timeout:   0,
	}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(respBody))
	}

	return resp.Body, nil
}

// ProbeProxy forwards a GET request to the upstream provider's discovery endpoint.
// It maps proxy request paths to the correct upstream paths.
func (s *Service) ProbeProxy(ctx context.Context, upstreamBase, path string, apiKey string) (int, []byte, error) {
	// upstreamBase is "https://opencode.ai/zen/v1" (includes /v1)
	// Strip /v1 to get the raw root for non-v1 paths
	upstreamRoot := strings.TrimSuffix(upstreamBase, "/v1")
	if upstreamRoot == "" {
		upstreamRoot = upstreamBase
	}

	// Map proxy paths to upstream paths
	upstreamPath := ""
	switch {
	case path == "/v1/models" || path == "/api/v1/models" || path == "/models":
		upstreamPath = upstreamBase + "/models"
	case path == "/v1/props" || path == "/props":
		upstreamPath = upstreamBase + "/props"
	case path == "/api/tags":
		upstreamPath = upstreamRoot + "/api/tags"
	case path == "/version":
		upstreamPath = upstreamRoot + "/version"
	default:
		return 0, nil, fmt.Errorf("discovery path not allowed: %s", path)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamPath, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("create upstream request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("read upstream response: %w", err)
	}

	return resp.StatusCode, body, nil
}

// Proxy forwards a request to an OpenAI-compatible provider.
func (s *Service) Proxy(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	if err := validateSSRF(req.BaseURL); err != nil {
		return nil, &SSRFError{Message: err.Error()}
	}

	// Build the upstream request body
	upstreamReq := buildUpstreamRequest(req)

	body, err := json.Marshal(upstreamReq)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream request: %w", err)
	}

	chatURL := strings.TrimRight(req.BaseURL, "/") + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("read upstream response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(respBody))
	}

	result, err := parseUpstreamResponse(respBody, req.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse upstream response: %w", err)
	}

	return result, nil
}

// buildUpstreamRequest constructs the OpenAI-compatible request body.
func buildUpstreamRequest(req *ProxyRequest) map[string]interface{} {
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Stream {
		body["stream"] = true
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	return body
}

// parseUpstreamResponse parses the OpenAI-compatible response.
func parseUpstreamResponse(body []byte, upstream string) (*ProxyResponse, error) {
	var upstreamResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int `json:"index"`
			Message     struct {
				Role             string      `json:"role"`
				Content          string      `json:"content"`
				ReasoningContent string      `json:"reasoning_content"`
				ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &upstreamResp); err != nil {
		return nil, fmt.Errorf("unmarshal upstream: %w", err)
	}

	content := ""
	finishReason := ""
	var toolCalls []ToolCall

	if len(upstreamResp.Choices) > 0 {
		msg := upstreamResp.Choices[0].Message
		content = msg.Content
		finishReason = upstreamResp.Choices[0].FinishReason
		toolCalls = msg.ToolCalls

		// If content is empty, use reasoning_content (reasoning models)
		if content == "" && msg.ReasoningContent != "" {
			content = msg.ReasoningContent
		}
	}

	return &ProxyResponse{
		Upstream:     upstream,
		Content:      content,
		FinishReason: finishReason,
		Model:        upstreamResp.Model,
		PromptTokens: upstreamResp.Usage.PromptTokens,
		OutputTokens: upstreamResp.Usage.CompletionTokens,
		TotalTokens:  upstreamResp.Usage.TotalTokens,
		ToolCalls:    toolCalls,
	}, nil
}

var blockListPrefixes = []string{
	"10.", "172.16.", "172.17.", "172.18.", "172.19.",
	"172.20.", "172.21.", "172.22.", "172.23.",
	"172.24.", "172.25.", "172.26.", "172.27.",
	"172.28.", "172.29.", "172.30.", "172.31.",
	"192.168.", "169.254.", "0.", "127.",
	"100.64.", "100.65.", "100.66.", "100.67.",
	"100.68.", "100.69.", "100.70.", "100.71.",
	"100.72.", "100.73.", "100.74.", "100.75.",
	"100.76.", "100.77.", "100.78.", "100.79.",
	"100.80.", "100.81.", "100.82.", "100.83.",
	"100.84.", "100.85.", "100.86.", "100.87.",
	"100.88.", "100.89.", "100.90.", "100.91.",
	"100.92.", "100.93.", "100.94.", "100.95.",
	"100.96.", "100.97.", "100.98.", "100.99.",
	"100.100.", "100.101.", "100.102.", "100.103.",
	"100.104.", "100.105.", "100.106.", "100.107.",
	"100.108.", "100.109.", "100.110.", "100.111.",
	"100.112.", "100.113.", "100.114.", "100.115.",
	"100.116.", "100.117.", "100.118.", "100.119.",
	"100.120.", "100.121.", "100.122.", "100.123.",
	"100.124.", "100.125.", "100.126.", "100.127.",
}

var blockListExact = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
	"0.0.0.0":   true,
}

// isPrivateIP checks if an IP string is in a private/reserved range.
func isPrivateIP(ipStr string) bool {
	if blockListExact[ipStr] {
		return true
	}
	for _, p := range blockListPrefixes {
		if strings.HasPrefix(ipStr, p) {
			return true
		}
	}
	return false
}

// validateSSRF checks that the URL is safe to proxy to.
func validateSSRF(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Must be HTTPS
	if u.Scheme != "https" {
		return fmt.Errorf("only https is allowed, got %s", u.Scheme)
	}

	// Block empty hosts
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty host")
	}

	// Strip port for IP check
	host = strings.TrimSuffix(host, ":"+u.Port())
	if u.Port() != "" {
		host = strings.TrimSuffix(host, ":"+u.Port())
	}

	// Check explicit block list
	if blockListExact[strings.ToLower(host)] {
		return fmt.Errorf("blocked host: %s", host)
	}

	// Resolve the hostname to IPs
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("blocked private/reserved IP: %s (resolved from %s)", ip, host)
		}
	}

	return nil
}
