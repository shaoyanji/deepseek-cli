// Package websearch provides web search functionality using DuckDuckGo.
package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Snippet     string `json:"snippet"`
}

// SearchResponse holds the complete search response
type SearchResponse struct {
	Query      string         `json:"query"`
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	SearchTime time.Duration  `json:"search_time"`
}

// Client is a web search client
type Client struct {
	httpClient *http.Client
	maxResults int
	timeout    time.Duration
}

// NewClient creates a new web search client
func NewClient(maxResults int, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxResults: maxResults,
		timeout:    timeout,
	}
}

// DefaultClient returns a client with default settings
func DefaultClient() *Client {
	return NewClient(10, 30*time.Second)
}

// Search performs a web search using DuckDuckGo HTML interface
func (c *Client) Search(ctx context.Context, query string) (*SearchResponse, error) {
	start := time.Now()
	
	// DuckDuckGo HTML search URL
	baseURL := "https://html.duckduckgo.com/html/"
	params := url.Values{}
	params.Set("q", query)
	
	reqURL := baseURL + "?" + params.Encode()
	
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing search: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	
	results := c.parseDuckDuckGoHTML(string(body), query)
	
	return &SearchResponse{
		Query:      query,
		Results:    results,
		TotalCount: len(results),
		SearchTime: time.Since(start),
	}, nil
}

// parseDuckDuckGoHTML parses DuckDuckGo's HTML search results
func (c *Client) parseDuckDuckGoHTML(html string, query string) []SearchResult {
	var results []SearchResult
	
	// Simple parsing - look for result blocks
	// DuckDuckGo HTML format uses <div class="result"> for each result
	lines := strings.Split(html, "\n")
	
	for i, line := range lines {
		if strings.Contains(line, `class="result"`) || strings.Contains(line, `class="results__line"`) {
			// Found a result block - extract title, URL, and snippet
			title := c.extractElement(lines, i, "a", "class=\"result__a\"")
			url := c.extractHref(lines, i)
			snippet := c.extractElement(lines, i, "a", "class=\"result__snippet\"")
			
			if title != "" && url != "" {
				results = append(results, SearchResult{
					Title:   c.cleanHTML(title),
					URL:     url,
					Snippet: c.cleanHTML(snippet),
				})
				
				if len(results) >= c.maxResults {
					break
				}
			}
		}
	}
	
	return results
}

// extractElement extracts text content from an HTML element
func (c *Client) extractElement(lines []string, startLine int, tag string, attrs string) string {
	combined := strings.Join(lines[startLine:], " ")
	
	// Find opening tag
	openTag := "<" + tag
	if attrs != "" {
		openTag += " " + attrs
	}
	
	startIdx := strings.Index(combined, openTag)
	if startIdx == -1 {
		return ""
	}
	
	// Find closing tag
	closeTag := "</" + tag + ">"
	endIdx := strings.Index(combined[startIdx:], closeTag)
	if endIdx == -1 {
		return ""
	}
	
	content := combined[startIdx : startIdx+endIdx+len(closeTag)]
	return c.stripTags(content)
}

// extractHref extracts href attribute from HTML
func (c *Client) extractHref(lines []string, startLine int) string {
	combined := strings.Join(lines[startLine:], " ")
	
	// Look for href attribute
	hrefIdx := strings.Index(combined, "href=\"")
	if hrefIdx == -1 {
		return ""
	}
	
	startIdx := hrefIdx + len("href=\"")
	endIdx := strings.Index(combined[startIdx:], "\"")
	if endIdx == -1 {
		return ""
	}
	
	href := combined[startIdx : startIdx+endIdx]
	
	// Handle DuckDuckGo's redirect URLs
	if strings.Contains(href, "/l.php?u=") {
		// Extract actual URL from redirect
		u, err := url.Parse(href)
		if err == nil {
			redirectURL := u.Query().Get("u")
			if redirectURL != "" {
				return redirectURL
			}
		}
	}
	
	return href
}

// stripTags removes HTML tags from a string
func (c *Client) stripTags(html string) string {
	// Remove script and style tags first
	html = strings.ReplaceAll(html, "<script>", "")
	html = strings.ReplaceAll(html, "</script>", "")
	html = strings.ReplaceAll(html, "<style>", "")
	html = strings.ReplaceAll(html, "</style>", "")
	
	// Remove all other tags
	var result strings.Builder
	inTag := false
	for _, ch := range html {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(ch)
		}
	}
	
	// Decode common HTML entities
	text := result.String()
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	return strings.TrimSpace(text)
}

// cleanHTML cleans and normalizes HTML content
func (c *Client) cleanHTML(html string) string {
	return c.stripTags(html)
}

// SearchSimple performs a simple search and returns formatted results as text
func (c *Client) SearchSimple(ctx context.Context, query string) (string, error) {
	resp, err := c.Search(ctx, query)
	if err != nil {
		return "", err
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	
	for i, result := range resp.Results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		if result.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", result.Snippet))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString(fmt.Sprintf("\nFound %d results in %v", resp.TotalCount, resp.SearchTime))
	
	return sb.String(), nil
}

// FetchURL fetches and returns the content of a URL
func (c *Client) FetchURL(ctx context.Context, urlString string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlString, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DeepSeek-CLI/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("URL returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	
	// Return plain text version
	return c.stripTags(string(body)), nil
}

// MarshalJSON implements json.Marshaler for SearchResponse
func (r *SearchResponse) MarshalJSON() ([]byte, error) {
	type Alias SearchResponse
	return json.Marshal(&struct {
		SearchTime string `json:"search_time"`
		*Alias
	}{
		SearchTime: r.SearchTime.String(),
		Alias:      (*Alias)(r),
	})
}
