// Package security provides security auditing for AI output.
package security

import (
	"regexp"
	"strings"
)

// DangerousPattern represents a pattern that may indicate dangerous commands
type DangerousPattern struct {
	Pattern     *regexp.Regexp
	Description string
	Severity    string // "high", "medium", "low"
}

// AuditResult holds the results of an audit scan
type AuditResult struct {
	Detected       bool
	DangerousLines []string
	Patterns       []string
	Severities     []string
}

// Define dangerous command patterns
var dangerousPatterns = []DangerousPattern{
	{
		Pattern:     regexp.MustCompile(`(?i)rm\s+(-[rf]+\s+)?/(\s|$|\*)`),
		Description: "Recursive root deletion",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)rm\s+-rf\s+\*`),
		Description: "Delete all files in current directory",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)dd\s+if=`),
		Description: "Disk dump operation",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)>\s*/dev/`),
		Description: "Redirect to device file",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)mkfs\.`),
		Description: "Filesystem creation",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`:\(\)\{\s*:|:&\s*\};:`),
		Description: "Fork bomb",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)chmod\s+(-R\s+)?777\s+/`),
		Description: "Make all files world-writable",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)chown\s+(-R\s+)?`),
		Description: "Change file ownership recursively",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)sudo\s+rm`),
		Description: "Sudo delete operation",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)>/etc/`),
		Description: "Write to /etc directory",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)curl.*\|\s*(ba)?sh`),
		Description: "Pipe curl to shell",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)wget.*\|\s*(ba)?sh`),
		Description: "Pipe wget to shell",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)base64\s+-d.*\|\s*(ba)?sh`),
		Description: "Decode and execute base64",
		Severity:    "high",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)eval\s*\(`),
		Description: "Eval execution",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)exec\s*\(`),
		Description: "Exec execution",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)system\s*\(`),
		Description: "System call execution",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)os\.system\s*\(`),
		Description: "Python os.system call",
		Severity:    "medium",
	},
	{
		Pattern:     regexp.MustCompile(`(?i)subprocess\.(call|run|Popen)`),
		Description: "Python subprocess execution",
		Severity:    "medium",
	},
}

// AuditOutputForDangerousCommands scans text for dangerous shell commands
func AuditOutputForDangerousCommands(text string) []string {
	var warnings []string
	
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		for _, pattern := range dangerousPatterns {
			if pattern.Pattern.MatchString(line) {
				warnings = append(warnings, line)
				break // Only report each line once
			}
		}
	}
	
	return warnings
}

// AuditOutputDetailed performs a detailed audit with pattern information
func AuditOutputDetailed(text string) *AuditResult {
	result := &AuditResult{
		Detected: false,
	}
	
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		for _, pattern := range dangerousPatterns {
			if pattern.Pattern.MatchString(line) {
				result.Detected = true
				result.DangerousLines = append(result.DangerousLines, line)
				result.Patterns = append(result.Patterns, pattern.Description)
				result.Severities = append(result.Severities, pattern.Severity)
				break
			}
		}
	}
	
	return result
}

// HasDangerousCommands returns true if any dangerous commands are detected
func HasDangerousCommands(text string) bool {
	warnings := AuditOutputForDangerousCommands(text)
	return len(warnings) > 0
}

// GetPatternDescriptions returns descriptions of all monitored patterns
func GetPatternDescriptions() []string {
	descriptions := make([]string, len(dangerousPatterns))
	for i, p := range dangerousPatterns {
		descriptions[i] = p.Description
	}
	return descriptions
}
