// Package tools provides built-in tools for agentic workflows.
package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ToolDefinition defines a tool's schema for function calling
type ToolDefinition struct {
	Type        string                 `json:"type"`
	Function    FunctionDefinition     `json:"function"`
}

// FunctionDefinition defines a function's schema
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// GetBuiltInTools returns all built-in tools available for function calling
func GetBuiltInTools() []ToolDefinition {
	return []ToolDefinition{
		getWebSearchTool(),
		getCurrentWeatherTool(),
		getCalculatorTool(),
	}
}

// getWebSearchTool returns the web search tool definition
func getWebSearchTool() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "web_search",
			Description: "Search the web for information using DuckDuckGo. Returns search results with titles, URLs, and snippets.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"num_results": map[string]interface{}{
						"type":        "number",
						"description": "Number of results to return (default: 5)",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// getCurrentWeatherTool returns the weather tool definition
func getCurrentWeatherTool() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "current_weather",
			Description: "Get current weather information for a city using wttr.in. Returns temperature, conditions, humidity, wind speed, etc.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type":        "string",
						"description": "The city name to get weather for",
					},
					"units": map[string]interface{}{
						"type":        "string",
						"description": "Units: 'metric' (Celsius) or 'imperial' (Fahrenheit). Default: metric",
						"enum":        []string{"metric", "imperial"},
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

// getCalculatorTool returns the calculator tool definition
func getCalculatorTool() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "calculator",
			Description: "Evaluate a mathematical expression safely. Supports basic arithmetic (+, -, *, /), parentheses, exponents (^), and common math functions (sin, cos, tan, log, sqrt, abs, etc.).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "The mathematical expression to evaluate (e.g., '2 + 2', 'sqrt(16)', 'sin(pi/2)')",
					},
				},
				"required": []string{"expression"},
			},
		},
	}
}

