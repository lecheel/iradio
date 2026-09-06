package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Station struct {
	ID        string `json:"id"`
	Region    string `json:"region"`
	NameZh    string `json:"name_zh"`
	NameEn    string `json:"name_en"`
	Dial      string `json:"dial"`
	Desc      string `json:"desc"`
	StreamURL string `json:"stream_url"`
}

var allStations []Station

// loadStations reads the station list created/synced by the main iradio app
func loadStations() {
	usr, _ := user.Current()
	configDir := filepath.Join(usr.HomeDir, ".config", "iradio")

	// Try loading custom stations first
	data, err := os.ReadFile(filepath.Join(configDir, "stations.json"))
	if err == nil {
		json.Unmarshal(data, &allStations)
	}

	// If no custom stations, fallback to a default example list so the client isn't empty
	if len(allStations) == 0 {
		allStations = []Station{
			{"R1", "HK", "香港電台第一台", "RTHK Radio 1", "FM 92.6 - 94.4 MHz", "News (Cantonese)", "https://rthkaudio1-lh.akamaihd.net/i/radio1_1@355864/master.m3u8"},
			{"HITFM", "TW", "Hit FM 聯播網", "Hit FM 107.7", "FM 107.7 MHz", "Taiwan's #1 Hit Music", "https://www.hitoradio.com/newweb/hichannel.php?channelID=1&action=getLIVEURL"},
			{"YES933", "SG", "YES 933 頂尖流行音樂台", "YES 933 FM", "FM 93.3 MHz", "Mandarin Hit Music", "https://playerservices.streamtheworld.com/api/livestream-redirect/YES933AAC.aac"},
		}
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type statusInfo struct {
	status string
	title  string
	err    error
}

type model struct {
	cursor  int
	status  statusInfo
	message string
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.status = getStatus()
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(allStations)-1 {
				m.cursor++
			}

		case "enter", " ":
			st := allStations[m.cursor]
			runPlayerCtl("open", st.StreamURL)
			m.message = fmt.Sprintf("Requested: %s", st.NameEn)

		case "p":
			runPlayerCtl("play-pause")
			m.message = "Toggled Play/Pause"

		case "s":
			runPlayerCtl("stop")
			m.message = "Stopped"

		case "n":
			runPlayerCtl("next")
			m.message = "Next Station"

		case "b":
			runPlayerCtl("previous")
			m.message = "Previous Station"
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F38BA8"))
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89DCEB"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6ADC8"))
	statusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))

	b.WriteString(titleStyle.Render("📻 IRADIO REMOTE CLIENT") + "\n\n")

	// Station List
	for i, st := range allStations {
		cursor := "  "
		if m.cursor == i {
			cursor = selStyle.Render("❯ ")
		}

		region := dimStyle.Render(fmt.Sprintf("[%-2s]", st.Region))
		name := fmt.Sprintf("%-25s", st.NameEn)
		dial := dimStyle.Render(st.Dial)

		b.WriteString(fmt.Sprintf("%s %s %s %s\n", cursor, region, name, dial))
	}

	b.WriteString("\n" + strings.Repeat("─", 50) + "\n")

	// Status Panel
	if m.status.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", m.status.err))
		b.WriteString(dimStyle.Render("Is the main iradio app running?\n"))
	} else {
		b.WriteString(fmt.Sprintf("Status: %s\n", statusStyle.Render(m.status.status)))
		b.WriteString(fmt.Sprintf("Title:  %s\n", m.status.title))
	}

	// Message or Controls
	if m.message != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", dimStyle.Render(m.message)))
	} else {
		b.WriteString(dimStyle.Render("\n[Enter/Space] Play • [p] Pause • [n/b] Next/Prev • [q] Quit\n"))
	}

	return b.String()
}

func runPlayerCtl(args ...string) {
	cmdArgs := append([]string{"-p", "iradio"}, args...)
	_ = exec.Command("playerctl", cmdArgs...).Run()
}

func getStatus() statusInfo {
	var s statusInfo
	out, err := exec.Command("playerctl", "-p", "iradio", "status").Output()
	if err != nil {
		s.err = fmt.Errorf("cannot connect to main player")
		return s
	}
	s.status = strings.TrimSpace(string(out))

	out, err = exec.Command("playerctl", "-p", "iradio", "metadata", "title").Output()
	if err == nil {
		s.title = strings.TrimSpace(string(out))
	}

	return s
}

func main() {
	loadStations()
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running client:", err)
		os.Exit(1)
	}
}
