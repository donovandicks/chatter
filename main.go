package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/donovandicks/chatter/ai"
	"github.com/donovandicks/chatter/internal/auth"
	"github.com/donovandicks/chatter/ui"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()

	// Initialize permission system
	permManager := auth.NewPermissionManager()
	permRequester := ui.NewUIPermissionRequester(permManager)

	// Create Agent Configuration
	agent, err := ai.NewMainAgent(ctx, permManager, permRequester)
	if err != nil {
		log.Fatal(err)
	}

	// Create Session
	session := agent.NewSession("cli-session")

	p := tea.NewProgram(ui.NewModel(session, permRequester))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(ui.RenderStats(session.Stats, true))
}