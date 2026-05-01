package speculative

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChatResponse represents a chat completion response (minimal for speculative package)
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Message  Message    `json:"message"`
	LogProbs *LogProbs  `json:"logprobs,omitempty"`
}

// Message represents a message
type Message struct {
	Content string `json:"content"`
}

// LogProbs represents log probabilities
type LogProbs struct {
	Content []TokenLogProb `json:"content"`
}

// TokenLogProb represents a token's log probability
type TokenLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
}

// APIClientIface represents the API client interface for speculative decoding
type APIClientIface interface {
	ChatCompletion(req interface{}) (interface{}, error)
}

// SpeculativeDecoder implements client-side speculative decoding
type SpeculativeDecoder struct {
	Client         APIClientIface
	DraftModel     string
	TargetModel    string
	MaxDraftTokens int
	Enabled        bool
	CachingEnabled bool
	CacheID        string
}

// NewSpeculativeDecoder creates a new speculative decoder
func NewSpeculativeDecoder(client APIClientIface, draftModel, targetModel string, maxDraftTokens int) *SpeculativeDecoder {
	return &SpeculativeDecoder{
		Client:         client,
		DraftModel:     draftModel,
		TargetModel:    targetModel,
		MaxDraftTokens: maxDraftTokens,
		Enabled:        true,
	}
}

// Draft generates draft tokens using the draft model
func (s *SpeculativeDecoder) Draft(prompt string) ([]string, error) {
	if !s.Enabled {
		return nil, fmt.Errorf("speculative decoding is disabled")
	}
	
	// Build request for draft model
	req := map[string]interface{}{
		"model": s.DraftModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": s.MaxDraftTokens,
		"logprobs": true,
		"top_logprobs": 1,
	}
	
	if s.CachingEnabled && s.CacheID != "" {
		req["cache_id"] = s.CacheID
	}
	
	resp, err := s.Client.ChatCompletion(req)
	if err != nil {
		return nil, fmt.Errorf("draft model error: %w", err)
	}
	
	chatResp, ok := resp.(*ChatResponse)
	if !ok || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("invalid draft response")
	}
	
	// Parse tokens from response
	content := chatResp.Choices[0].Message.Content
	tokens := tokenize(content)
	
	// Limit to max draft tokens
	if len(tokens) > s.MaxDraftTokens {
		tokens = tokens[:s.MaxDraftTokens]
	}
	
	return tokens, nil
}

// Verify verifies draft tokens against the target model
func (s *SpeculativeDecoder) Verify(prompt string, draftTokens []string) ([]string, error) {
	if !s.Enabled {
		return nil, fmt.Errorf("speculative decoding is disabled")
	}
	
	// Build request for target model with draft tokens
	draftText := joinTokens(draftTokens)
	req := map[string]interface{}{
		"model": s.TargetModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
			{"role": "assistant", "content": draftText},
		},
		"logprobs": true,
		"top_logprobs": 1,
	}
	
	if s.CachingEnabled && s.CacheID != "" {
		req["cache_id"] = s.CacheID
	}
	
	resp, err := s.Client.ChatCompletion(req)
	if err != nil {
		return nil, fmt.Errorf("target model error: %w", err)
	}
	
	chatResp, ok := resp.(*ChatResponse)
	if !ok || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("invalid target response")
	}
	
	// Get logprobs from target model
	var targetLogProbs []TokenLogProb
	if chatResp.Choices[0].LogProbs != nil {
		targetLogProbs = chatResp.Choices[0].LogProbs.Content
	}
	
	// Accept tokens that match
	accepted := s.acceptTokens(draftTokens, targetLogProbs)
	
	// Update cache ID if provided
	if s.CachingEnabled {
		// In real implementation, extract cache ID from response headers
		s.CacheID = "cache-" + fmt.Sprint(len(prompt) + len(draftText))
	}
	
	return accepted, nil
}

// Decode performs the full speculative decoding flow
func (s *SpeculativeDecoder) Decode(prompt string) (string, error) {
	if !s.Enabled {
		return "", fmt.Errorf("speculative decoding is disabled")
	}
	// 1. Draft tokens
	draftTokens, err := s.Draft(prompt)
	if err != nil {
		return "", err
	}
	
	// 2. Verify tokens
	accepted, err := s.Verify(prompt, draftTokens)
	if err != nil {
		return "", err
	}
	
	// 3. Return accepted tokens joined
	return joinTokens(accepted), nil
}

// EnableCaching enables KV cache reuse via DeepSeek's context caching
func (s *SpeculativeDecoder) EnableCaching(enable bool) {
	s.CachingEnabled = enable
}

// SetCacheID sets the cache ID for KV cache reuse
func (s *SpeculativeDecoder) SetCacheID(id string) {
	s.CacheID = id
}

// IsEnabled returns whether speculative decoding is enabled
func (s *SpeculativeDecoder) IsEnabled() bool {
	return s.Enabled
}

// SetEnabled enables or disables speculative decoding
func (s *SpeculativeDecoder) SetEnabled(enabled bool) {
	s.Enabled = enabled
}

// acceptTokens determines which draft tokens are accepted based on logprobs
func (s *SpeculativeDecoder) acceptTokens(draftTokens []string, targetLogProbs []TokenLogProb) []string {
	accepted := make([]string, 0, len(draftTokens))
	for i, draft := range draftTokens {
		if i >= len(targetLogProbs) {
			break
		}
		// Accept if token matches or logprob is high enough
		if draft == targetLogProbs[i].Token {
			accepted = append(accepted, draft)
		} else {
			// Stop on first mismatch
			break
		}
	}
	return accepted
}

// joinTokens joins tokens into a string
func joinTokens(tokens []string) string {
	return strings.Join(tokens, " ")
}

// tokenize is a simple tokenizer (in production, use proper tokenization)
func tokenize(text string) []string {
	// Simple whitespace tokenization for demo
	// Real implementation would use the model's tokenizer
	return strings.Fields(text)
}

// buildChatRequest builds a ChatRequest for the API
func buildChatRequest(model, prompt string, maxTokens int) (interface{}, error) {
	req := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if maxTokens > 0 {
		req["max_tokens"] = maxTokens
	}
	return req, nil
}

// parseResponse parses a JSON response into ChatResponse
func parseResponse(data []byte) (*ChatResponse, error) {
	var resp ChatResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
