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

// Station represents a broadcast channel.
type Station struct {
	ID        string
	Region    string // "HK" or "TW"
	NameZh    string
	NameEn    string
	Dial      string
	Desc      string
	StreamURL string
}

var allStations = []Station{
	// --- Hong Kong (RTHK) ---
	{
		ID:        "R1",
		Region:    "HK",
		NameZh:    "香港電台第一台",
		NameEn:    "RTHK Radio 1",
		Dial:      "FM 92.6 - 94.4 MHz",
		Desc:      "News, Current Affairs & Public Services (Cantonese)",
		StreamURL: "https://rthkaudio1-lh.akamaihd.net/i/radio1_1@355864/master.m3u8",
	},
	{
		ID:        "R2",
		Region:    "HK",
		NameZh:    "香港電台第二台",
		NameEn:    "RTHK Radio 2",
		Dial:      "FM 94.8 - 96.9 MHz",
		Desc:      "Youth, Pop Music & Community Promotion (Cantonese)",
		StreamURL: "https://rthkaudio2-lh.akamaihd.net/i/radio2_1@355865/master.m3u8",
	},
	{
		ID:        "R3",
		Region:    "HK",
		NameZh:    "香港電台第三台",
		NameEn:    "RTHK Radio 3",
		Dial:      "AM 567 / AM 1584 kHz / FM 97.9 - 106.8 MHz",
		Desc:      "News, Lifestyle & Music (English)",
		StreamURL: "https://rthkaudio3-lh.akamaihd.net/i/radio3_1@355866/master.m3u8",
	},
	{
		ID:        "R4",
		Region:    "HK",
		NameZh:    "香港電台第四台",
		NameEn:    "RTHK Radio 4",
		Dial:      "FM 97.6 - 98.9 MHz",
		Desc:      "Fine Music & Classical Arts (Bilingual)",
		StreamURL: "https://rthkaudio4-lh.akamaihd.net/i/radio4_1@355867/master.m3u8",
	},
	{
		ID:        "R5",
		Region:    "HK",
		NameZh:    "香港電台第五台",
		NameEn:    "RTHK Radio 5",
		Dial:      "AM 783 kHz / FM 92.3 - 106.8 MHz",
		Desc:      "Cultural Heritage, Elderly & Chinese Opera (Cantonese)",
		StreamURL: "https://rthkaudio5-lh.akamaihd.net/i/radio5_1@355868/master.m3u8",
	},
	{
		ID:        "PTH",
		Region:    "HK",
		NameZh:    "香港電台普通話台",
		NameEn:    "RTHK Putonghua Channel",
		Dial:      "AM 621 kHz / FM 100.9 - 103.3 MHz",
		Desc:      "News, Finance & Chinese Culture (Mandarin)",
		StreamURL: "https://rthkaudiopth-lh.akamaihd.net/i/radiopth_1@355869/master.m3u8",
	},
	{
		ID:        "R6",
		Region:    "HK",
		NameZh:    "香港電台第六台 (CNR)",
		NameEn:    "RTHK Radio 6 / Voice of HK",
		Dial:      "AM 675 kHz",
		Desc:      "Relay of China National Radio Voice of Hong Kong",
		StreamURL: "https://rthkaudio6cnr-lh.akamaihd.net/i/radio6cnr_1@575604/master.m3u8",
	},

	// --- Taiwan (News98, UFO, ICRT, Bravo, Classical) ---
	{
		ID:        "N98",
		Region:    "TW",
		NameZh:    "九八新聞台",
		NameEn:    "News98 FM 98.1",
		Dial:      "FM 98.1 MHz",
		Desc:      "Taiwan 24/7 News, Finance, Analysis & Talk (Taipei)",
		StreamURL: "https://stream.rcs.revma.com/pntx1639ntzuv.m4a",
	},
	{
		ID:        "UFO",
		Region:    "TW",
		NameZh:    "飛碟聯播網",
		NameEn:    "UFO Radio FM 92.1",
		Dial:      "FM 92.1 MHz",
		Desc:      "Pop Music, Talk Shows & Lifestyle (Taiwan)",
		StreamURL: "https://stream.rcs.revma.com/em90w4aeewzuv.m4a",
	},
	{
		ID:        "ICRT",
		Region:    "TW",
		NameZh:    "台北國際社區廣播電台",
		NameEn:    "ICRT FM 100",
		Dial:      "FM 100.7 MHz",
		Desc:      "Taiwan's Premier English Radio Station",
		StreamURL: "https://stream.rcs.revma.com/nkdfurztxp3vv",
	},
	{
		ID:        "BRAVO",
		Region:    "TW",
		NameZh:    "台北都會音樂台",
		NameEn:    "Bravo FM 91.3",
		Dial:      "FM 91.3 MHz",
		Desc:      "Jazz, Classical & Urban Lifestyle (Taipei)",
		StreamURL: "https://onair.bravo913.com.tw:9130/live.mp3",
	},
	{
		ID:        "CFM",
		Region:    "TW",
		NameZh:    "好家庭古典音樂台",
		NameEn:    "Classical FM 97.7",
		Dial:      "FM 97.7 MHz",
		Desc:      "Classical Music, Philosophy & Arts (Taichung)",
		StreamURL: "https://onair.family977.com.tw:8977/live.mp3",
	},
	{
		ID:        "BCCN",
		Region:    "TW",
		NameZh:    "中廣新聞網",
		NameEn:    "BCC News Radio",
		Dial:      "AM 648 kHz / App",
		Desc:      "Taiwan BCC 24/7 Professional News Network",
		StreamURL: "https://stream.rcs.revma.com/78fm9wyy2tzuv",
	},
	{
		ID:        "BCCM",
		Region:    "TW",
		NameZh:    "中廣音樂網 (i radio)",
		NameEn:    "BCC i Radio / Music",
		Dial:      "FM 96.3 MHz / Web",
		Desc:      "Taiwan & Mandopop Non-stop Music Station",
		StreamURL: "https://stream.rcs.revma.com/ndk05tyy2tzuv",
	},
	{
		ID:        "BCCP",
		Region:    "TW",
		NameZh:    "中廣流行網",
		NameEn:    "BCC i like radio",
		Dial:      "FM 103.3 MHz",
		Desc:      "Taiwan Pop Culture, Lifestyle & Talk",
		StreamURL: "https://stream.rcs.revma.com/aw9uqyxy2tzuv",
	},

	// --- Japan (Shonan Beach, OTTAVA, FM Setagaya, AnimeNfo) ---
	{
		ID:        "SBFM",
		Region:    "JP",
		NameZh:    "湘南ビーチFM",
		NameEn:    "Shonan Beach FM",
		Dial:      "FM 78.9 MHz (Kanagawa)",
		Desc:      "Jazz, City Pop, Oldies & Coastal Vibes (Hayama/Zushi)",
		StreamURL: "https://shonanbeachfm.out.airtime.pro:8000/shonanbeachfm_a",
	},
	{
		ID:        "OTV",
		Region:    "JP",
		NameZh:    "OTTAVA 經典音樂台",
		NameEn:    "OTTAVA Classic Radio",
		Dial:      "Online (Tokyo)",
		Desc:      "Japan's Premier Classical Music & Arts Broadcast",
		StreamURL: "https://ottava2.out.airtime.pro/ottava2_a",
	},
	{
		ID:        "FMS834",
		Region:    "JP",
		NameZh:    "エフエム世田谷",
		NameEn:    "FM Setagaya 83.4",
		Dial:      "FM 83.4 MHz (Tokyo)",
		Desc:      "Tokyo Community Broadcast, News, Culture & Pop",
		StreamURL: "https://fmsetagaya834.out.airtime.pro/fmsetagaya834_a",
	},
	{
		ID:        "ANIFO",
		Region:    "JP",
		NameZh:    "AnimeNfo 動畫音樂台",
		NameEn:    "AnimeNfo Radio",
		Dial:      "Online / Tokyo",
		Desc:      "24/7 Anime OSTs, J-Pop, Vocaloid & Game Music",
		StreamURL: "https://stream.animenfo.com:8000/live",
	},

	// --- Singapore (YES 933, CNA938, Class 95, UFM 100.3, Kiss92) ---
	{
		ID:        "YES933",
		Region:    "SG",
		NameZh:    "YES 933 頂尖流行音樂台",
		NameEn:    "YES 933 FM",
		Dial:      "FM 93.3 MHz (Singapore)",
		Desc:      "Singapore's #1 Mandarin Hit Music Station (Mediacorp)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/YES933AAC.aac",
	},
	{
		ID:        "CNA938",
		Region:    "SG",
		NameZh:    "CNA938 新加坡新聞台",
		NameEn:    "CNA938 News Radio",
		Dial:      "FM 93.8 MHz (Singapore)",
		Desc:      "Singapore 24/7 News, Business & Current Affairs",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/938NOWAAC.aac",
	},
	{
		ID:        "CLS95",
		Region:    "SG",
		NameZh:    "Class 95 英語音樂台",
		NameEn:    "Class 95 FM",
		Dial:      "FM 95.0 MHz (Singapore)",
		Desc:      "The Best Mix of Music & Adult Contemporary",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/CLASS95AAC.aac",
	},
	{
		ID:        "UFM100",
		Region:    "SG",
		NameZh:    "UFM 100.3 流行音樂",
		NameEn:    "UFM 100.3 FM",
		Dial:      "FM 100.3 MHz (Singapore)",
		Desc:      "Top Mandopop Hits & Dynamic Morning Talk (SPH)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/UFM_1003AAC.aac",
	},
	{
		ID:        "KISS92",
		Region:    "SG",
		NameZh:    "Kiss92 英語流行台",
		NameEn:    "Kiss92 FM",
		Dial:      "FM 92.0 MHz (Singapore)",
		Desc:      "All The Great Songs In One Place (SPH)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/KISS_92AAC.aac",
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

func (m *mprisPlayer) Seek(offset int64) *dbus.Error                              { return nil }
func (m *mprisPlayer) SetPosition(trackId dbus.ObjectPath, pos int64) *dbus.Error { return nil }
func (m *mprisPlayer) OpenUri(uri string) *dbus.Error                             { return nil }

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
			"Identity":            {Value: "HK & Taiwan Internet Radio", Writable: false, Emit: prop.EmitTrue},
			"SupportedUriSchemes": {Value: []string{"http", "https"}, Writable: false, Emit: prop.EmitTrue},
			"SupportedMimeTypes":  {Value: []string{"audio/mpeg", "application/x-mpegurl", "audio/aac"}, Writable: false, Emit: prop.EmitTrue},
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
		broadcaster := "RTHK 香港電台"
		if s.Region == "TW" {
			broadcaster = s.NameZh + " (Taiwan)"
		} else if s.Region == "JP" {
			broadcaster = s.NameZh + " (Japan)"
		} else if s.Region == "SG" {
			broadcaster = s.NameZh + " (Singapore)"
		}
		meta["xesam:artist"] = dbus.MakeVariant([]string{broadcaster})
		meta["xesam:album"] = dbus.MakeVariant(s.Dial)
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(meta))
}

