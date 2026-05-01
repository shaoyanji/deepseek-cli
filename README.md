# DeepSeek CLI

A comprehensive command-line interface for the DeepSeek API, supporting chat completions, FIM (Fill-In-the-Middle) code completion, model listing, balance checking, and an interactive TUI mode.

## Features

- **Interactive TUI**: Beautiful terminal UI with syntax highlighting, scrolling, and session management
- **Chat Completions**: Full support for DeepSeek V4 models with all parameters
- **FIM Completions**: Code completion with Fill-In-the-Middle support
- **Streaming**: Real-time streaming responses for both chat and FIM
- **Thinking Mode**: Support for DeepSeek's reasoning/thinking capabilities
- **JSON Mode**: Structured JSON output mode
- **Tools**: Function calling and tool use support
- **Beta Features**: Access to beta endpoints and prefix completion
- **Parameter Validation**: Comprehensive input validation with clear error messages
- **Multiple Output Formats**: Formatted responses, usage statistics, and raw JSON options
- **Multiple Modes**: Interactive TUI, single-turn prompts, stdin input, and history-based sessions

## Installation

### From Source

```bash
go install github.com/shaoyanji/deepseek-cli@latest
```

### From Release Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/shaoyanji/deepseek-cli/releases).

### Using Taskfile

```bash
# Build for current platform
task build

# Build for all platforms
task build:all

# Build for specific platform
task build:linux:amd64
```

## Configuration

### Environment Variables

Set your DeepSeek API key as an environment variable:

```bash
export DEEPSEEK_API_KEY="your-api-key-here"
```

Optionally set a custom base URL:

```bash
export DEEPSEEK_API_BASE="https://api.deepseek.com"
```

### Configuration File

The CLI supports a configuration file for setting default values. The config file follows XDG Base Directory specification:

**Config file locations:**
- **Linux**: `~/.config/deepseek-cli/config.yaml`
- **macOS**: `~/Library/Application Support/deepseek-cli/config.yaml`
- **Windows**: `%APPDATA%\deepseek-cli\config.yaml`

### Creating a Config File

Run the following command to create a sample configuration file:

```bash
deepseek config init
```

This will create a sample config file at the appropriate location for your OS.

### Config File Structure

```yaml
# API settings
api_key: ""  # Your DeepSeek API key (can also be set via DEEPSEEK_API_KEY env var)
base_url: ""  # Custom base URL (optional, defaults to https://api.deepseek.com)

# Chat completion defaults
chat:
  model: "deepseek-v4-pro"  # Default model
  system: ""  # Default system message
  temperature: 1.0  # Sampling temperature (0.0 to 2.0)
  top_p: 1.0  # Nucleus sampling threshold (0.0 to 1.0)
  max_tokens: 0  # Maximum tokens to generate (0 = no limit)
  frequency_penalty: 0.0  # Frequency penalty (-2.0 to 2.0)
  presence_penalty: 0.0  # Presence penalty (-2.0 to 2.0)
  thinking: "enabled"  # Thinking mode: enabled or disabled
  reasoning_effort: "high"  # Reasoning effort: high or max
  stream: false  # Enable streaming by default
  include_usage: false  # Include usage info in streaming
  json_mode: false  # Enable JSON mode by default
  beta: false  # Use beta endpoint by default

# FIM completion defaults
fim:
  model: "deepseek-v4-pro"  # Default model for FIM
  max_tokens: 128  # Maximum tokens to generate (max 4096 for FIM)
  temperature: 0.2  # Sampling temperature (lower = more focused)
  top_p: 1.0  # Nucleus sampling threshold (0.0 to 1.0)
  frequency_penalty: 0.0  # Frequency penalty (-2.0 to 2.0)
  presence_penalty: 0.0  # Presence penalty (-2.0 to 2.0)
  stream: false  # Enable streaming by default
  include_usage: false  # Include usage info in streaming
  echo: false  # Echo back the prompt with completion
  beta: true  # Use beta endpoint by default for FIM
```

**Note:** The configuration file is optional. The CLI will use sensible defaults if no config file exists.

### Config Priority

Configuration values are applied in the following priority (highest to lowest):

