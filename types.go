package main

// Chat completion request structures
type ChatRequest struct {
	Model            string                 `json:"model"`
	Messages         []Message              `json:"messages"`
	Thinking         *ThinkingConfig        `json:"thinking,omitempty"`
	ReasoningEffort  string                 `json:"reasoning_effort,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"` // string or []string
	Stream           bool                   `json:"stream,omitempty"`
	StreamOptions    *StreamOptions         `json:"stream_options,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"` // string or ToolChoiceFunction
	Logprobs         *bool                  `json:"logprobs,omitempty"`
	TopLogprobs      *int                   `json:"top_logprobs,omitempty"`
}

type Message struct {
	Role            string      `json:"role"`
	Content         interface{} `json:"content"` // string or null
	Name            *string     `json:"name,omitempty"`
	Prefix          *bool       `json:"prefix,omitempty"`           // For beta prefix completion
	ReasoningContent *string    `json:"reasoning_content,omitempty"` // For thinking mode
	ToolCallID      *string     `json:"tool_call_id,omitempty"`    // For tool responses
}

type ThinkingConfig struct {
	Type string `json:"type"` // "enabled" or "disabled"
}

type ResponseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolChoiceFunction struct {
	Type     string `json:"type"` // "function"
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// Chat completion response structures
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	Delta        *Message     `json:"delta,omitempty"`        // For streaming
	FinishReason *string      `json:"finish_reason,omitempty"` // For streaming
	Logprobs     *Logprobs    `json:"logprobs,omitempty"`
}

type Logprobs struct {
	Content []TokenLogprob `json:"content,omitempty"`
}

type TokenLogprob struct {
	Token   string   `json:"token"`
	Logprob float64  `json:"logprob"`
	TopLogprobs []TopLogprob `json:"top_logprobs,omitempty"`
}

type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// FIM completion request structures
type FIMRequest struct {
	Model            string          `json:"model"`
	Prompt           string          `json:"prompt"`
	Suffix           *string         `json:"suffix,omitempty"`
	MaxTokens        *int            `json:"max_tokens,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	FrequencyPenalty *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64        `json:"presence_penalty,omitempty"`
	Stop             interface{}     `json:"stop,omitempty"` // string or []string
	Stream           bool            `json:"stream,omitempty"`
	StreamOptions    *StreamOptions  `json:"stream_options,omitempty"`
	Echo             bool            `json:"echo,omitempty"`
	Logprobs         *int            `json:"logprobs,omitempty"`
}

// FIM completion response structures
type FIMResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []FIMChoice `json:"choices"`
	Usage   *Usage    `json:"usage,omitempty"`
}

type FIMChoice struct {
	Index        int      `json:"index"`
	Text         string   `json:"text"`
	FinishReason *string  `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
}

// Models response structures
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Balance response structures
type BalanceResponse struct {
	Balance         float64 `json:"balance"`
	TotalBalance    float64 `json:"total_balance"`
	AvailableBalance float64 `json:"available_balance"`
	GrantedBalance  float64 `json:"granted_balance"`
}
