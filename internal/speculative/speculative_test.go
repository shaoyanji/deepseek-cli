package speculative

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIClient mocks the API client for testing
type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) ChatCompletion(req interface{}) (interface{}, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
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
	client := new(MockAPIClient)
	client.On("ChatCompletion", mock.Anything).Return(&ChatResponse{
		Choices: []Choice{
			{Message: Message{Content: "token1 token2 token3"}},
		},
	}, nil)
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	tokens, err := decoder.Draft("test prompt")
	
	assert.NoError(t, err)
	assert.Len(t, tokens, 3)
	client.AssertExpectations(t)
}

func TestVerifyTokensAllAccepted(t *testing.T) {
	client := new(MockAPIClient)
	// Target model accepts all draft tokens
	client.On("ChatCompletion", mock.Anything).Return(&ChatResponse{
		Choices: []Choice{
			{
				Message: Message{Content: "token1 token2 token3"},
				LogProbs: &LogProbs{
					Content: []TokenLogProb{
						{Token: "token1", LogProb: -0.1},
						{Token: "token2", LogProb: -0.2},
						{Token: "token3", LogProb: -0.15},
					},
				},
			},
		},
	}, nil)
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	draftTokens := []string{"token1", "token2", "token3"}
	
	accepted, err := decoder.Verify("test prompt", draftTokens)
	assert.NoError(t, err)
	assert.Len(t, accepted, 3)
}

func TestVerifyTokensPartialAccept(t *testing.T) {
	client := new(MockAPIClient)
	// Target model only accepts first 2 tokens
	client.On("ChatCompletion", mock.Anything).Return(&ChatResponse{
		Choices: []Choice{
			{
				Message: Message{Content: "token1 token2"},
				LogProbs: &LogProbs{
					Content: []TokenLogProb{
						{Token: "token1", LogProb: -0.1},
						{Token: "token2", LogProb: -0.2},
					},
				},
			},
		},
	}, nil)
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	draftTokens := []string{"token1", "token2", "token3"}
	
	accepted, err := decoder.Verify("test prompt", draftTokens)
	assert.NoError(t, err)
	assert.Len(t, accepted, 2)
}

func TestSpeculativeDecodingFlow(t *testing.T) {
	client := new(MockAPIClient)
	
	// First call: draft model generates tokens
	client.On("ChatCompletion", mock.MatchedBy(func(req interface{}) bool {
		return true // Accept any request for simplicity
	})).Return(&ChatResponse{
		Choices: []Choice{
			{Message: Message{Content: "hello world foo"}},
		},
	}, nil).Once()
	
	// Second call: target model verifies
	client.On("ChatCompletion", mock.Anything).Return(&ChatResponse{
		Choices: []Choice{
			{
				Message: Message{Content: "hello world"},
				LogProbs: &LogProbs{
					Content: []TokenLogProb{
						{Token: "hello", LogProb: -0.1},
						{Token: "world", LogProb: -0.15},
					},
				},
			},
		},
	}, nil).Once()
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	result, err := decoder.Decode("test prompt")
	
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	client.AssertExpectations(t)
}

func TestKVReuse(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	
	// Enable context caching
	decoder.EnableCaching(true)
	assert.True(t, decoder.CachingEnabled)
	
	// Test that cache ID is tracked
	decoder.SetCacheID("test-cache-id")
	assert.Equal(t, "test-cache-id", decoder.CacheID)
}

func TestConfigFlags(t *testing.T) {
	client := new(MockAPIClient)
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 5)
	
	// Test default values
	assert.True(t, decoder.IsEnabled())
	assert.Equal(t, 5, decoder.MaxDraftTokens)
	
	// Disable
	decoder.SetEnabled(false)
	assert.False(t, decoder.IsEnabled())
}

func TestEmptyDraft(t *testing.T) {
	client := new(MockAPIClient)
	client.On("ChatCompletion", mock.Anything).Return(&ChatResponse{
		Choices: []Choice{
			{Message: Message{Content: ""}},
		},
	}, nil)
	
	decoder := NewSpeculativeDecoder(client, "deepseek-v4-flash", "deepseek-v4-pro", 3)
	tokens, err := decoder.Draft("test prompt")
	
	assert.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestAPIError(t *testing.T) {
	client := new(MockAPIClient)
	client.On("ChatCompletion", mock.Anything).Return(nil, assert.AnError)
	
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
