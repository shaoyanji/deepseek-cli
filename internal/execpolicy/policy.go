// Package execpolicy provides execution policies for agent tool usage.
// It defines three modes: Acme (Plan), Agent, and YOLO.
package execpolicy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ExecutionMode represents the agent's autonomy level
type ExecutionMode string

const (
	// ModeAcme - Read-only mode for exploration and analysis (Plan mode)
	ModeAcme ExecutionMode = "acme"
	// ModeAgent - Interactive mode with human-in-the-loop approval
	ModeAgent ExecutionMode = "agent"
	// ModeYOLO - Fully automated mode without approval
	ModeYOLO ExecutionMode = "yolo"
)

// ToolApproval represents the result of a tool approval check
type ToolApproval struct {
	Approved bool
	Reason   string
}

// Policy defines the execution policy interface
type Policy interface {
	// Name returns the policy name
	Name() string
	
	// CanExecute checks if a tool can be executed without approval
	CanExecute(toolName string, args map[string]interface{}) bool
	
	// RequiresApproval checks if a tool requires user approval
	RequiresApproval(toolName string, args map[string]interface{}) bool
	
	// IsReadOnly returns true if this is a read-only policy
	IsReadOnly() bool
	
	// ApproveTool prompts for or determines tool approval
	ApproveTool(toolName string, args map[string]interface{}, description string) (*ToolApproval, error)
}

// AcmePolicy implements read-only exploration mode
type AcmePolicy struct{}

func NewAcmePolicy() *AcmePolicy {
	return &AcmePolicy{}
}

func (p *AcmePolicy) Name() string {
	return "Acme (Plan)"
}

func (p *AcmePolicy) CanExecute(toolName string, args map[string]interface{}) bool {
	// Only read-only tools are allowed in Acme mode
	return p.isReadOnlyTool(toolName)
}

func (p *AcmePolicy) RequiresApproval(toolName string, args map[string]interface{}) bool {
	// No approval needed - write tools are simply not allowed
	return false
}

func (p *AcmePolicy) IsReadOnly() bool {
	return true
}

func (p *AcmePolicy) ApproveTool(toolName string, args map[string]interface{}, description string) (*ToolApproval, error) {
	if p.isReadOnlyTool(toolName) {
		return &ToolApproval{Approved: true, Reason: "Read-only tool"}, nil
	}
	return &ToolApproval{
		Approved: false,
		Reason:   fmt.Sprintf("Tool '%s' is not allowed in Acme (Plan) mode - write operations disabled", toolName),
	}, nil
}

func (p *AcmePolicy) isReadOnlyTool(toolName string) bool {
	readOnlyTools := map[string]bool{
		"view":      true,
		"ls":        true,
		"grep":      true,
		"fetch":     true,
		"web_search": true,
		"lsp":       true,
	}
	return readOnlyTools[toolName]
}

// AgentPolicy implements interactive human-in-the-loop mode
type AgentPolicy struct {
	reader *bufio.Reader
}

func NewAgentPolicy() *AgentPolicy {
	return &AgentPolicy{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (p *AgentPolicy) Name() string {
	return "Agent"
}

func (p *AgentPolicy) CanExecute(toolName string, args map[string]interface{}) bool {
	// In Agent mode, nothing executes without approval
	return false
}

func (p *AgentPolicy) RequiresApproval(toolName string, args map[string]interface{}) bool {
	// All tools require approval in Agent mode
	return true
}

func (p *AgentPolicy) IsReadOnly() bool {
	return false
}

func (p *AgentPolicy) ApproveTool(toolName string, args map[string]interface{}, description string) (*ToolApproval, error) {
	fmt.Printf("\n🔧 Tool Request: %s\n", toolName)
	fmt.Printf("Description: %s\n", description)
	
	// Show arguments
	if len(args) > 0 {
		fmt.Println("Arguments:")
		for k, v := range args {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	
	fmt.Print("\nApprove? [y/n/skip]: ")
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	
	switch input {
	case "y", "yes":
		return &ToolApproval{Approved: true, Reason: "User approved"}, nil
	case "s", "skip":
		return &ToolApproval{Approved: false, Reason: "User skipped"}, nil
	default:
		return &ToolApproval{Approved: false, Reason: "User denied"}, nil
	}
}

// YOLOPolicy implements fully automated mode
type YOLOPolicy struct{}

func NewYOLOPolicy() *YOLOPolicy {
	return &YOLOPolicy{}
}

func (p *YOLOPolicy) Name() string {
	return "YOLO"
}

func (p *YOLOPolicy) CanExecute(toolName string, args map[string]interface{}) bool {
	// In YOLO mode, everything executes automatically
	return true
}

func (p *YOLOPolicy) RequiresApproval(toolName string, args map[string]interface{}) bool {
	// No approval needed in YOLO mode
	return false
}

func (p *YOLOPolicy) IsReadOnly() bool {
	return false
}

func (p *YOLOPolicy) ApproveTool(toolName string, args map[string]interface{}, description string) (*ToolApproval, error) {
	// Auto-approve everything in YOLO mode
	return &ToolApproval{Approved: true, Reason: "Auto-approved in YOLO mode"}, nil
}

// PolicyFactory creates policies by name
type PolicyFactory struct{}

func NewPolicyFactory() *PolicyFactory {
	return &PolicyFactory{}
}

func (f *PolicyFactory) Create(mode ExecutionMode) (Policy, error) {
	switch mode {
	case ModeAcme:
		return NewAcmePolicy(), nil
	case ModeAgent:
		return NewAgentPolicy(), nil
	case ModeYOLO:
		return NewYOLOPolicy(), nil
	default:
		return nil, fmt.Errorf("unknown execution mode: %s", mode)
	}
}

func (f *PolicyFactory) ParseMode(modeStr string) (ExecutionMode, error) {
	modeStr = strings.ToLower(strings.TrimSpace(modeStr))
	
	switch modeStr {
	case "acme", "plan", "read-only":
		return ModeAcme, nil
	case "agent", "interactive":
		return ModeAgent, nil
	case "yolo", "auto", "automatic":
		return ModeYOLO, nil
	default:
		return "", fmt.Errorf("unknown mode: %s (valid: acme, agent, yolo)", modeStr)
	}
}
