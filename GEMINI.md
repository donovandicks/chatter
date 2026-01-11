# Chatter

Chatter is an experimental AI coding agent designed to run in the terminal. It leverages the Google GenAI SDK and a TUI (Text User Interface) built with Bubble Tea to provide an interactive coding assistant.

## Project Overview

* **Goal:** To create a capable, CLI-based AI agent that can understand codebase context, plan tasks, and execute file operations safely.
* **Key Features:**
  * **TUI:** Rich terminal interface using `charmbracelet/bubbletea`.
  * **AI Backend:** Powered by Google's Gemini models (Flash and Pro) via `google.golang.org/genai`.
  * **Agent Architecture:** Supports main agent and specialized sub-agents (e.g., "planner") via delegation.
  * **Safety:** Built-in permission system (`internal/auth`) to control file access and tool usage.

## Architecture

The project follows a standard Go project layout:

* **`main.go`**: The application entry point. Initializes the permission manager, the AI agent, and starts the TUI program.
* **`ai/`**: Contains the core AI logic.
  * `agent.go`: Manages the agent state, conversation history, and tool execution loop.
  * `tools/`: Definitions for tools the agent can use (e.g., `ReadFile`, `WriteFile`, `DelegateAgent`).
  * `agent_system.md` / `planner.md`: System prompts defining the agent's persona and capabilities.
* **`ui/`**: The presentation layer.
  * `model.go`: The main Bubble Tea model handling state and view updates.
  * `slash_commands.go`: Handles user input commands (e.g., `/help`, `/quit`).
* **`internal/`**: Private application code.
  * `auth/`: Handles permission logic (Ask, Grant Session, Reject).
  * `fsutil/`: File system utilities.

## Development

### Prerequisites

* **Go:** Version 1.25+
* **API Key:** A valid Google Gemini API key is required (usually set in a `.env` file).

### Commands

The project uses a `Makefile` to automate common tasks:

* **Clean:** `make clean` (Removes built binary and coverage files, if any)
* **Build:** `make build` (Outputs binary to `./chatter`)
* **Test:** `make test` (Runs all tests)
* **Coverage:** `make test-cov` (Runs all tests while collecting coverage data, outputs to `./coverage.out`)
* **Format:** `make fmt` (Applies `gofumpt`)
* **Lint:** `make lint` (Runs `gofumpt` check and `golangci-lint`)

### Coding Conventions

* **Style:** Strict adherence to `gofumpt`.
* **Error Handling:** Use `errors.Join(fmt.Errorf("..."), err)` for wrapping errors.
* **Logging:** Use `log/slog` for structured logging.
* **Concurrency:** Prefer channels and `select` for coordination; ensure `context.Context` is propagated.
* **Testing:** Use table-driven tests for logic and `t.Run()` for subtests.
  * Use `t.Parallel()` for slow but independent tests that can safely run in parallel.

## Dependencies

* **UI:** `github.com/charmbracelet/bubbletea` (and related libraries like `lipgloss`, `bubbles`)
* **AI:** `google.golang.org/genai`
* **Config:** `github.com/joho/godotenv`
