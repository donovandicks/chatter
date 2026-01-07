package ui

import (
	"strings"

	"github.com/donovandicks/chatter/internal/fsutil"

	tea "github.com/charmbracelet/bubbletea"
)

// Autocomplete manages the state and logic for file path autocompletion in the chat input.
type Autocomplete struct {
	allFiles      []string
	suggestions   []string
	suggestionIdx int
	Active        bool
}

// NewAutocomplete creates a new Autocomplete instance and initializes the file list.
func NewAutocomplete() *Autocomplete {
	files, _ := fsutil.ListFiles(".")
	return &Autocomplete{
		allFiles: files,
		Active:   false,
	}
}

// Update handles key events for navigation and selection, and filters suggestions based on input.
func (a *Autocomplete) Update(msg tea.Msg, inputVal string, cursor int) (bool, string, int) {
	if a.Active {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyUp:
				if a.suggestionIdx > 0 {
					a.suggestionIdx--
				}
				return true, "", 0
			case tea.KeyDown:
				if a.suggestionIdx < len(a.suggestions)-1 {
					a.suggestionIdx++
				}
				return true, "", 0
			case tea.KeyEnter, tea.KeyTab:
				if len(a.suggestions) > 0 {
					selected := a.suggestions[a.suggestionIdx]
					// Find the start of the current @mention
					start := strings.LastIndex(inputVal[:cursor], "@")
					if start != -1 {
						newValue := inputVal[:start] + selected + " " + inputVal[cursor:]
						newCursor := start + len(selected) + 1
						a.Active = false
						return true, newValue, newCursor
					}
				}
				a.Active = false
				return true, "", 0
			case tea.KeyEsc:
				a.Active = false
				return true, "", 0
			}
		}
	}

	// Check for trigger to show/update suggestions
	lastAt := strings.LastIndex(inputVal[:cursor], "@")

	if lastAt != -1 {
		// potential mention, check if there are spaces between @ and cursor
		query := inputVal[lastAt+1 : cursor]
		if !strings.Contains(query, " ") {
			a.suggestions = fsutil.FilterFiles(a.allFiles, query)
			if len(a.suggestions) > 0 {
				a.Active = true
				// Keep index in bounds if list shrinks
				if a.suggestionIdx >= len(a.suggestions) {
					a.suggestionIdx = 0
				}
			} else {
				a.Active = false
			}
		} else {
			a.Active = false
		}
	} else {
		a.Active = false
	}

	return false, "", 0
}

// View renders the autocomplete suggestion list.
func (a *Autocomplete) View() string {
	if !a.Active {
		return ""
	}

	var views []string
	// Window size
	windowSize := 5

	// Ensure the selected index is visible
	start := 0
	if a.suggestionIdx >= windowSize {
		start = a.suggestionIdx - windowSize + 1
	}

	end := start + windowSize
	if end > len(a.suggestions) {
		end = len(a.suggestions)
		// Adjust start if we hit the bottom but have space at the top
		start = max(end-windowSize, 0)
	}

	for i, s := range a.suggestions[start:end] {
		idx := start + i
		if idx == a.suggestionIdx {
			views = append(views, selectedSuggestionStyle.Render("> "+s))
		} else {
			views = append(views, suggestionStyle.Render("  "+s))
		}
	}

	// Wrap suggestions in a container
	content := strings.Join(views, "\n")
	return "\n" + suggestionContainerStyle.Render(content)
}
