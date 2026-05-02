package speculative

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIClient mocks the API client for testing
type MockAPIClient struct {
	mock.Mock
	Responses         []map[string]interface{}
	Err               error
	CallCount         int
	ChatCompletionFunc func(req interface{}) (interface{}, error)
}

func (m *MockAPIClient) ChatCompletion(req interface{}) (interface{}, error) {
	if m.ChatCompletionFunc != nil {
		return m.ChatCompletionFunc(req)
	}
	if m.Err != nil {
		return nil, m.Err
	}
	if m.CallCount >= len(m.Responses) {
		return nil, fmt.Errorf("no more mock responses")
	}
	respMap := m.Responses[m.CallCount]
	m.CallCount++
	
	// Convert map to ChatResponse struct
	resp := &ChatResponse{}
	if choices, ok := respMap["choices"].([]map[string]interface{}); ok {
		for _, c := range choices {
			choice := Choice{}
			if msg, ok := c["message"].(map[string]string); ok {
				choice.Message = Message{Content: msg["content"]}
			}
			if logProbs, ok := c["logprobs"].(map[string]interface{}); ok {
				if content, ok := logProbs["content"].([]map[string]interface{}); ok {
					choice.LogProbs = &LogProbs{}
					for _, token := range content {
						if tok, ok := token["token"].(string); ok {
							if logprob, ok := token["logprob"].(float64); ok {
								choice.LogProbs.Content = append(choice.LogProbs.Content, TokenLogProb{
									Token:   tok,
									LogProb: logprob,
								})
							}
						}
					}
				}
			}
			resp.Choices = append(resp.Choices, choice)
		}
	}
	return resp, nil
}

func TestNewSpeculativeDecoder(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 5)
	
	assert.NotNil(t, decoder)
	assert.Equal(t, "deepseek-v4-flash", decoder.DraftModel)
	assert.Equal(t, "deepseek-v4-pro", decoder.TargetModel)
	assert.Equal(t, 5, decoder.MaxDraftTokens)
}

func TestDraftTokens(t *testing.T) {
	client := &MockAPIClient{
		Responses: []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{"content": "token1 token2 token3"},
					},
				},
			},
		},
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	tokens, err := decoder.Draft("test prompt")
	
	assert.NoError(t, err)
	assert.Len(t, tokens, 3)
}

func TestVerifyTokensAllAccepted(t *testing.T) {
	client := &MockAPIClient{
		Responses: []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{"content": "token1 token2 token3"},
						"logprobs": map[string]interface{}{
							"content": []map[string]interface{}{
								{"token": "token1", "logprob": -0.1},
								{"token": "token2", "logprob": -0.2},
								{"token": "token3", "logprob": -0.15},
							},
						},
					},
				},
			},
		},
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	draftTokens := []string{"token1", "token2", "token3"}
	
	accepted, err := decoder.Verify("test prompt", draftTokens)
	assert.NoError(t, err)
	assert.Len(t, accepted, 3)
}

func TestVerifyTokensPartialAccept(t *testing.T) {
	client := &MockAPIClient{
		Responses: []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{"content": "token1 token2"},
						"logprobs": map[string]interface{}{
							"content": []map[string]interface{}{
								{"token": "token1", "logprob": -0.1},
								{"token": "token2", "logprob": -0.2},
							},
						},
					},
				},
			},
		},
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	draftTokens := []string{"token1", "token2", "token3"}
	
	accepted, err := decoder.Verify("test prompt", draftTokens)
	assert.NoError(t, err)
	assert.Len(t, accepted, 2)
}

func TestSpeculativeDecodingFlow(t *testing.T) {
	client := &MockAPIClient{
		Responses: []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{"message": map[string]string{"content": "hello world foo"}},
				},
			},
			{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{"content": "hello world foo bar"},
						"logprobs": map[string]interface{}{
							"content": []map[string]interface{}{
								{"token": "hello", "logprob": -0.1},
								{"token": "world", "logprob": -0.2},
								{"token": "foo", "logprob": -0.15},
							},
						},
					},
				},
			},
		},
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 5)
	result, err := decoder.Decode("test prompt")
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestKVReuse(t *testing.T) {
	client := &MockAPIClient{}
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	
	// Enable context caching
	decoder.EnableCaching(true)
	assert.True(t, decoder.CachingEnabled)
	
	// Test that cache ID is tracked
	decoder.SetCacheID("test-cache-id")
	assert.Equal(t, "test-cache-id", decoder.CacheID)
}

