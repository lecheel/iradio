// Package main is the entry point for iradio.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"iradio/internal/config"
	"iradio/internal/mpris"
	"iradio/internal/ui"
)

func main() {
	config.WriteExampleStations()

	m := ui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())

	svc := mpris.Start(p)
	m.SetService(svc)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running player: %v\n", err)
		os.Exit(1)
	}

	m.Stop()
}
