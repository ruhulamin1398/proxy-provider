package ai

// ─── Existing proxy types ───

// ProxyRequest represents the incoming request to the AI proxy endpoint.
type ProxyRequest struct {
	BaseURL     string    `json:"base_url"     validate:"required,url"`
	APIKey      string    `json:"api_key"      validate:"required"`
	Model       string    `json:"model"        validate:"required"`
	Messages    []Message `json:"messages"     validate:"required,min=1"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message in the OpenAI-compatible format.
type Message struct {
	Role    string `json:"role"    validate:"required,oneof=system user assistant"`
	Content string `json:"content" validate:"required"`
}

// ProxyResponse wraps the upstream response.
type ProxyResponse struct {
	Upstream       string `json:"upstream"`
	Content        string `json:"content"`
	FinishReason   string `json:"finish_reason"`
	Model          string `json:"model"`
	PromptTokens   int    `json:"prompt_tokens"`
	OutputTokens   int    `json:"output_tokens"`
	TotalTokens    int    `json:"total_tokens"`
}

// ─── OpenAI-compatible types (for /v1/chat/completions) ───

// ChatCompletionRequest represents an OpenAI-compatible chat completion request.
type ChatCompletionRequest struct {
	Model       string        `json:"model"       validate:"required"`
	Messages    []ChatMessage `json:"messages"    validate:"required,min=1"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatMessage represents a message in OpenAI format.
type ChatMessage struct {
	Role    string `json:"role"    validate:"required"`
	Content string `json:"content" validate:"required"`
}

// ChatCompletionResponse represents an OpenAI-compatible response.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *UsageInfo             `json:"usage,omitempty"`
}

// ChatCompletionChoice represents a single choice in the response.
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageInfo holds token usage statistics.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ─── Models list type (for GET /v1/models) ───

// ModelInfo represents a model entry.
type ModelInfo struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Created int64 `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the response for GET /v1/models.
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}