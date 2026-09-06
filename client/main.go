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

// -----------------------------------------------------------------------------
// UI Styling & Helpers (Matched from original main.go)
// -----------------------------------------------------------------------------

var (
	colorPink    = lipgloss.Color("#F38BA8")
	colorMauve   = lipgloss.Color("#CBA6F7")
	colorGreen   = lipgloss.Color("#A6E3A1")
	colorYellow  = lipgloss.Color("#F9E2AF")
	colorPeach   = lipgloss.Color("#FAB387")
	colorCyan    = lipgloss.Color("#89DCEB")
	colorSubtext = lipgloss.Color("#A6ADC8")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorMauve).
			Padding(0, 1)

	nowPlayingBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)
)

func padWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

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
	cursor    int
	status    statusInfo
	message   string
	favorites map[string]bool
}

func initialModel() model {
	return model{
		favorites: loadFavorites(),
	}
}

// loadFavorites reads the favorites synced by the main iradio app
func loadFavorites() map[string]bool {
	favs := make(map[string]bool)
	usr, _ := user.Current()
	configDir := filepath.Join(usr.HomeDir, ".config", "iradio")
	data, err := os.ReadFile(filepath.Join(configDir, "favorites.json"))
	if err == nil {
		json.Unmarshal(data, &favs)
	}
	return favs
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

	b.WriteString(titleStyle.Render("📻 IRADIO REMOTE CLIENT") + " " +
		lipgloss.NewStyle().Foreground(colorSubtext).Render(" • Connected via playerctl"))
	b.WriteString("\n\n")

	if len(allStations) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSubtext).Render("No stations found. Run the main 'iradio' app first to generate the list.\n\n"))
	}

	// Station List
	for i, st := range allStations {
		cursor := "  "
		if m.cursor == i {
			cursor = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("❯ ")
		}

		favStar := "  "
		if m.favorites[st.ID] {
			favStar = lipgloss.NewStyle().Foreground(colorYellow).Render("★ ")
		} else {
			favStar = lipgloss.NewStyle().Foreground(colorSubtext).Render("☆ ")
		}

		playBadge := "       "
		isPlaying := m.status.status == "Playing" && (st.NameZh+" ("+st.NameEn+")") == m.status.title
		if isPlaying {
			playBadge = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("▶ PLAY ")
		}

		var regionTag string
		switch st.Region {
		case "TW":
			regionTag = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("[TW] ")
		case "JP":
			regionTag = lipgloss.NewStyle().Foreground(colorPink).Bold(true).Render("[JP] ")
		case "SG":
			regionTag = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("[SG] ")
		case "MY":
			regionTag = lipgloss.NewStyle().Foreground(colorPeach).Bold(true).Render("[MY] ")
		case "HK":
			regionTag = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("[HK] ")
		default:
			regionTag = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(fmt.Sprintf("[%s] ", padWidth(st.Region, 2)))
		}

		idCol := padWidth(st.ID, 6)
		zhCol := padWidth(st.NameZh, 22)
		enCol := padWidth(st.NameEn, 28)
		dialCol := lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("(%s)", st.Dial))

		var textStyle lipgloss.Style
		if m.cursor == i {
			textStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
		} else {
			textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
		}

		rowText := textStyle.Render(fmt.Sprintf("%s %s %s", idCol, zhCol, enCol))
		line := fmt.Sprintf("%s%s%s%s%s %s", cursor, favStar, playBadge, regionTag, rowText, dialCol)
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// Now Playing Card
	var cardLine1, cardLine2 string
	lblTitle := lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render(padWidth("▶ Title:", 12))
	lblNotice := lipgloss.NewStyle().Foreground(colorYellow).Render(padWidth("  Notice:", 12))

	if m.status.err != nil {
		cardLine1 = lblTitle + lipgloss.NewStyle().Foreground(colorSubtext).Render("Disconnected")
		cardLine2 = lblNotice + lipgloss.NewStyle().Foreground(colorSubtext).Render("Is the main iradio app running?")
	} else {
		cardLine1 = lblTitle + lipgloss.NewStyle().Bold(true).Render(m.status.title)
		cardLine2 = lblNotice + lipgloss.NewStyle().Foreground(colorYellow).Render(m.message)
	}

	nowPlayingInfo := fmt.Sprintf("%s\n%s\n", cardLine1, cardLine2)
	// Dynamic width based on content
	maxW := 60
	playerCard := nowPlayingBox.Width(maxW).Height(2).Render(nowPlayingInfo)
	b.WriteString(playerCard)
	b.WriteString("\n\n")

	// Footer
	footerText := "[Enter/Space] Play • [p] Pause • [n/b] Next/Prev • [s] Stop • [q] Quit"
	b.WriteString(helpStyle.Render(footerText) + "\n")

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