1. **CLI flags** - Command-line arguments override everything
2. **Environment variables** - `DEEPSEEK_API_KEY` and `DEEPSEEK_API_BASE`
3. **Config file** - Default values from config file
4. **Built-in defaults** - Fallback values if nothing else is set

### Viewing Config Location

To see where your config file is located:

```bash
deepseek config
```

## Usage Modes

The DeepSeek CLI supports multiple usage modes for different workflows:

### Interactive TUI Mode (Default)

When you run `deepseek` without any subcommands or flags, it launches an interactive terminal UI:

```bash
deepseek
```

**TUI Features:**
- Beautiful terminal interface with syntax highlighting
- Markdown rendering with code block highlighting
- Scrollable chat history
- Multi-line input support
- Session persistence (saves/loads automatically)
- Real-time token usage display
- Color-coded messages (user vs assistant)

**TUI Controls:**
- `Enter` - Send message
- `Ctrl+C` or `Esc` - Exit
- Arrow keys - Navigate in input
- Scroll with mouse or keyboard in chat history

### Single-Turn Mode

For quick, one-off prompts without entering the TUI:

```bash
deepseek -p "What is the capital of France?"
# or
deepseek --prompt "Explain quantum computing"
```

### Stdin Mode

Pipe input to the CLI for scripting and automation:

```bash
echo "Write a haiku about coding" | deepseek
cat prompt.txt | deepseek
```

### History-Based Mode

Continue conversations from a previous session:

```bash
deepseek --history session.json
```

The history file should contain a JSON array of messages in the format:
```json
[
  {"role": "user", "content": "Hello"},
  {"role": "assistant", "content": "Hi there!"}
]
```

## Usage

### Chat Completions

```bash
# Basic chat
deepseek chat --user "Hello, how are you?"

# With system message
deepseek chat --system "You are a helpful assistant" --user "Hello"

# With parameters
deepseek chat --user "Hello" --temperature 0.7 --max-tokens 100

# JSON mode
deepseek chat --json-mode --user "Return JSON data"

# Streaming
deepseek chat --stream --user "Tell me a story"

# With conversation history
deepseek chat --system "You are helpful" --user "Hi" --assistant "Hello!" --user "How are you?"

# Thinking mode
deepseek chat --thinking enabled --reasoning-effort high --user "Solve this problem"

# Beta features
deepseek chat --beta --prefix-completion --assistant "partial response" --user "continue"

# Tools
deepseek chat --tools '[{"type":"function","function":{"name":"weather","parameters":{"type":"object"}}}]' --user "What's the weather?"
```

### FIM Completions

```bash
# Basic FIM
deepseek fim --prompt "func main() {" --suffix "}"

# With parameters
deepseek fim --prompt "func main() {" --suffix "}" --max-tokens 128 --temperature 0.2

# Streaming FIM
deepseek fim --stream --prompt "func main() {"

# With echo
deepseek fim --echo --prompt "func main() {"
```

### Model Management

```bash
# List available models
deepseek models

# Get account balance
deepseek balance
```

### Configuration Management

```bash
# View config file location
deepseek config

# Create sample config file
deepseek config init
```

## Chat Completion Flags

### Input Flags

- `--system string` - System message content
- `--user string` - User message content
- `--assistant string` - Assistant message content (for conversation history)
- `--json string` - Raw JSON input (bypasses individual flags)

### Model & Thinking

- `--model string` - Model to use (default: deepseek-v4-pro)
- `--thinking string` - Thinking mode: enabled/disabled (default: enabled)
- `--reasoning-effort string` - Reasoning effort: high/max (default: high)

### Sampling Parameters

- `--temperature float` - Sampling temperature (0.0 to 2.0, default: 1.0)
- `--top-p float` - Nucleus sampling threshold (0.0 to 1.0, default: 1.0)
- `--max-tokens int` - Maximum tokens to generate (0 = no limit)
- `--frequency-penalty float` - Frequency penalty (-2.0 to 2.0, default: 0.0)
- `--presence-penalty float` - Presence penalty (-2.0 to 2.0, default: 0.0)

### Output Format

