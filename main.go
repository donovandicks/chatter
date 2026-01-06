package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/donovandicks/chatter/ai"
	"github.com/donovandicks/chatter/ui"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()
	agent, err := ai.NewAgent(ctx)
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(ui.NewModel(agent))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}