func TestConfigFlags(t *testing.T) {
	client := &MockAPIClient{}
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 5)
	
	// Test default values
	assert.True(t, decoder.IsEnabled())
	assert.Equal(t, 5, decoder.MaxDraftTokens)
	
	// Disable
	decoder.SetEnabled(false)
	assert.False(t, decoder.IsEnabled())
}

func TestEmptyDraft(t *testing.T) {
	client := &MockAPIClient{
		Responses: []map[string]interface{}{
			{
				"choices": []map[string]interface{}{
					{"message": map[string]string{"content": ""}},
				},
			},
		},
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	tokens, err := decoder.Draft("test prompt")
	
	assert.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestAPIError(t *testing.T) {
	client := &MockAPIClient{
		Err: fmt.Errorf("API error"),
	}
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	_, err := decoder.Draft("test prompt")
	
	assert.Error(t, err)
}

func TestLogProbComparison(t *testing.T) {
	decoder := NewSpeculativeDecoder(nil, "draft", "target", 3)
	
	// Test acceptance logic based on logprobs
	draftTokens := []string{"a", "b", "c"}
	targetLogProbs := []TokenLogProb{
		{Token: "a", LogProb: -0.5},
		{Token: "b", LogProb: -0.8},
		{Token: "x", LogProb: -0.3}, // Different token
	}
	
	accepted := decoder.acceptTokens(draftTokens, targetLogProbs)
	assert.Len(t, accepted, 2) // Only "a" and "b" accepted
}

func TestBuildChatRequest(t *testing.T) {
	// Test buildChatRequest with max tokens
	req, err := buildChatRequest("deepseek-chat", "test prompt", 100)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	
	reqMap, ok := req.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "deepseek-chat", reqMap["model"])
	assert.Equal(t, 100, reqMap["max_tokens"])
	
	// Test buildChatRequest without max tokens
	req, err = buildChatRequest("deepseek-chat", "test prompt", 0)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	
	reqMap, ok = req.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "deepseek-chat", reqMap["model"])
	_, hasMaxTokens := reqMap["max_tokens"]
	assert.False(t, hasMaxTokens)
	
	// Test with messages
	req, err = buildChatRequest("deepseek-chat", "hello world", 50)
	assert.NoError(t, err)
	reqMap, ok = req.(map[string]interface{})
	assert.True(t, ok)
	
	messages, ok := reqMap["messages"].([]map[string]string)
	assert.True(t, ok)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0]["role"])
	assert.Equal(t, "hello world", messages[0]["content"])
}

func TestParseResponse(t *testing.T) {
	// Test parseResponse with valid JSON
	jsonData := []byte(`{
		"choices": [{
			"message": {
				"content": "test response"
			}
		}]
	}`)
	
	resp, err := parseResponse(jsonData)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "test response", resp.Choices[0].Message.Content)
	
	// Test parseResponse with invalid JSON
	invalidJSON := []byte(`{invalid json}`)
	resp, err = parseResponse(invalidJSON)
	assert.Error(t, err)
	assert.Nil(t, resp)
	
	// Test parseResponse with empty JSON
	emptyJSON := []byte(`{}`)
	resp, err = parseResponse(emptyJSON)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Choices, 0)
}

func TestTokenize(t *testing.T) {
	// Test tokenize function
	tokens := tokenize("hello world")
	assert.Equal(t, []string{"hello", "world"}, tokens)
	
	// Test with single word
	tokens = tokenize("hello")
	assert.Equal(t, []string{"hello"}, tokens)
	
	// Test with empty string
	tokens = tokenize("")
	assert.Equal(t, []string{}, tokens)
	
	// Test with multiple spaces
	tokens = tokenize("hello  world  test")
	assert.Equal(t, []string{"hello", "world", "test"}, tokens)
}
