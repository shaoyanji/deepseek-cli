# DeepSeek TUI Feature Parity Implementation Plan

## Overview
This document outlines the implementation plan to achieve feature parity with the Rust-based DeepSeek TUI project.

## Current State Analysis
The existing Go-based DeepSeek CLI has:
- ✅ Basic CLI with chat, FIM, models, balance commands
- ✅ Simple TUI with Bubble Tea
- ✅ Basic agent tools (view, edit, bash, ls, grep, git, fetch, lsp, web_search)
- ✅ Configuration via YAML and environment variables
- ✅ Streaming support
- ✅ Thinking mode support

## Missing Features to Implement

### Phase 1: Core Agent Architecture
1. **Agent Loop Engine** (`internal/engine/`)
   - Turn management
   - Tool orchestration
   - Session state management
   - Async execution support

2. **Execution Policies** (`internal/execpolicy/`)
   - Three modes: Acme (Plan), Agent, YOLO
   - Tool approval workflows
   - Mode switching

3. **Enhanced TUI** (`internal/tui/`)
   - Thinking-mode streaming visualization
   - Live cost tracking (per-turn and session-level)
   - Dark theme (DeepSeek-blue palette)
   - Multi-panel layout (chat, tools, thinking, status)

### Phase 2: Advanced Session Management
4. **Session Persistence** (`internal/session/`)
   - Checkpoint save/resume
   - Conversation history serialization
   - Token usage tracking

5. **Workspace Rollback** (`internal/rollback/`)
   - Side-git system for pre/post-turn snapshots
   - `/restore` and `revert_turn` commands
   - Non-destructive versioning

### Phase 3: Extended Tooling
6. **Sub-Agents** (`internal/agent/subagent.go`)
   - Child agent orchestration
   - Task delegation
   - Result aggregation

7. **MCP Client** (`internal/mcp/`)
   - Model Context Protocol implementation
   - stdio transport
   - HTTP/SSE transport
   - External server connections

8. **Skills/Plugins** (`internal/skills/`)
   - Plugin system architecture
   - Dynamic skill loading
   - Skill registry

### Phase 4: Advanced Reasoning
9. **RLM System** (`internal/rlm/`)
   - Recursive Language Model
   - Parallel sub-task fan-out (up to 16)
   - Batch analysis and decomposition
   - Python REPL integration

10. **Enhanced Prompts** (`internal/prompts/`)
    - Decomposition-first system prompts
    - Checklist writing
    - Plan updating
    - Mode-specific prompts

### Phase 5: Runtime & Task Management
11. **Runtime API** (`internal/runtime/`)
    - HTTP/SSE API for headless mode
    - RESTful endpoints
    - WebSocket support

12. **Task Manager** (`internal/tasks/`)
    - Persistent task queue
    - Long-running operation handling
    - Task scheduling

### Phase 6: Developer Experience
13. **LSP Diagnostics** (`internal/lsp/`)
    - Post-edit diagnostics
    - Real-time error display in TUI
    - Multiple language server support

14. **Hooks System** (`internal/hooks/`)
    - Pre-execution hooks
    - Post-execution hooks
    - Custom hook scripts

15. **CLI Enhancements**
    - Interactive mode selection
    - Enhanced help system
    - Command aliases

## Implementation Priority

### High Priority (Core Functionality)
1. Execution policies (Acme/Agent/YOLO modes)
2. Enhanced TUI with thinking visualization
3. Session save/resume
4. Tool approval workflow
5. Live cost tracking

### Medium Priority (Extended Features)
6. Sub-agents
7. MCP client
8. Workspace rollback
9. Skills/plugins
10. Enhanced prompts

### Lower Priority (Advanced)
11. RLM system
12. Runtime API
13. Task manager
14. Hooks system
15. LSP diagnostics

## File Structure Changes

```
/workspace/
├── internal/
│   ├── engine/           # NEW: Core agent loop
│   │   ├── engine.go
│   │   ├── turn.go
│   │   └── orchestrator.go
│   ├── execpolicy/       # NEW: Execution policies
│   │   ├── policy.go
│   │   ├── acme.go
│   │   ├── agent.go
│   │   └── yolo.go
│   ├── session/          # NEW: Session management
│   │   ├── session.go
│   │   ├── checkpoint.go
│   │   └── history.go
│   ├── rollback/         # NEW: Workspace rollback
│   │   ├── snapshot.go
│   │   └── sidegit.go
│   ├── mcp/              # NEW: MCP protocol
│   │   ├── client.go
│   │   ├── transport.go
│   │   └── protocol.go
│   ├── skills/           # NEW: Plugin system
│   │   ├── registry.go
│   │   ├── loader.go
│   │   └── skill.go
│   ├── rlm/              # NEW: Recursive LM
│   │   ├── rlm.go
│   │   └── parallel.go
│   ├── runtime/          # NEW: HTTP API
│   │   ├── server.go
│   │   └── handlers.go
│   ├── tasks/            # NEW: Task management
│   │   ├── manager.go
│   │   └── queue.go
│   ├── hooks/            # NEW: Hook system
│   │   ├── hooks.go
│   │   └── runner.go
│   ├── prompts/          # NEW: Prompt templates
│   │   ├── templates.go
│   │   └── system.go
│   ├── agent/            # EXISTING: Enhance
│   │   ├── agent.go
│   │   ├── subagent.go   # NEW
│   │   └── tools.go      # NEW
│   ├── tui/              # EXISTING: Major enhancement
│   │   ├── tui.go
│   │   ├── thinking.go   # NEW
│   │   ├── cost.go       # NEW
│   │   ├── panels.go     # NEW
│   │   └── theme.go      # NEW
│   ├── lsp/              # EXISTING: Enhance
│   │   └── lsp.go
│   └── exec/             # EXISTING: Enhance
│       └── exec.go
├── cmd/                  # NEW: Command structure
│   └── deepseek/
│       └── main.go
└── ...existing files...
```

## Key Technical Decisions

1. **Go Version**: Maintain Go 1.19+ compatibility
2. **TUI Framework**: Continue with Bubble Tea + Bubbles
3. **Async Pattern**: Use goroutines + channels for async operations
4. **Storage**: JSON for session data, git for rollback
5. **API Compatibility**: Maintain backward compatibility with existing CLI

## Testing Strategy

1. Unit tests for each new package
2. Integration tests for agent loop
3. E2E tests for TUI interactions
4. Mock LLM client for testing without API calls

## Migration Path

1. Keep existing CLI commands functional
2. Gradual rollout of new features
3. Feature flags for experimental features
4. Documentation updates for each phase
