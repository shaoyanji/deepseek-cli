# PLAN.md: DeepSeek CLI → Agentic Code Execution Harness

## 1. Project Overview
Transform the existing DeepSeek API CLI into a full-featured agentic code execution harness inspired by `charmbracelet/crush`, with:
- Retained existing API mapping CLI commands (`chat`/`fim`/`models`/`balance`)
- New non-interactive shapes: `deepseek -p "<prompt>"`, `echo "prompt" | deepseek`
- Bubble Tea TUI (crush-like interface)
- Client-side speculative decoding
- Best N evaluation with DeepSeek as judge
- 100% test coverage via strict TDD

## 2. Phase Breakdown (TDD-First)

### Phase 1: Project Restructuring & Dependencies
**Goal**: Reorganize codebase, add dependencies, make API client mockable.
**Tests**:
- `TestAPIClientInterface`: Validate interface compliance
- `TestPackageImports`: Validate internal package structure

**Steps**:
1. Refactor `client.go` to use `APIClient` interface for mockability:
   ```go
   type APIClient interface {
       ChatCompletion(req ChatRequest) (*ChatResponse, error)
       FIMCompletion(req FIMRequest) (*FIMResponse, error)
       ListModels() (*ModelsResponse, error)
       GetBalance() (*BalanceResponse, error)
   }
   ```
2. Add Charm TUI dependencies:
   ```bash
   go get github.com/charmbracelet/bubbletea@latest
   go get github.com/charmbracelet/lipgloss@latest
   go get github.com/charmbracelet/bubbles@latest
   go get github.com/charmbracelet/glamour@latest
   go get github.com/stretchr/testify@latest
   ```
3. Reorganize into `internal/` packages:
   - `internal/agent/`: Tool definitions, agent logic
   - `internal/tui/`: Bubble Tea UI components
   - `internal/speculative/`: Client-side speculative decoding
   - `internal/bestn/`: Best N candidate evaluation
   - `internal/exec/`: Code/command execution
4. Update `Taskfile.yml` with new build/test/lint targets.

### Phase 2: Retain & Extend CLI (TDD First)
**Goal**: Preserve existing commands, add new non-interactive shapes. Write tests *before* implementation.
**Tests** (100% coverage):
- `TestChatCommand`: Validate existing `chat` subcommand behavior
- `TestFIMCommand`: Validate existing `fim` subcommand behavior
- `TestModelsCommand`: Validate existing `models` subcommand behavior
- `TestBalanceCommand`: Validate existing `balance` subcommand behavior
- `TestPromptFlag`: Validate `-p` flag for single-turn code generation
- `TestStdinInput`: Validate piped stdin input handling
- `TestMultiTurn`: Validate multi-turn support via `--history` flag

**Steps**:
1. Write tests for all existing commands first (establish baseline coverage)
2. Keep all existing Cobra commands unchanged
3. Add `prompt` flag (`-p`/`--prompt`) to root command
4. Add stdin detection: read prompt from stdin if not a terminal
5. Add `--history` flag for multi-turn session persistence

### Phase 3: Bubble Tea TUI
**Goal**: Implement crush-like TUI with Charm stack.
**Tests** (100% coverage):
- `TestTUIInitialization`: Validate TUI starts correctly
- `TestChatViewRendering`: Validate chat history rendering with code highlighting
- `TestInputViewRendering`: Validate multiline input rendering
- `TestToolOutputPanel`: Validate tool output display
- `TestStatusBar`: Validate model/mode/token usage display
- `TestSessionSaveLoad`: Validate session persistence to disk

**Steps**:
1. Implement root TUI model with Bubble Tea
2. Create sub-models for each view: chat, input, tools, status bar
3. Add syntax highlighting via `glamour` for code blocks
4. Implement session save/load to `~/.local/share/deepseek-cli/sessions/`

### Phase 4: Agentic Tools & Code Execution
**Goal**: Add built-in tools and local code execution.
**Tests** (100% coverage):
- `TestViewTool`: Validate file viewing with line numbers
- `TestEditTool`: Validate file editing with backup creation
- `TestBashTool`: Validate command execution (sandboxed)
- `TestLSTool`: Validate directory listing
- `TestGrepTool`: Validate content search
- `TestGitTool`: Validate git commands
- `TestCodeExec`: Validate code execution (Go/Python/Node) with stdout/stderr capture

**Steps**:
1. Define tool interface:
   ```go
   type Tool interface {
       Name() string
       Run(args map[string]interface{}) (string, error)
   }
   ```
2. Implement built-in tools: `view`, `edit`, `bash`, `ls`, `grep`, `git`
3. Implement `internal/exec/` for code execution with runtime detection

### Phase 5: Client-Side Speculative Decoding
**Goal**: Orchestrate draft/target model calls via DeepSeek API.
**Tests** (100% coverage):
- `TestDraftModelCall`: Validate draft model API request/response
- `TestTargetModelVerify`: Validate target model verification of draft tokens
- `TestSpeculativeDecodingFlow`: End-to-end flow with mock API
- `TestKVReuse`: Validate DeepSeek context caching for KV reuse
- `TestConfigFlags`: Validate `--speculative`, `--draft-model`, `--num-speculative-tokens` flags

**Steps**:
1. Implement `internal/speculative/` with mockable interfaces
2. Use DeepSeek's context caching API to reuse KV cache between calls
3. Add CLI flags for speculative decoding configuration

### Phase 6: Best N Evaluation
**Goal**: Generate N candidates, evaluate with DeepSeek as judge.
**Tests** (100% coverage):
- `TestGenerateNCandidates`: Validate N candidate generation
- `TestDeepSeekEvaluator`: Validate DeepSeek evaluation with JSON schema
- `TestBestNSelection`: Validate winner selection and merge
- `TestEvaluatorSchema`: Validate JSON schema compliance for evaluator responses

**Steps**:
1. Implement `internal/bestn/` with evaluator interface
2. Define strict JSON schema for evaluator response:
   ```json
   {
     "type": "object",
     "properties": {
       "winner": { "type": "integer" },
       "recommendations": { "type": "array", "items": { "type": "string" } },
       "merged": { "type": "string" }
     },
     "required": ["winner"]
   }
   ```
3. Add CLI flags: `--best-n`, `--evaluator-model`

### Phase 7: Testing & Quality Assurance
**Goal**: Maintain 100% test coverage, add linting.
**Steps**:
1. Write table-driven tests for all functions
2. Use `testify/mock` for API client mocking
3. Update `Taskfile.yml` with:
   - `task test`: Run all tests with coverage
   - `task lint`: Run `golangci-lint`
   - `task ci`: Run test + lint + 100% coverage check
4. Update GitHub Actions to fail on <100% coverage

## 3. Coverage Enforcement
- All packages must have 100% test coverage (measured via `go test -coverprofile=coverage.out`)
- CI pipeline fails if coverage < 100%
- New code must have corresponding tests *before* implementation (TDD)

## 4. Timeline
1. Phase 1-2: 2 days (restructure + CLI extensions + baseline tests)
2. Phase 3: 3 days (TUI implementation)
3. Phase 4: 2 days (tools + exec)
4. Phase 5: 3 days (speculative decoding)
5. Phase 6: 2 days (Best N)
6. Phase 7: Ongoing (testing + QA)
