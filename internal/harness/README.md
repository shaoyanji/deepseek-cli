# Go Harness - DeepSeek Agent Framework

A Go implementation of an agent harness inspired by [deepseek-harness](https://github.com/deepseek-ai/deepseek-harness), maintaining a non-TypeScript approach.

## Architecture

This harness follows the **everything-is-a-plugin** architecture pattern, where all capabilities are provided through swappable plugins mounted into a shared runtime context.

### Core Concepts

1. **Context**: The shared runtime context that all plugins contribute to and consume from. There's no privileged core - everything is a plugin.

2. **Plugins**: Self-contained units that register services, events, and reversible effects. Plugins can be mounted and unmounted dynamically.

3. **Capability Seams**: Swappable capabilities with three roles:
   - **Service Definition**: Declares the interface
   - **Service Provider**: Implements the capability
   - **Consumer**: Uses the capability (commonly a model-facing tool)

4. **Scopes**: Provide scoped registration that isolates one agent's contributions from others.

5. **Agent Loop**: The default driver that creates agents and runs turn/step lifecycles.

### Package Structure

```
internal/harness/
├── context.go      # Runtime context and plugin system
├── services.go     # Service interfaces (capability seams)
└── agentloop.go    # Default agent loop implementation
```

## Key Services

The harness defines these core capability seams:

- **SessionService**: Manages durable session event logs
- **SystemPromptService**: Assembles system prompts from sections and tool schemas
- **ToolRegistry**: Manages model-facing tools
- **AgentRegistry**: Manages agent instances
- **LLMService**: Provides language model adapters
- **FSService**: Filesystem capabilities
- **ShellService**: Shell execution capabilities
- **SubagentService**: Subagent delegation
- **SandboxService**: Process confinement

## Usage Example

```go
package main

import (
    "deepseek-cli/internal/harness"
)

func main() {
    // Create runtime context
    ctx := harness.NewContext()
    
    // Create agent loop with configuration
    agentLoop := harness.NewAgentLoop(harness.AgentLoopConfig{
        MaxParallelToolCalls: 10,
        Agents: []harness.AgentConfig{
            {
                ID:       "main",
                Provider: "deepseek",
                Model:    "deepseek-chat",
            },
        },
    })
    
    // Mount the agent loop plugin
    if err := ctx.Mount(agentLoop); err != nil {
        panic(err)
    }
    
    // Access agents through the context
    agents := ctx.Agents.List()
    
    // ... use agents ...
    
    // Cleanup
    ctx.Unmount("dsh-agent-loop")
}
```

## Plugin Development

To create a custom plugin:

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Install(ctx *harness.Context) error {
    // Register services, events, effects
    ctx.Inject(func() {
        // Setup
    }, func() {
        // Cleanup (undo effect)
    })
    return nil
}

func (p *MyPlugin) Uninstall(ctx *harness.Context) error {
    // Reverse registrations
    return nil
}
```

## Design Principles

1. **Everything is a Plugin**: No privileged core code. All functionality comes from plugins.

2. **Reversible Effects**: All plugin registrations have corresponding undo operations for clean unmounting.

3. **Capability Seams**: Separate service definitions from implementations for swappability.

4. **Scoped Registration**: Each agent's contributions are isolated through scopes.

5. **Durable Session Log**: All model-visible information is recorded in append-only session events.

## Comparison with TypeScript Harness

| Feature | TypeScript Harness | Go Harness |
|---------|-------------------|------------|
| Plugin System | Cordis framework | Native Go interfaces |
| Type System | TypeScript | Go |
| Concurrency | Async/await | Goroutines |
| Configuration | YAML patches | Go structs |
| Session Storage | JSONL backends | Pluggable backends |

## Roadmap

- [ ] Implement session persistence backends (JSONL, SQLite)
- [ ] Add LLM adapter for DeepSeek API
- [ ] Implement tool execution pipeline
- [ ] Add filesystem and shell capability providers
- [ ] Implement system prompt assembly
- [ ] Add profile and bundle composition
- [ ] Create CLI entry point
- [ ] Add Web UI host

## License

MIT
