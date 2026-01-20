# AI Package (`ai`)

The `ai` package contains the core logic for the Chatter agent. It manages the
interaction between the user, the tools, and the underlying LLM.

## Architecture

The system is built around a few key components:

* **Agent (`Agent`):** The immutable configuration of the assistant. It defines the model to use, the system prompt, and the available tools.
* **Session (`Session`):** Represents an active conversation. It maintains the chat history and manages the turn-taking loop (User input -> LLM -> Tool Call -> LLM -> Response).
* **Provider (`Provider`):** An abstraction layer over the LLM API. Currently, `GeminiProvider` implements this for Google's GenAI SDK.
* **Middleware (`ToolMiddleware`):** A chaining mechanism to intercept tool executions. This is primarily used for the Permission System (`NewPermissionMiddleware`), ensuring users authorize sensitive actions.

## Key Files

* **`agent.go`**: Defines the `Agent` struct and factory functions like `NewMainAgent`.
* **`session.go`**: Handles the conversation loop and context management.
* **`provider.go`**: Defines the `Provider` interface.
* **`gemini_provider.go`**: Implementation of the provider for Google Gemini.
* **`middleware.go`**: Implements the middleware pattern for tool execution, including the permission check logic.

## Subdirectories

### `tools/`

Contains the definitions and implementations of tools the agent can use.

* **Interface:** `FunctionTool` (must implement `Decl`, `Run`, and `RequestPermission`).
* **Examples:** `ReadFile`, `WriteFile`.

### `skills/`

Contains logic for specialized agent skills, currently supporting YAML-based skill definitions.

## extending the Agent

### Adding a New Tool

To add a new capability to the agent:

1. Create a new file in `ai/tools/`.
2. Define a struct for your tool.
3. Implement the `FunctionTool` interface:
    * `Decl()`: Returns the function schema for the LLM.
    * `Run()`: The actual logic.
    * `RequestPermission()`: Returns the `auth.Action` required for this operation.
4. Register the tool in `NewMainAgent` within `ai/agent.go`.

### Changing the Model

The model version is currently hardcoded in `NewMainAgent` in `ai/agent.go`.
