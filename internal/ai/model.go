package ai

// ProxyRequest represents the incoming request to the AI proxy endpoint.
type ProxyRequest struct {
	BaseURL     string         `json:"base_url"     validate:"required,url"`
	APIKey      string         `json:"api_key"      validate:"required"`
	Model       string         `json:"model"        validate:"required"`
	Messages    []Message      `json:"messages"     validate:"required,min=1"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
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
