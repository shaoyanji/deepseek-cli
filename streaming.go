package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// streamChatCompletion handles streaming chat completions
func (c *Client) streamChatCompletion(req *ChatRequest) error {
	var buf io.Reader
	js, err := json.Marshal(req)
	if err != nil {
		return err
	}
	buf = bytes.NewReader(js)

	url := c.Base + "/chat/completions"
	httpReq, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return c.parseSSEStream(resp.Body, "chat")
}

// streamFIMCompletion handles streaming FIM completions
func (c *Client) streamFIMCompletion(req *FIMRequest) error {
	var buf io.Reader
	js, err := json.Marshal(req)
	if err != nil {
		return err
	}
	buf = bytes.NewReader(js)

	url := c.Base + "/completions"
	httpReq, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return c.parseSSEStream(resp.Body, "fim")
}

// parseSSEStream parses Server-Sent Events and outputs them
func (c *Client) parseSSEStream(body io.Reader, completionType string) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if line == "" {
			continue
		}

		// SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for termination signal
		if data == "[DONE]" {
			break
		}

		// Parse JSON data
		if completionType == "chat" {
			var chunk ChatResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Try parsing as a delta chunk
				var deltaChunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
						FinishReason *string `json:"finish_reason"`
					} `json:"choices"`
					Usage *Usage `json:"usage"`
				}
				if err := json.Unmarshal([]byte(data), &deltaChunk); err != nil {
					continue
				}
				
				// Output content
				for _, choice := range deltaChunk.Choices {
					if choice.Delta.Content != "" {
						fmt.Print(choice.Delta.Content)
					}
					if choice.FinishReason != nil {
						fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
					}
				}
				
				// Output usage if present
				if deltaChunk.Usage != nil {
					fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
						deltaChunk.Usage.PromptTokens,
						deltaChunk.Usage.CompletionTokens,
						deltaChunk.Usage.TotalTokens)
				}
			} else {
				// Output content from parsed chunk
				for _, choice := range chunk.Choices {
					if choice.Delta != nil && choice.Delta.Content != "" {
						fmt.Print(choice.Delta.Content)
					}
					if choice.FinishReason != nil {
						fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
					}
				}
				
				// Output usage if present
				if chunk.Usage != nil {
					fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
						chunk.Usage.PromptTokens,
						chunk.Usage.CompletionTokens,
						chunk.Usage.TotalTokens)
				}
			}
		} else if completionType == "fim" {
			var chunk FIMResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Try parsing as a simpler chunk
				var simpleChunk struct {
					Choices []struct {
						Text         string  `json:"text"`
						FinishReason *string `json:"finish_reason"`
					} `json:"choices"`
					Usage *Usage `json:"usage"`
				}
				if err := json.Unmarshal([]byte(data), &simpleChunk); err != nil {
					continue
				}
				
				// Output text
				for _, choice := range simpleChunk.Choices {
					if choice.Text != "" {
						fmt.Print(choice.Text)
					}
					if choice.FinishReason != nil {
						fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
					}
				}
				
				// Output usage if present
				if simpleChunk.Usage != nil {
					fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
						simpleChunk.Usage.PromptTokens,
						simpleChunk.Usage.CompletionTokens,
						simpleChunk.Usage.TotalTokens)
				}
			} else {
				// Output text from parsed chunk
				for _, choice := range chunk.Choices {
					if choice.Text != "" {
						fmt.Print(choice.Text)
					}
					if choice.FinishReason != nil {
						fmt.Printf("\n[finish_reason: %s]\n", *choice.FinishReason)
					}
				}
				
				// Output usage if present
				if chunk.Usage != nil {
					fmt.Printf("\n[usage: prompt_tokens=%d, completion_tokens=%d, total_tokens=%d]\n",
						chunk.Usage.PromptTokens,
						chunk.Usage.CompletionTokens,
						chunk.Usage.TotalTokens)
				}
			}
		}
	}

	// Ensure final newline
	fmt.Println()

	return scanner.Err()
}