// -----------------------------------------------------------------------------
// Bubble Tea TUI Model
// -----------------------------------------------------------------------------

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampOffset(cursor, offset, count, visibleHeight int) int {
	if visibleHeight <= 0 || count == 0 {
		return 0
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+visibleHeight {
		offset = cursor - visibleHeight + 1
	}
	if offset > count-visibleHeight {
		offset = count - visibleHeight
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

type Model struct {
	width      int
	height     int
	activeTab  int // 0 = Stations, 1 = Favorites
	cursor     int
	favCursor  int
	offset     int
	favOffset  int
	favorites  map[string]bool
	audio      *AudioPlayer
	mpris      *MPRISService
	playingIdx int // index in allStations, or -1 if stopped
	isPlaying  bool
	statusMsg  string
	bars       []int
}

func (m Model) getListHeight() int {
	h := m.height - 16
	if h < 3 {
		return 3
	}
	return h
}

func (m *Model) clampOffsets() {
	listH := m.getListHeight()
	m.offset = clampOffset(m.cursor, m.offset, len(allStations), listH)
	favs := m.getFavoritesList()
	m.favOffset = clampOffset(m.favCursor, m.favOffset, len(favs), listH)
}

func initialModel() Model {
	favs := loadFavorites()
	return Model{
		activeTab:  0,
		cursor:     0,
		favCursor:  0,
		offset:     0,
		favOffset:  0,
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
		m.clampOffsets()

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
			m.clampOffsets()

		case "2":
			m.activeTab = 1
			m.clampOffsets()

		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.clampOffsets()

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
			m.clampOffsets()

		case "down", "j":
			if m.activeTab == 0 {
				if m.cursor < len(allStations)-1 {
					m.cursor++
				}
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 && m.favCursor < len(favs)-1 {
					m.favCursor++
				}
			}
			m.clampOffsets()

		case "pgup":
			listH := m.getListHeight()
			if m.activeTab == 0 {
				m.cursor = maxInt(0, m.cursor-listH)
			} else {
				m.favCursor = maxInt(0, m.favCursor-listH)
			}
			m.clampOffsets()

		case "pgdown":
			listH := m.getListHeight()
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.cursor+listH)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favCursor+listH)
				}
			}
			m.clampOffsets()

		case "home", "g":
			if m.activeTab == 0 {
				m.cursor = 0
			} else {
				m.favCursor = 0
			}
			m.clampOffsets()

		case "end", "G":
			if m.activeTab == 0 {
				m.cursor = maxInt(0, len(allStations)-1)
			} else {
				favs := m.getFavoritesList()
				m.favCursor = maxInt(0, len(favs)-1)
			}
			m.clampOffsets()

		case "f":
			target := m.getCurrentStation()
			if target != nil {
				if m.favorites[target.ID] {
					delete(m.favorites, target.ID)
					m.statusMsg = fmt.Sprintf("Removed %s from favorites", target.NameEn)
					favs := m.getFavoritesList()
					if m.favCursor >= len(favs) && m.favCursor > 0 {
						m.favCursor = len(favs) - 1
					}
				} else {
					m.favorites[target.ID] = true
					m.statusMsg = fmt.Sprintf("Added %s to favorites", target.NameEn)
				}
				saveFavorites(m.favorites)
				m.clampOffsets()
			}

		case " ":
			m.togglePlay()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) getCurrentStation() *Station {
	if m.activeTab == 0 {
		if m.cursor >= 0 && m.cursor < len(allStations) {
			return &allStations[m.cursor]
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
	for _, st := range allStations {
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
	for idx, s := range allStations {
		if s.ID == target.ID {
			targetIdx = idx
			break
		}
	}

	if m.isPlaying && m.playingIdx == targetIdx {
		m.audio.Stop()
		m.isPlaying = false
		m.statusMsg = fmt.Sprintf("Stopped: %s", target.NameEn)
		if m.mpris != nil {
			m.mpris.Update("Stopped", nil)
		}
		return
	}

	err := m.audio.Play(target.StreamURL)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
		m.isPlaying = false
		return
	}

	m.isPlaying = true
	m.playingIdx = targetIdx
	m.statusMsg = fmt.Sprintf("Playing live: [%s] %s", target.Region, target.NameEn)
	if m.mpris != nil {
		m.mpris.Update("Playing", target)
	}
}

func (m *Model) selectNextStation() {
	if len(allStations) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx + 1) % len(allStations)
	st := &allStations[m.playingIdx]
	_ = m.audio.Play(st.StreamURL)
	m.isPlaying = true
	if m.mpris != nil {
		m.mpris.Update("Playing", st)
	}
}

func (m *Model) selectPrevStation() {
	if len(allStations) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx - 1 + len(allStations)) % len(allStations)
	st := &allStations[m.playingIdx]
	_ = m.audio.Play(st.StreamURL)
	m.isPlaying = true
	if m.mpris != nil {
		m.mpris.Update("Playing", st)
	}
}

