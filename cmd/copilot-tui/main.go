package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"copilot-tui/internal/app"
)

func main() {
	program := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if err := program.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
