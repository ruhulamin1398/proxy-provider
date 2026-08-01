package ai

// DefaultUpstreamURL is the default OpenAI-compatible provider endpoint.
const DefaultUpstreamURL = "https://opencode.ai/zen/v1"

// ─── Existing proxy types ───

// ProxyRequest represents the incoming request to the AI proxy endpoint.
type ProxyRequest struct {
	BaseURL     string    `json:"base_url"     validate:"required,url"`
	APIKey      string    `json:"api_key,omitempty"`
	Model       string    `json:"model"        validate:"required"`
	Messages    []Message `json:"messages"     validate:"required,min=1"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []ToolDef `json:"tools,omitempty"`
}

// Message represents a chat message in the OpenAI-compatible format.
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// ProxyResponse wraps the upstream response.
type ProxyResponse struct {
	Upstream         string     `json:"upstream"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	FinishReason     string     `json:"finish_reason"`
	Model            string     `json:"model"`
	PromptTokens     int        `json:"prompt_tokens"`
	OutputTokens     int        `json:"output_tokens"`
	TotalTokens      int        `json:"total_tokens"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ─── OpenAI-compatible types (for /v1/chat/completions) ───

// ChatCompletionRequest represents an OpenAI-compatible chat completion request.
type ChatCompletionRequest struct {
	Model       string        `json:"model"       validate:"required"`
	Messages    []ChatMessage `json:"messages"    validate:"required,min=1"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Tools       []ToolDef     `json:"tools,omitempty"`
}

// ToolDef defines a tool that the model can call.
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a callable function for tool use.
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatMessage represents a message in OpenAI format.
type ChatMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool/function call from the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction represents the function details in a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse is the response for GET /v1/models.
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}