// -----------------------------------------------------------------------------
// UI Styling & Rendering
// -----------------------------------------------------------------------------

var (
	colorPink    = lipgloss.Color("#F38BA8")
	colorMauve   = lipgloss.Color("#CBA6F7")
	colorGreen   = lipgloss.Color("#A6E3A1")
	colorYellow  = lipgloss.Color("#F9E2AF")
	colorCyan    = lipgloss.Color("#89DCEB")
	colorSubtext = lipgloss.Color("#A6ADC8")
	colorBase    = lipgloss.Color("#1E1E2E")
	colorSurface = lipgloss.Color("#313244")

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

func padWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func fitWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w > width {
		return truncateWidth(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	contentWidth := maxInt(40, m.width-4)
	listHeight := m.getListHeight()

	// 1. Header
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("📻 INTERNET RADIO (HK • TW • JP • SG)"),
		lipgloss.NewStyle().Foreground(colorSubtext).Render(" • MPRIS2 Enabled"),
	)

	// 2. Tabs
	favsList := m.getFavoritesList()
	favCount := len(favsList)

	var tab1, tab2 string
	if m.activeTab == 0 {
		tab1 = activeTabStyle.Render(fmt.Sprintf("1: All Stations (%d/%d) [1]", m.cursor+1, len(allStations)))
		tab2 = inactiveTabStyle.Render(fmt.Sprintf("2: Favorites (%d) [2]", favCount))
	} else {
		currentFavPos := 0
		if favCount > 0 {
			currentFavPos = m.favCursor + 1
		}
		tab1 = inactiveTabStyle.Render(fmt.Sprintf("1: All Stations (%d) [1]", len(allStations)))
		tab2 = activeTabStyle.Render(fmt.Sprintf("2: Favorites (%d/%d) [2]", currentFavPos, favCount))
	}

	var stationsToRender []Station
	activeCursor := m.cursor
	currentOffset := m.offset

	if m.activeTab == 0 {
		stationsToRender = allStations
		currentOffset = clampOffset(m.cursor, m.offset, len(allStations), listHeight)
	} else {
		stationsToRender = favsList
		activeCursor = m.favCursor
		currentOffset = clampOffset(m.favCursor, m.favOffset, favCount, listHeight)
	}

	totalItems := len(stationsToRender)
	scrollInfo := ""
	if totalItems > listHeight {
		endIdx := minInt(totalItems, currentOffset+listHeight)
		scrollInfo = lipgloss.NewStyle().Foreground(colorSubtext).Render(
			fmt.Sprintf("  [Showing %d-%d of %d]", currentOffset+1, endIdx, totalItems),
		)
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Center, tab1, " ", tab2, scrollInfo)

	// 3. Station List (Strictly budgeted to listHeight lines for full-screen view)
	var listLines []string
	showScrollbar := totalItems > listHeight

	// Scrollbar thumb metrics
	thumbHeight := 1
	thumbStart := 0
	if showScrollbar {
		thumbHeight = maxInt(1, (listHeight*listHeight)/totalItems)
		maxOffset := totalItems - listHeight
		if maxOffset > 0 {
			thumbStart = (currentOffset * (listHeight - thumbHeight)) / maxOffset
		}
	}
	thumbEnd := thumbStart + thumbHeight

	if totalItems == 0 {
		msg1 := lipgloss.NewStyle().Foreground(colorSubtext).Render("  No favorite stations added yet.")
		msg2 := lipgloss.NewStyle().Foreground(colorSubtext).Render("  Press 'f' on any station in Tab 1 to add.")
		listLines = append(listLines, msg1, msg2)
		for len(listLines) < listHeight {
			listLines = append(listLines, "")
		}
	} else {
		for row := 0; row < listHeight; row++ {
			itemIdx := currentOffset + row
			if itemIdx < totalItems {
				st := stationsToRender[itemIdx]
				isSelected := (itemIdx == activeCursor)
				isThisPlaying := (m.isPlaying && m.playingIdx >= 0 && allStations[m.playingIdx].ID == st.ID)

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

				// Region tag
				var regionTag string
				switch st.Region {
				case "TW":
					regionTag = lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("[TW] ")
				case "JP":
					regionTag = lipgloss.NewStyle().Foreground(colorPink).Bold(true).Render("[JP] ")
				case "SG":
					regionTag = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("[SG] ")
				default:
					regionTag = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("[HK] ")
				}

				// Exact original column widths preserving alignment
				idCol := padWidth(st.ID, 6)
				zhCol := padWidth(st.NameZh, 22)
				enCol := padWidth(st.NameEn, 28)
				dialCol := lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("(%s)", st.Dial))

				var textStyle lipgloss.Style
				if isSelected {
					textStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
				} else {
					textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
				}

				rowText := textStyle.Render(fmt.Sprintf("%s %s %s", idCol, zhCol, enCol))
				line := fmt.Sprintf("%s%s%s %s%s %s", cursorMarker, favStar, playBadge, regionTag, rowText, dialCol)

				if showScrollbar {
					if row >= thumbStart && row < thumbEnd {
						line += " " + lipgloss.NewStyle().Foreground(colorMauve).Render("█")
					} else {
						line += " " + lipgloss.NewStyle().Foreground(colorSurface).Render("│")
					}
				}
				listLines = append(listLines, line)
			} else {
				blankLine := ""
				if showScrollbar {
					padSpaces := strings.Repeat(" ", maxInt(0, contentWidth-2))
					blankLine = padSpaces + " " + lipgloss.NewStyle().Foreground(colorSurface).Render("│")
				}
				listLines = append(listLines, blankLine)
			}
		}
	}
	listBlock := strings.Join(listLines, "\n")

	// 4. Equalizer / Now Playing Panel (Fixed 3 content lines, 5 total with borders)
	var eqBar strings.Builder
	eqChars := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇"}
	for _, val := range m.bars {
		eqBar.WriteString(eqChars[val])
	}

	cardInnerWidth := maxInt(20, contentWidth-4)
	lblTitle := lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render(fitWidth("▶ Title:", 12))
	lblDesc := lipgloss.NewStyle().Foreground(colorSubtext).Render(fitWidth("  Describe:", 12))
	lblNotice := lipgloss.NewStyle().Foreground(colorYellow).Render(fitWidth("  Notice:", 12))

	var cardLine1, cardLine2, cardLine3 string
	if m.isPlaying && m.playingIdx >= 0 {
		curr := allStations[m.playingIdx]
		eqStyled := lipgloss.NewStyle().Foreground(colorGreen).Render(eqBar.String())

		rawTitle := fmt.Sprintf("[%s] %s  %s  (%s)", curr.Region, curr.NameZh, curr.NameEn, curr.Dial)
		cardLine1 = lblTitle + lipgloss.NewStyle().Bold(true).Render(fitWidth(rawTitle, maxInt(10, cardInnerWidth-12)))

		descAvail := maxInt(10, cardInnerWidth-12-lipgloss.Width(eqStyled)-2)
		cardLine2 = lblDesc + fmt.Sprintf("%s  %s", eqStyled, fitWidth(curr.Desc, descAvail))
	} else {
		cardLine1 = lblTitle + lipgloss.NewStyle().Foreground(colorSubtext).Render("Stopped")
		cardLine2 = lblDesc + lipgloss.NewStyle().Foreground(colorSubtext).Render("Select a station and press [Space] to play")
	}

	if m.statusMsg != "" {
		cardLine3 = lblNotice + lipgloss.NewStyle().Foreground(colorYellow).Render(fitWidth(m.statusMsg, maxInt(10, cardInnerWidth-12)))
	} else {
		cardLine3 = ""
	}

	nowPlayingInfo := fmt.Sprintf("%s\n%s\n%s", cardLine1, cardLine2, cardLine3)
	playerCard := nowPlayingBox.Width(cardInnerWidth).Height(3).Render(nowPlayingInfo)

	// 5. Help Footer
	footer := helpStyle.Render(
		"[Space] Play/Stop • [f] Fav • [1/2] Tab • [↑/↓, j/k] Navigate • [PgUp/PgDn] Page • [q] Quit",
	)

	// Assemble layout to fill the exact full-screen window
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tabsRow,
		"",
		listBlock,
		"",
		playerCard,
		"",
		footer,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

// -----------------------------------------------------------------------------
// Favorites Persistence
// -----------------------------------------------------------------------------

func getFavFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "iradio")
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

	mpris := startMPRIS(p)
	m.mpris = mpris

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running player: %v\n", err)
		os.Exit(1)
	}

	m.audio.Stop()
}