// ExecuteTool executes a built-in tool by name
func ExecuteTool(toolName string, args map[string]interface{}) (*ToolResult, error) {
	switch toolName {
	case "web_search":
		return executeWebSearch(args)
	case "current_weather":
		return executeWeather(args)
	case "calculator":
		return executeCalculator(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// executeWebSearch performs a web search using DuckDuckGo HTML
func executeWebSearch(args map[string]interface{}) (*ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &ToolResult{Error: "query argument is required"}, nil
	}

	numResults := 5
	if n, ok := args["num_results"].(float64); ok {
		numResults = int(n)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://html.duckduckgo.com/html/"
	params := url.Values{}
	params.Set("q", query)

	reqURL := baseURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("creating request: %v", err)}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ToolResult{Error: fmt.Sprintf("search returned status %d", resp.StatusCode)}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("reading response: %v", err)}, nil
	}

	results := parseDuckDuckGoHTML(string(body), numResults)
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	return &ToolResult{Content: sb.String()}, nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// parseDuckDuckGoHTML parses DuckDuckGo's HTML search results
func parseDuckDuckGoHTML(html string, maxResults int) []SearchResult {
	var results []SearchResult
	
	lines := strings.Split(html, "\n")
	for i, line := range lines {
		if strings.Contains(line, `class="result"`) {
			title := extractElement(lines, i, "a", `class="result__a"`)
			url := extractHref(lines, i)
			snippet := extractElement(lines, i, "a", `class="result__snippet"`)
			
			if title != "" && url != "" {
				results = append(results, SearchResult{
					Title:   cleanHTML(title),
					URL:     url,
					Snippet: cleanHTML(snippet),
				})
				
				if len(results) >= maxResults {
					break
				}
			}
		}
	}
	
	return results
}

func extractElement(lines []string, startLine int, tag string, attrs string) string {
	combined := strings.Join(lines[startLine:], " ")
	
	openTag := "<" + tag
	if attrs != "" {
		openTag += " " + attrs
	}
	
	startIdx := strings.Index(combined, openTag)
	if startIdx == -1 {
		return ""
	}
	
	closeTag := "</" + tag + ">"
	endIdx := strings.Index(combined[startIdx:], closeTag)
	if endIdx == -1 {
		return ""
	}
	
	content := combined[startIdx : startIdx+endIdx+len(closeTag)]
	return stripTags(content)
}

func extractHref(lines []string, startLine int) string {
	combined := strings.Join(lines[startLine:], " ")
	
	hrefIdx := strings.Index(combined, `href="`)
	if hrefIdx == -1 {
		return ""
	}
	
	startIdx := hrefIdx + len(`href="`)
	endIdx := strings.Index(combined[startIdx:], `"`)
	if endIdx == -1 {
		return ""
	}
	
	return combined[startIdx : startIdx+endIdx]
}

func stripTags(html string) string {
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
	
	text := result.String()
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	return strings.TrimSpace(text)
}

func cleanHTML(html string) string {
	return stripTags(html)
}

// executeWeather gets weather from wttr.in
func executeWeather(args map[string]interface{}) (*ToolResult, error) {
	city, ok := args["city"].(string)
	if !ok || city == "" {
		return &ToolResult{Error: "city argument is required"}, nil
	}

	units := "m" // metric by default
	if u, ok := args["units"].(string); ok {
		if u == "imperial" {
			units = "u"
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	url := fmt.Sprintf("https://wttr.in/%s?format=j1&%s", url.QueryEscape(city), units)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("creating request: %v", err)}, nil
	}

	req.Header.Set("User-Agent", "deepseek-cli/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("weather request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ToolResult{Error: fmt.Sprintf("weather API returned status %d", resp.StatusCode)}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("reading response: %v", err)}, nil
	}

	// Parse JSON response - wttr.in returns nested values
	var weatherData map[string]interface{}
	if err := json.Unmarshal(body, &weatherData); err != nil {
		return &ToolResult{Error: fmt.Sprintf("parsing weather data: %v", err)}, nil
	}

	// Extract current condition
	currentCond, ok := weatherData["current_condition"].([]interface{})
	if !ok || len(currentCond) == 0 {
		return &ToolResult{Error: "no weather data available"}, nil
	}

	cc, ok := currentCond[0].(map[string]interface{})
	if !ok {
		return &ToolResult{Error: "invalid weather data format"}, nil
	}

	// Extract nearest area
	nearestArea, ok := weatherData["nearest_area"].([]interface{})
	var areaName, areaRegion, areaCountry string
	if ok && len(nearestArea) > 0 {
		area, ok := nearestArea[0].(map[string]interface{})
		if ok {
			if cityVal, ok := area["areaName"].([]interface{}); ok && len(cityVal) > 0 {
				if m, ok := cityVal[0].(map[string]interface{}); ok {
					if v, ok := m["value"].(string); ok {
						areaName = v
					}
				}
			}
			if regionVal, ok := area["region"].([]interface{}); ok && len(regionVal) > 0 {
				if m, ok := regionVal[0].(map[string]interface{}); ok {
					if v, ok := m["value"].(string); ok {
						areaRegion = v
					}
				}
			}
			if countryVal, ok := area["country"].([]interface{}); ok && len(countryVal) > 0 {
				if m, ok := countryVal[0].(map[string]interface{}); ok {
					if v, ok := m["value"].(string); ok {
						areaCountry = v
					}
				}
			}
		}
	}

	// Helper to get string value from weather data
	getStr := func(key string) string {
		if v, ok := cc[key].(string); ok {
			return v
		}
		return ""
	}

	// Get weather description (it's an array of objects with "value" field)
	weatherDesc := ""
	if descRaw, ok := cc["weatherDesc"].([]interface{}); ok && len(descRaw) > 0 {
		if descMap, ok := descRaw[0].(map[string]interface{}); ok {
			if v, ok := descMap["value"].(string); ok {
				weatherDesc = v
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Weather for %s, %s, %s\n", 
		cleanLocation(areaName), 
		cleanLocation(areaRegion),
		cleanLocation(areaCountry)))
	sb.WriteString(fmt.Sprintf("Conditions: %s\n", weatherDesc))
	sb.WriteString(fmt.Sprintf("Temperature: %s°C (%s°F)\n", getStr("temp_C"), getStr("temp_F")))
	sb.WriteString(fmt.Sprintf("Feels like: %s°C (%s°F)\n", getStr("FeelsLikeC"), getStr("FeelsLikeF")))
	sb.WriteString(fmt.Sprintf("Humidity: %s%%\n", getStr("humidity")))
	sb.WriteString(fmt.Sprintf("Wind: %s km/h (%s mph)\n", getStr("windspeedKmph"), getStr("windspeedMiles")))
	sb.WriteString(fmt.Sprintf("Pressure: %s mb\n", getStr("pressure")))
	sb.WriteString(fmt.Sprintf("Visibility: %s km\n", getStr("visibility")))
	sb.WriteString(fmt.Sprintf("UV Index: %s\n", getStr("uvIndex")))

	return &ToolResult{Content: sb.String()}, nil
}

func cleanLocation(s string) string {
	return strings.TrimSpace(s)
}

// executeCalculator evaluates a mathematical expression safely
func executeCalculator(args map[string]interface{}) (*ToolResult, error) {
	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return &ToolResult{Error: "expression argument is required"}, nil
	}

	// Validate expression contains only allowed characters
	
	lowerExpr := strings.ToLower(expression)
	
	// Check for dangerous patterns
	if strings.Contains(lowerExpr, "exec") || strings.Contains(lowerExpr, "eval") || 
	   strings.Contains(lowerExpr, "system") || strings.Contains(lowerExpr, "import") {
		return &ToolResult{Error: "expression contains disallowed keywords"}, nil
	}

	// Use Python for safe evaluation via subprocess
	cmdStr := fmt.Sprintf(`python3 -c "
import math
import sys
expr = sys.argv[1]
try:
    # Replace common math functions
    expr = expr.replace('^', '**')
    expr = expr.replace('pi', str(math.pi))
    expr = expr.replace('e', str(math.e))
    expr = expr.replace('sin', 'math.sin')
    expr = expr.replace('cos', 'math.cos')
    expr = expr.replace('tan', 'math.tan')
    expr = expr.replace('asin', 'math.asin')
    expr = expr.replace('acos', 'math.acos')
    expr = expr.replace('atan', 'math.atan')
    expr = expr.replace('log', 'math.log10')
    expr = expr.replace('ln', 'math.log')
    expr = expr.replace('sqrt', 'math.sqrt')
    expr = expr.replace('abs', 'abs')
    expr = expr.replace('exp', 'math.exp')
    result = eval(expr)
    print(result)
except Exception as e:
    print(f'ERROR: {e}', file=sys.stderr)
    sys.exit(1)
" %s`, expression)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("bash", "-c", cmdStr)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Error: fmt.Sprintf("calculation failed: %v, output: %s", err, string(output))}, nil
	}

	result := strings.TrimSpace(string(output))
	
	// Try to format the result nicely
	if num, err := strconv.ParseFloat(result, 64); err == nil {
		// Format with reasonable precision
		if num == float64(int64(num)) {
			result = fmt.Sprintf("%.0f", num)
		} else {
			result = fmt.Sprintf("%.6g", num)
		}
	}

	return &ToolResult{Content: fmt.Sprintf("%s = %s", expression, result)}, nil
}
