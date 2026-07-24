package ai

import (
	"errors"
	"net/http"

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

func (h *Handler) ChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.val.Struct(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Extract Bearer token from Authorization header
	token := ""
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}

	upstreamReq := &ProxyRequest{
		BaseURL:     "https://opencode.ai/zen/v1",
		APIKey: token,
		Model:       req.Model,
		Messages:    toInternalMessages(req.Messages),
		Temperature: 0,
		MaxTokens:   0,
	}
	if req.Temperature != nil {
		upstreamReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		upstreamReq.MaxTokens = *req.MaxTokens
	}
	result, err := h.svc.Proxy(c.Request.Context(), upstreamReq)
	if err != nil {
		var ssrfErr *SSRFError
		if errors.As(err, &ssrfErr) {
			common.Fail(c, http.StatusBadRequest, "SSRF_BLOCKED", err.Error())
			return
		}
		common.Fail(c, http.StatusBadGateway, "PROXY_ERROR", err.Error())
		return
	}
	openAIResp := toOpenAIResponse(req.Model, result)
	c.JSON(http.StatusOK, openAIResp)
}

func (h *Handler) ListModels(c *gin.Context) {
	models := []ModelInfo{
		{ID: "deepseek-v4-flash-free", Object: "model", Created: 1735689600, OwnedBy: "opencode"},
		{ID: "deepseek-v4-flash", Object: "model", Created: 1735689600, OwnedBy: "opencode"},
	}
	resp := ModelsResponse{
		Object: "list",
		Data:   models,
	}
	c.JSON(http.StatusOK, resp)
}

func toInternalMessages(msgs []ChatMessage) []Message {
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		out[i] = Message{Role: m.Role, Content: m.Content}
	}
	return out
}

func toOpenAIResponse(model string, resp *ProxyResponse) *ChatCompletionResponse {
	return &ChatCompletionResponse{
		ID:      "chatcmpl-" + resp.Upstream,
		Object:  "chat.completion",
		Created: 0,
		Model:   model,
		Choices: []ChatCompletionChoice{
			{
				Index:        0,
				Message:      ChatMessage{Role: "assistant", Content: resp.Content},
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