# Ideas

General list of ideas to implement.

## Automatic Decision Documentation

Summarize the result of a coding session into a document.

Include the following:

- Key architecture and implementation choices
- Key tradeoffs and rationale for decisions
- Insights into debugging and edge cases
- Workflows or tooling that can be reused for future development or debugging
- Learnings about the codebase and associated tooling

## Hooks

Configurable hooks that can be triggered on specific events.

Examples:

```yaml
---
type: command
description: Auto format
trigger: 
  type: ToolUse
  matching: Write|Edit
  timing: After
command: |
  gofumpt ./...

---
type: prompt
description: Trigger Review
trigger:
  type: Stop
  timing: After
prompt: |
  Review the most recent changes to the project.
```

## Deep Introspection

Expose conversation history, usage statistics, and all other information to the
user via slash commands and other tooling.

Particularly interesting would be:

- Allow the user to manually edit the conversation history (context).
- Allow users to select specific content for fine-grained compression.

## Session Export

Serialize and export the session contents to a file.

- File should be structured and clean for post-processing.
- Restarting from a session file.

## Model Selection

- Manual - user selects model for the response.
- Automatic - system routes the request to the appropriate model.

## Diff Reviews

Display intended edit diffs to the user.

- Semantic diff could be interesting
  - <https://github.com/Wilfred/difftastic>
  - <https://github.com/afnanenayet/diffsitter>

## Modal Interactions

Allow the user to select a "mode" for the interaction.

- Planner -> no code changes, only planning.
- Coding -> only implementation.
- Dynamic -> agent decides whether to plan/document/implement.

## Message Queueing

Allow the user to queue up messages to be included in the next interaction.

## Directed Compaction/Compression

- Prompt the model with guided compression to prioritize specific context/history
- Suggest options to the user based on the history, e.g. "themes" or possible directions
  that the conversation can continue in
