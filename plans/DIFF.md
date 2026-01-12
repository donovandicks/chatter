# Plan: Edit Diffs

**Objective:** Enhance the `write_file` tool and UI to generate and display color-coded diffs during permission requests, allowing users to review changes before execution.

## Phase 1: Dependencies & Core Structure
1.  **Add Dependency:** Add `github.com/hexops/gotextdiff` to `go.mod` to handle text comparison and unified diff generation.
2.  **Update `Action` Struct:** Modify the `Action` struct in `internal/auth/permissions.go` to include a `Diff` field (type `string`) for transporting the diff data to the UI.

## Phase 2: Tool Logic (`ai/tools/write_file.go`)
1.  **Enhance `RequestPermission`:**
    *   Update the `RequestPermission` method in `WriteFile`.
    *   Implement logic to read the existing file content (if it exists).
    *   Compare the existing content with the new content (provided in arguments).
    *   Generate a unified diff.
    *   Populate the new `Diff` field in the returned `auth.Action`.

## Phase 3: UI Implementation (`ui/permissions.go`)
1.  **Update `RenderPermissionInline`:**
    *   Modify the rendering logic to check for the presence of the `Diff` field.
    *   If a diff exists, parse the string line-by-line.
    *   **Apply Styling (KR003):**
        *   Lines starting with `+` (additions) -> **Green** (`lipgloss.Color("42")`).
        *   Lines starting with `-` (deletions) -> **Red** (`lipgloss.Color("196")`).
        *   Context lines -> Neutral/Grey.
    *   Integrate this diff view into the existing permission dialog box, ensuring it fits within the TUI layout.

## Phase 4: Verification
1.  **Manual Verification:**
    *   Start the agent.
    *   Ask it to modify an existing file.
    *   Verify the permission dialog shows the correct additions/deletions in the correct colors.
    *   Ask it to create a new file (should show all green/additions).
2.  **Test Updates:** Ensure existing tests in `ai/tools/write_file_test.go` still pass and potentially add a test case for diff generation.