- `--json-mode` - Enable JSON mode (response_format: json_object)
- `--stop strings` - Stop sequences (up to 16 strings)

### Streaming

- `--stream` - Enable streaming responses
- `--include-usage` - Include usage info in streaming responses

### Tools

- `--tools string` - Tools JSON array
- `--tool-choice string` - Tool choice: none, auto, required, or JSON for function

### Logprobs

- `--logprobs` - Return log probabilities
- `--top-logprobs int` - Number of top log probabilities to return (0-20)

### Beta Features

- `--beta` - Use beta endpoint (https://api.deepseek.com/beta)
- `--prefix-completion` - Enable prefix completion (beta feature)

### Other

- `--base-url string` - Override base URL for this request

## FIM Completion Flags

- `--prompt string` - Code prefix before cursor (required)
- `--suffix string` - Code suffix after cursor
- `--model string` - Model to use for FIM (default: deepseek-v4-pro)
- `--max-tokens int` - Maximum tokens to generate (max 4096 for FIM, default: 128)
- `--temperature float` - Sampling temperature (default: 0.2)
- `--top-p float` - Nucleus sampling threshold (default: 1.0)
- `--frequency-penalty float` - Frequency penalty (-2.0 to 2.0)
- `--presence-penalty float` - Presence penalty (-2.0 to 2.0)
- `--stop strings` - Stop sequences (up to 16 strings)
- `--stream` - Enable streaming responses
- `--include-usage` - Include usage info in streaming responses
- `--echo` - Echo back the prompt with completion
- `--logprobs int` - Return log probabilities (0-20)
- `--beta` - Use beta endpoint (default true for FIM)
- `--base-url string` - Override base URL for this request

## Response Format

### Non-Streaming Responses

The CLI formats responses for readability:

```
Response content here

[finish_reason: stop]

[usage: prompt_tokens=10, completion_tokens=20, total_tokens=30]
```

### Streaming Responses

**Important**: When using `--stream`, the CLI does **NOT** output pure JSON. Instead, it:

1. **Parses SSE chunks** from the API (format: `data: {...}`)
2. **Extracts content** from each JSON chunk in real-time
3. **Outputs plain text content** as it's generated
4. **Displays metadata** (finish_reason, usage) at the end

**Streaming output format:**

```
Streaming content appears here as it's generated...
[finish_reason: stop]
[usage: prompt_tokens=10, completion_tokens=20, total_tokens=30]
```

### SSE Format (Internal)

The API returns Server-Sent Events (SSE) when streaming is enabled. Each chunk is a JSON object prefixed with `data:`:

```
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: [DONE]
```

The CLI parses these SSE chunks internally and outputs the extracted content in real-time, with metadata (finish_reason, usage) displayed at the end.

### Raw JSON Mode

If you need raw JSON output (including streaming), use the `--json` flag with a JSON payload instead of individual flags. This will output the raw API response without CLI formatting.

## Examples

### Simple Chat

```bash
export DEEPSEEK_API_KEY="your-key"
deepseek chat --user "What is the capital of France?"
```

### Code Completion

```bash
deepseek fim --prompt "func add(a, b int) int {" --suffix "}"
```

### Streaming with Usage

```bash
deepseek chat --stream --include-usage --user "Write a short poem"
```

### JSON Mode

```bash
deepseek chat --json-mode --user "Return a JSON object with name and age fields"
```

## Development

### Building

```bash
# Build for current platform
go build -o deepseek

# Build with optimization
go build -ldflags="-s -w" -o deepseek

# Using Taskfile
task build
```

### Testing

```bash
# Run tests
go test ./...

# Test with validation
DEEPSEEK_API_KEY=test deepseek chat --thinking invalid --user "test"
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:

- GitHub Issues: [https://github.com/shaoyanji/deepseek-cli/issues](https://github.com/shaoyanji/deepseek-cli/issues)
- DeepSeek API Docs: [https://api-docs.deepseek.com/](https://api-docs.deepseek.com/)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- Styling with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Markdown rendering with [Glamour](https://github.com/charmbracelet/glamour)
- Powered by [DeepSeek API](https://api.deepseek.com/)
