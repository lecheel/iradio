package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

// Station represents an RTHK broadcast channel.
type Station struct {
	ID       string
	NameZh   string
	NameEn   string
	Dial     string
	Desc     string
	StreamURL string
}

var rthkStations = []Station{
	{
		ID:        "R1",
		NameZh:    "香港電台第一台",
		NameEn:    "RTHK Radio 1",
		Dial:      "FM 92.6 - 94.4 MHz",
		Desc:      "News, Current Affairs & Public Services (Cantonese)",
		StreamURL: "https://rthkaudio1-lh.akamaihd.net/i/radio1_1@355864/master.m3u8",
	},
	{
		ID:        "R2",
		NameZh:    "香港電台第二台",
		NameEn:    "RTHK Radio 2",
		Dial:      "FM 94.8 - 96.9 MHz",
		Desc:      "Youth, Pop Music & Community Promotion (Cantonese)",
		StreamURL: "https://rthkaudio2-lh.akamaihd.net/i/radio2_1@355865/master.m3u8",
	},
	{
		ID:        "R3",
		NameZh:    "香港電台第三台",
		NameEn:    "RTHK Radio 3",
		Dial:      "AM 567 / AM 1584 kHz / FM 97.9 - 106.8 MHz",
		Desc:      "News, Lifestyle & Music (English)",
		StreamURL: "https://rthkaudio3-lh.akamaihd.net/i/radio3_1@355866/master.m3u8",
	},
	{
		ID:        "R4",
		NameZh:    "香港電台第四台",
		NameEn:    "RTHK Radio 4",
		Dial:      "FM 97.6 - 98.9 MHz",
		Desc:      "Fine Music & Classical Arts (Bilingual)",
		StreamURL: "https://rthkaudio4-lh.akamaihd.net/i/radio4_1@355867/master.m3u8",
	},
	{
		ID:        "R5",
		NameZh:    "香港電台第五台",
		NameEn:    "RTHK Radio 5",
		Dial:      "AM 783 kHz / FM 92.3 - 106.8 MHz",
		Desc:      "Cultural Heritage, Elderly & Chinese Opera (Cantonese)",
		StreamURL: "https://rthkaudio5-lh.akamaihd.net/i/radio5_1@355868/master.m3u8",
	},
	{
		ID:        "PT",
		NameZh:    "香港電台普通話台",
		NameEn:    "RTHK Putonghua Channel",
		Dial:      "AM 621 kHz / FM 100.9 - 103.3 MHz",
		Desc:      "News, Finance & Chinese Culture (Mandarin)",
		StreamURL: "https://rthkaudiopth-lh.akamaihd.net/i/radiopth_1@355869/master.m3u8",
	},
	{
		ID:        "R6",
		NameZh:    "香港電台第六台 (CNR)",
		NameEn:    "RTHK Radio 6 / Voice of HK",
		Dial:      "AM 675 kHz",
		Desc:      "Relay of China National Radio Voice of Hong Kong",
		StreamURL: "https://rthkaudio6cnr-lh.akamaihd.net/i/radio6cnr_1@575604/master.m3u8",
	},
}

// -----------------------------------------------------------------------------
// Audio Stream Backend
// -----------------------------------------------------------------------------

type AudioPlayer struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (p *AudioPlayer) Play(url string) error {
	p.Stop()

	// Check available players
	var playerBin string
	var args []string
	if _, err := exec.LookPath("mpv"); err == nil {
		playerBin = "mpv"
		args = []string{"--no-video", "--really-quiet", "--network-timeout=15", url}
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		playerBin = "ffplay"
		args = []string{"-nodisp", "-autoexit", "-loglevel", "quiet", url}
	} else {
		return fmt.Errorf("neither 'mpv' nor 'ffplay' found in PATH")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()

	cmd := exec.CommandContext(ctx, playerBin, args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func (p *AudioPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}

// -----------------------------------------------------------------------------
// MPRIS2 Server
// -----------------------------------------------------------------------------

type MPRISActionMsg struct {
	Action string
}

type mprisRoot struct {
	prog *tea.Program
}

func (m *mprisRoot) Raise() *dbus.Error { return nil }
func (m *mprisRoot) Quit() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(tea.Quit())
	}
	return nil
}

type mprisPlayer struct {
	prog *tea.Program
}

func (m *mprisPlayer) Next() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "next"})
	}
	return nil
}

func (m *mprisPlayer) Previous() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "previous"})
	}
	return nil
}

