package ai

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/ruhulamin1398/ai-backend/internal/common"
)

type Handler struct {
	svc *Service
	val *validator.Validate
}

func NewHandler(svc *Service, val *validator.Validate) *Handler {
	return &Handler{svc: svc, val: val}
}

func (h *Handler) Proxy(c *gin.Context) {
	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.val.Struct(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.svc.Proxy(c.Request.Context(), &req)
	if err != nil {
		var ssrfErr *SSRFError
		if errors.As(err, &ssrfErr) {
			common.Fail(c, http.StatusBadRequest, "SSRF_BLOCKED", err.Error())
			return
		}
		common.Fail(c, http.StatusBadGateway, "PROXY_ERROR", err.Error())
		return
	}
	common.Success(c, http.StatusOK, result)
}

func openAIError(c *gin.Context, httpStatus int, message, errType, code string) {
	c.JSON(httpStatus, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
			"param":   nil,
			"code":    code,
		},
	})
}

func (h *Handler) ChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		openAIError(c, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_request")
		return
	}
	if err := h.val.Struct(&req); err != nil {
		openAIError(c, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_request")
		return
	}

	// Extract Bearer token from Authorization header
	token := ""
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}

	upstreamReq := &ProxyRequest{
		BaseURL:     DefaultUpstreamURL,
		APIKey:      token,
		Model:       req.Model,
		Messages:    toInternalMessages(req.Messages),
		Temperature: 0,
		MaxTokens:   0,
		Stream:      req.Stream,
	}
	if req.Temperature != nil {
		upstreamReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		upstreamReq.MaxTokens = *req.MaxTokens
	}
	// Default max_tokens for reasoning models (only when not explicitly set)
	if req.MaxTokens == nil {
		upstreamReq.MaxTokens = 8192
	} else {
		// Pad small max_tokens for reasoning models — thinking consumes tokens too
		if *req.MaxTokens < 16384 {
			padded := *req.MaxTokens * 2
			upstreamReq.MaxTokens = padded
		}
	}

	reqSummary, _ := json.Marshal(map[string]interface{}{
		"model":    upstreamReq.Model,
		"messages": upstreamReq.Messages,
	})

	// ── Streaming path ──
	if req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		bodyReader, err := h.svc.ProxyStream(c.Request.Context(), upstreamReq)
		if err != nil {
			// Log downstream failure
			common.WriteDownstreamLog(&common.DownstreamEntry{
				Timestamp: time.Now().Format(time.RFC3339),
				Model:     upstreamReq.Model,
				URL:       upstreamReq.BaseURL + "/chat/completions",
				ReqBody:   string(reqSummary),
				Status:    "error",
				Error:     err.Error(),
			})
			openAIError(c, http.StatusBadGateway, err.Error(), "server_error", "upstream_error")
			return
		}
		defer bodyReader.Close()

		// Log downstream start
		common.WriteDownstreamLog(&common.DownstreamEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Model:     upstreamReq.Model,
			URL:       upstreamReq.BaseURL + "/chat/completions",
			ReqBody:   string(reqSummary),
			Status:    "streaming",
		})

		// Stream SSE events from upstream to client
		flusher, _ := c.Writer.(http.Flusher)
		scanner := bufio.NewScanner(bodyReader)
		scanner.Buffer(make([]byte, 64*1024), 256*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if _, err := fmt.Fprintf(c.Writer, "%s\n", line); err != nil {
				break
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		return
	}

	// ── Non-streaming path ──
	start := time.Now()
	result, err := h.svc.Proxy(c.Request.Context(), upstreamReq)

	if err != nil {
		// Log downstream failure
		common.WriteDownstreamLog(&common.DownstreamEntry{
			Timestamp: start.Format(time.RFC3339),
			Model:     upstreamReq.Model,
			URL:       upstreamReq.BaseURL + "/chat/completions",
			ReqBody:   string(reqSummary),
			Status:    "error",
			Error:     err.Error(),
		})

		var ssrfErr *SSRFError
		if errors.As(err, &ssrfErr) {
			openAIError(c, http.StatusBadRequest, err.Error(), "invalid_request_error", "ssrf_blocked")
			return
		}
		openAIError(c, http.StatusBadGateway, err.Error(), "server_error", "upstream_error")
		return
	}

	// Log downstream success
	common.WriteDownstreamLog(&common.DownstreamEntry{
		Timestamp:    start.Format(time.RFC3339),
		Model:        upstreamReq.Model,
		URL:          upstreamReq.BaseURL + "/chat/completions",
		ReqBody:      string(reqSummary),
		RespBody:     result.Content,
		Status:       "success",
		PromptTokens: result.PromptTokens,
		OutputTokens: result.OutputTokens,
		TotalTokens:  result.TotalTokens,
	})

	openAIResp := toOpenAIResponse(req.Model, result)
	c.JSON(http.StatusOK, openAIResp)
}

// DiscoveryProxy forwards a discovery/probe request to the upstream provider.
// It extracts the API key from the Authorization header and proxies the path.
func (h *Handler) DiscoveryProxy(c *gin.Context) {
	// Extract API key from Authorization header
	token := ""
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}

	path := c.Request.URL.Path

	status, body, err := h.svc.ProbeProxy(c.Request.Context(), DefaultUpstreamURL, path, token)
	if err != nil || status != http.StatusOK {
		// If upstream fails or returns non-200, return a sensible fallback
		// so Hermes continues working
		isModelsPath := path == "/v1/models" || path == "/api/v1/models" || path == "/models"
		isPropsPath := path == "/api/tags" || path == "/v1/props" || path == "/props"
		isVersionPath := path == "/version"

		switch {
		case isModelsPath:
			c.JSON(http.StatusOK, gin.H{"object": "list", "data": []interface{}{}})
		case isPropsPath:
			c.JSON(http.StatusOK, gin.H{"models": []interface{}{}})
		case isVersionPath:
			c.JSON(http.StatusOK, gin.H{
				"version":  "1.0.0",
				"platform": "opencode",
			})
		default:
			c.JSON(http.StatusOK, gin.H{})
		}
		return
	}

	// Forward the response as-is
	for k, v := range map[string]string{
		"Content-Type": "application/json",
	} {
		c.Header(k, v)
	}
	c.Data(status, "application/json", body)
}

func toInternalMessages(msgs []ChatMessage) []Message {
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		out[i] = Message{Role: m.Role, Content: m.Content}
	}
	return out
}

func toOpenAIResponse(model string, resp *ProxyResponse) *ChatCompletionResponse {
	msg := ChatMessage{Role: "assistant", Content: resp.Content}
	if len(resp.ToolCalls) > 0 {
		msg.ToolCalls = resp.ToolCalls
	}
	return &ChatCompletionResponse{
		ID:      "chatcmpl-" + model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: resp.FinishReason,
			},
		},
		Usage: &UsageInfo{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.OutputTokens,
			TotalTokens:      resp.TotalTokens,
		},
	}
}