func (m *mprisPlayer) Pause() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "pause"})
	}
	return nil
}

func (m *mprisPlayer) PlayPause() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "playpause"})
	}
	return nil
}

func (m *mprisPlayer) Stop() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "stop"})
	}
	return nil
}

func (m *mprisPlayer) Play() *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "play"})
	}
	return nil
}

func (m *mprisPlayer) Seek(offset int64) *dbus.Error                           { return nil }
func (m *mprisPlayer) SetPosition(trackId dbus.ObjectPath, pos int64) *dbus.Error { return nil }
func (m *mprisPlayer) OpenUri(uri string) *dbus.Error                         { return nil }

type MPRISService struct {
	conn       *dbus.Conn
	properties *prop.Properties
	active     bool
}

func startMPRIS(p *tea.Program) *MPRISService {
	conn, err := dbus.SessionBus()
	if err != nil {
		return &MPRISService{active: false}
	}

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.rthk", dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return &MPRISService{active: false}
	}

	root := &mprisRoot{prog: p}
	player := &mprisPlayer{prog: p}

	_ = conn.Export(root, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2")
	_ = conn.Export(player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")

	propsSpec := prop.Map{
		"org.mpris.MediaPlayer2": {
			"CanQuit":             {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitTrue},
			"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitTrue},
			"Identity":            {Value: "RTHK Radio", Writable: false, Emit: prop.EmitTrue},
			"SupportedUriSchemes": {Value: []string{"http", "https"}, Writable: false, Emit: prop.EmitTrue},
			"SupportedMimeTypes":  {Value: []string{"audio/mpeg", "application/x-mpegurl"}, Writable: false, Emit: prop.EmitTrue},
		},
		"org.mpris.MediaPlayer2.Player": {
			"PlaybackStatus": {Value: "Stopped", Writable: false, Emit: prop.EmitTrue},
			"Rate":           {Value: 1.0, Writable: false, Emit: prop.EmitTrue},
			"Metadata":       {Value: map[string]dbus.Variant{}, Writable: false, Emit: prop.EmitTrue},
			"Volume":         {Value: 1.0, Writable: false, Emit: prop.EmitTrue},
			"CanControl":     {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanPause":       {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanGoPrevious":  {Value: true, Writable: false, Emit: prop.EmitTrue},
			"CanSeek":        {Value: false, Writable: false, Emit: prop.EmitTrue},
		},
	}

	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propsSpec)
	if err != nil {
		return &MPRISService{active: false}
	}

	return &MPRISService{
		conn:       conn,
		properties: props,
		active:     true,
	}
}

func (m *MPRISService) Update(status string, s *Station) {
	if !m.active || m.properties == nil {
		return
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "PlaybackStatus", dbus.MakeVariant(status))

	meta := map[string]dbus.Variant{}
	if s != nil {
		meta["mpris:trackid"] = dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/track/0"))
		meta["xesam:title"] = dbus.MakeVariant(s.NameZh + " (" + s.NameEn + ")")
		meta["xesam:artist"] = dbus.MakeVariant([]string{"RTHK 香港電台"})
		meta["xesam:album"] = dbus.MakeVariant(s.Dial)
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(meta))
}

// -----------------------------------------------------------------------------
// Bubbletea TUI Model
// -----------------------------------------------------------------------------

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type Model struct {
	width       int
	height      int
	activeTab   int // 0 = Stations, 1 = Favorites
	cursor      int
	favCursor   int
	favorites   map[string]bool
	audio       *AudioPlayer
	mpris       *MPRISService
	playingIdx  int // index in rthkStations, or -1 if stopped
	isPlaying   bool
	statusMsg   string
	bars        []int
}

func initialModel() Model {
	favs := loadFavorites()
	return Model{
		activeTab:  0,
		cursor:     0,
		favCursor:  0,
		favorites:  favs,
		audio:      &AudioPlayer{},
		playingIdx: -1,
		isPlaying:  false,
		bars:       make([]int, 14),
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		if m.isPlaying {
			for i := range m.bars {
				m.bars[i] = rand.Intn(7)
			}
		} else {
			for i := range m.bars {
				m.bars[i] = 0
			}
		}
		cmds = append(cmds, tickCmd())

	case MPRISActionMsg:
		switch msg.Action {
		case "playpause":
			m.togglePlay()
		case "play":
			if !m.isPlaying {
				m.togglePlay()
			}
		case "pause", "stop":
			if m.isPlaying {
				m.togglePlay()
			}
		case "next":
			m.selectNextStation()
		case "previous":
			m.selectPrevStation()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.audio.Stop()
			if m.mpris != nil {
				m.mpris.Update("Stopped", nil)
			}
			return m, tea.Quit

		case "1":
			m.activeTab = 0
		case "2":
			m.activeTab = 1
			m.favCursor = 0
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			if m.activeTab == 1 {
				m.favCursor = 0
			}

		case "up", "k":
			if m.activeTab == 0 {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				if m.favCursor > 0 {
					m.favCursor--
				}
			}

		case "down", "j":
			if m.activeTab == 0 {
				if m.cursor < len(rthkStations)-1 {
					m.cursor++
				}
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 && m.favCursor < len(favs)-1 {
					m.favCursor++
				}
			}

		case "f":
			target := m.getCurrentStation()
			if target != nil {
				if m.favorites[target.ID] {
					delete(m.favorites, target.ID)
					m.statusMsg = fmt.Sprintf("Removed %s from favorites", target.NameEn)
				} else {
					m.favorites[target.ID] = true
					m.statusMsg = fmt.Sprintf("Added %s to favorites", target.NameEn)
				}
				saveFavorites(m.favorites)
			}

		case " ":
			m.togglePlay()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) getCurrentStation() *Station {
	if m.activeTab == 0 {
		if m.cursor >= 0 && m.cursor < len(rthkStations) {
			return &rthkStations[m.cursor]
		}
	} else {
		favs := m.getFavoritesList()
		if len(favs) > 0 && m.favCursor < len(favs) {
			return &favs[m.favCursor]
		}
	}
	return nil
}

func (m *Model) getFavoritesList() []Station {
	var list []Station
	for _, st := range rthkStations {
		if m.favorites[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m *Model) togglePlay() {
	target := m.getCurrentStation()
	if target == nil {
		return
	}

	targetIdx := -1
	for idx, s := range rthkStations {
		if s.ID == target.ID {
			targetIdx = idx
			break
		}
	}

	// If already playing this station, stop it
	if m.isPlaying && m.playingIdx == targetIdx {
		m.audio.Stop()
		m.isPlaying = false
		m.statusMsg = fmt.Sprintf("Stopped: %s", target.NameEn)
		if m.mpris != nil {
			m.mpris.Update("Stopped", nil)
		}
		return
	}

	// Start playing
	err := m.audio.Play(target.StreamURL)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
		m.isPlaying = false
		return
	}

	m.isPlaying = true
	m.playingIdx = targetIdx
	m.statusMsg = fmt.Sprintf("Playing live: %s", target.NameEn)
	if m.mpris != nil {
		m.mpris.Update("Playing", target)
	}
}

func (m *Model) selectNextStation() {
	if len(rthkStations) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx + 1) % len(rthkStations)
	st := &rthkStations[m.playingIdx]
	_ = m.audio.Play(st.StreamURL)
	m.isPlaying = true
	if m.mpris != nil {
		m.mpris.Update("Playing", st)
	}
}

func (m *Model) selectPrevStation() {
	if len(rthkStations) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx - 1 + len(rthkStations)) % len(rthkStations)
	st := &rthkStations[m.playingIdx]
	_ = m.audio.Play(st.StreamURL)
	m.isPlaying = true
	if m.mpris != nil {
		m.mpris.Update("Playing", st)
	}
}

// -----------------------------------------------------------------------------
// UI Rendering with Lip Gloss
// -----------------------------------------------------------------------------

var (
	colorPink   = lipgloss.Color("#F38BA8")
	colorMauve  = lipgloss.Color("#CBA6F7")
	colorGreen  = lipgloss.Color("#A6E3A1")
	colorYellow = lipgloss.Color("#F9E2AF")
	colorSubtext= lipgloss.Color("#A6ADC8")
	colorBase   = lipgloss.Color("#1E1E2E")
	colorSurface= lipgloss.Color("#313244")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorMauve).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Background(colorSurface).
			Foreground(colorYellow).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorYellow).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
			Background(colorBase).
			Foreground(colorSubtext).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSurface).
			Padding(0, 2)

	nowPlayingBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)
)

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// 1. Header
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("📻 RTHK INTERNET RADIO (香港電台)"),
		lipgloss.NewStyle().Foreground(colorSubtext).Render(" • MPRIS2 Enabled"),
	)

	// 2. Tabs
	var tab1, tab2 string
	if m.activeTab == 0 {
		tab1 = activeTabStyle.Render("1: All Stations [1]")
		tab2 = inactiveTabStyle.Render("2: Favorites [2]")
	} else {
		tab1 = inactiveTabStyle.Render("1: All Stations [1]")
		tab2 = activeTabStyle.Render(fmt.Sprintf("2: Favorites (%d) [2]", len(m.getFavoritesList())))
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tab1, " ", tab2)

	// 3. Station List
	var listContent strings.Builder
	var stationsToRender []Station
	activeCursor := m.cursor

	if m.activeTab == 0 {
		stationsToRender = rthkStations
	} else {
		stationsToRender = m.getFavoritesList()
		activeCursor = m.favCursor
	}

	if len(stationsToRender) == 0 {
		listContent.WriteString(lipgloss.NewStyle().
			Foreground(colorSubtext).
			Padding(2, 0).
			Render("  No favorite stations added yet.\n  Press 'f' on any station in Tab 1 to add."))
	} else {
		for i, st := range stationsToRender {
			isSelected := (i == activeCursor)
			isThisPlaying := (m.isPlaying && m.playingIdx >= 0 && rthkStations[m.playingIdx].ID == st.ID)

			cursorMarker := "  "
			if isSelected {
				cursorMarker = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("❯ ")
			}

			favStar := "  "
			if m.favorites[st.ID] {
				favStar = lipgloss.NewStyle().Foreground(colorYellow).Render("★ ")
			} else {
				favStar = lipgloss.NewStyle().Foreground(colorSubtext).Render("☆ ")
			}

			playBadge := "       "
			if isThisPlaying {
				playBadge = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("▶ PLAY ")
			}

			rowTitle := fmt.Sprintf("%-2s %-16s %s", st.ID, st.NameZh, st.NameEn)
			if isSelected {
				rowTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render(rowTitle)
			} else {
				rowTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Render(rowTitle)
			}

			dialInfo := lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("(%s)", st.Dial))

			listContent.WriteString(fmt.Sprintf("%s%s%s %-46s %s\n", cursorMarker, favStar, playBadge, rowTitle, dialInfo))
		}
	}

	// 4. Equalizer / Now Playing Panel
	var eqBar strings.Builder
	eqChars := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇"}
	for _, val := range m.bars {
		eqBar.WriteString(eqChars[val])
	}

	nowPlayingInfo := "Status: Stopped"
	if m.isPlaying && m.playingIdx >= 0 {
		curr := rthkStations[m.playingIdx]
		eqStyled := lipgloss.NewStyle().Foreground(colorGreen).Render(eqBar.String())
		nowPlayingInfo = fmt.Sprintf("▶ Tuning: %s %s [%s]\n  %s  %s",
			curr.NameZh, curr.NameEn, curr.Dial, eqStyled, curr.Desc)
	}

	if m.statusMsg != "" {
		nowPlayingInfo += "\n  " + lipgloss.NewStyle().Foreground(colorYellow).Render("Notice: "+m.statusMsg)
	}

	playerCard := nowPlayingBox.Width(m.width - 4).Render(nowPlayingInfo)

	// 5. Help Footer
	footer := helpStyle.Render(
		"[Space] Play/Stop • [f] Favorite/Unfav • [1/2] Switch Tab • [↑/↓, j/k] Navigate • [q] Quit",
	)

	// Assemble full layout
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"\n",
		tabsRow,
		"\n",
		listContent.String(),
		"\n",
		playerCard,
		"\n",
		footer,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

// -----------------------------------------------------------------------------
// Favorites Persistence (~/.config/rthk-radio/favorites.json)
// -----------------------------------------------------------------------------

func getFavFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "rthk-radio")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "favorites.json")
}

func loadFavorites() map[string]bool {
	favs := make(map[string]bool)
	data, err := os.ReadFile(getFavFilePath())
	if err == nil {
		_ = json.Unmarshal(data, &favs)
	}
	return favs
}

func saveFavorites(favs map[string]bool) {
	data, err := json.MarshalIndent(favs, "", "  ")
	if err == nil {
		_ = os.WriteFile(getFavFilePath(), data, 0644)
	}
}

// -----------------------------------------------------------------------------
// Entry Point
// -----------------------------------------------------------------------------

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Initialize MPRIS2 session bus integration
	mpris := startMPRIS(p)
	m.mpris = mpris

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running player: %v\n", err)
		os.Exit(1)
	}

	// Terminate any running audio sub-processes on exit
	m.audio.Stop()
}
