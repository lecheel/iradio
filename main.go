package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	ID        string `json:"id"`
	Region    string `json:"region"`
	NameZh    string `json:"name_zh"`
	NameEn    string `json:"name_en"`
	Dial      string `json:"dial"`
	Desc      string `json:"desc"`
	StreamURL string `json:"stream_url"`
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

	// --- Taiwan (Hit FM, News98, UFO, ICRT, Bravo, Classical) ---
	{
		ID:        "HITFM",
		Region:    "TW",
		NameZh:    "Hit FM 聯播網",
		NameEn:    "Hit FM 107.7",
		Dial:      "FM 107.7 MHz (Taipei)",
		Desc:      "Taiwan's #1 Hit Music Station (Taipei/Hitoradio)",
		StreamURL: "https://www.hitoradio.com/newweb/hichannel.php?channelID=1&action=getLIVEURL",
	},
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

	// --- Malaysia (988, Ai FM, BFM 89.9, Suria FM) ---
	{
		ID:        "M988",
		Region:    "MY",
		NameZh:    "988電台",
		NameEn:    "988 FM",
		Dial:      "FM 98.8 MHz (Kuala Lumpur)",
		Desc:      "Malaysia Top Chinese Hit Music & News Network",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/988_FMAAC.aac",
	},
	{
		ID:        "AIFM",
		Region:    "MY",
		NameZh:    "爱FM (Ai FM)",
		NameEn:    "Ai FM Malaysia",
		Dial:      "FM 89.3 / 106.7 MHz",
		Desc:      "RTM National Chinese Radio Station (Mandarin)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/AI_FMAAC.aac",
	},
	{
		ID:        "BFM899",
		Region:    "MY",
		NameZh:    "BFM 商業電台",
		NameEn:    "BFM 89.9",
		Dial:      "FM 89.9 MHz (Kuala Lumpur)",
		Desc:      "The Business Station, Current Affairs & Finance (English)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/BFM.mp3",
	},
	{
		ID:        "SURIA",
		Region:    "MY",
		NameZh:    "Suria FM 陽光電台",
		NameEn:    "Suria FM 105.3",
		Dial:      "FM 105.3 MHz (Klang Valley)",
		Desc:      "Segar & Terkini, Malay Hits & Entertainment (Malay)",
		StreamURL: "https://playerservices.streamtheworld.com/api/livestream-redirect/SURIA_FMAAC.aac",
	},
}

// -----------------------------------------------------------------------------
// Audio Stream Backend
// -----------------------------------------------------------------------------

type AudioPlayer struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func resolveStreamURL(u string) string {
	if strings.Contains(u, "hitoradio.com") && strings.Contains(u, "getLIVEURL") {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.Header.Set("Referer", "https://www.hitoradio.com/")
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				target := strings.TrimSpace(string(b))
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
					return target
				}
			}
		}

		postReq, err := http.NewRequest("POST", "https://www.hitoradio.com/mobile/hichannel.php", strings.NewReader("channelID=1&action=getLIVEURL"))
		if err == nil {
			postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			postReq.Header.Set("User-Agent", "Mozilla/5.0")
			postReq.Header.Set("Referer", "https://www.hitoradio.com/")
			resp, err := client.Do(postReq)
			if err == nil {
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				target := strings.TrimSpace(string(b))
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
					return target
				}
			}
		}
	}
	return u
}

func (p *AudioPlayer) Play(url string) error {
	p.Stop()
	url = resolveStreamURL(url)

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
		} else if s.Region == "MY" {
			broadcaster = s.NameZh + " (Malaysia)"
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

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

type tabSwitchTimeoutMsg struct {
	id  int
	tab int
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
	width        int
	height       int
	activeTab    int // 0 = Stations, 1 = Favorites
	cursor       int
	favCursor    int
	offset       int
	favOffset    int
	favorites    map[string]bool
	audio        *AudioPlayer
	mpris        *MPRISService
	playingIdx   int // index in allStations, or -1 if stopped
	isPlaying    bool
	statusMsg    string
	bars         []int
	countBuffer  string
	pendingTabID int
}

func (m *Model) getAndResetCount() int {
	if m.countBuffer == "" {
		return 1
	}
	c, err := strconv.Atoi(m.countBuffer)
	m.countBuffer = ""
	if err != nil || c < 1 {
		return 1
	}
	return c
}

func (m *Model) jumpNextCountry() {
	stations := allStations
	if m.activeTab == 1 {
		stations = m.getFavoritesList()
	}
	if len(stations) == 0 {
		return
	}

	curIdx := m.cursor
	if m.activeTab == 1 {
		curIdx = m.favCursor
	}
	if curIdx < 0 || curIdx >= len(stations) {
		curIdx = 0
	}

	currentRegion := stations[curIdx].Region
	targetIdx := -1

	// Look forward for the first station of the next region
	for i := curIdx + 1; i < len(stations); i++ {
		if stations[i].Region != currentRegion {
			targetIdx = i
			break
		}
	}

	// Wrap around from the start if needed
	if targetIdx == -1 {
		for i := 0; i < curIdx; i++ {
			if stations[i].Region != currentRegion {
				targetIdx = i
				break
			}
		}
	}

	if targetIdx != -1 {
		if m.activeTab == 0 {
			m.cursor = targetIdx
		} else {
			m.favCursor = targetIdx
		}
		m.clampOffsets()
	}
}

func (m *Model) jumpPrevCountry() {
	stations := allStations
	if m.activeTab == 1 {
		stations = m.getFavoritesList()
	}
	if len(stations) == 0 {
		return
	}

	curIdx := m.cursor
	if m.activeTab == 1 {
		curIdx = m.favCursor
	}
	if curIdx < 0 || curIdx >= len(stations) {
		curIdx = 0
	}

	currentRegion := stations[curIdx].Region

	// 1. Walk backward to find any item belonging to a previous region
	prevRegionIdx := -1
	for i := curIdx - 1; i >= 0; i-- {
		if stations[i].Region != currentRegion {
			prevRegionIdx = i
			break
		}
	}
	if prevRegionIdx == -1 {
		// Wrap around from the end
		for i := len(stations) - 1; i > curIdx; i-- {
			if stations[i].Region != currentRegion {
				prevRegionIdx = i
				break
			}
		}
	}

	if prevRegionIdx == -1 {
		return
	}

	// 2. Find the very first station of that target region
	targetRegion := stations[prevRegionIdx].Region
	targetIdx := prevRegionIdx
	for i := prevRegionIdx; i >= 0; i-- {
		if stations[i].Region == targetRegion {
			targetIdx = i
		} else {
			break
		}
	}

	if m.activeTab == 0 {
		m.cursor = targetIdx
	} else {
		m.favCursor = targetIdx
	}
	m.clampOffsets()
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
	customCount := initCustomStations()
	status := ""
	if customCount > 0 {
		status = fmt.Sprintf("Loaded %d custom station(s) from config", customCount)
	}

	return Model{
		activeTab:    0,
		cursor:       0,
		favCursor:    0,
		offset:       0,
		favOffset:    0,
		favorites:    favs,
		audio:        &AudioPlayer{},
		playingIdx:   -1,
		isPlaying:    false,
		statusMsg:    status,
		bars:         make([]int, 14),
		countBuffer:  "",
		pendingTabID: 0,
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

	case tabSwitchTimeoutMsg:
		if msg.id == m.pendingTabID && (m.countBuffer == "1" || m.countBuffer == "2") {
			m.activeTab = msg.tab
			m.countBuffer = ""
			m.clampOffsets()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.audio.Stop()
			if m.mpris != nil {
				m.mpris.Update("Stopped", nil)
			}
			return m, tea.Quit

		case "esc":
			m.pendingTabID++
			m.countBuffer = ""

		case "1", "2":
			if m.countBuffer == "" {
				m.countBuffer = msg.String()
				m.pendingTabID++
				id := m.pendingTabID
				tabIdx := 0
				if msg.String() == "2" {
					tabIdx = 1
				}
				cmds = append(cmds, tea.Tick(260*time.Millisecond, func(t time.Time) tea.Msg {
					return tabSwitchTimeoutMsg{id: id, tab: tabIdx}
				}))
			} else {
				m.countBuffer += msg.String()
			}

		case "3", "4", "5", "6", "7", "8", "9":
			m.pendingTabID++
			m.countBuffer += msg.String()

		case "0":
			if m.countBuffer != "" {
				m.countBuffer += "0"
			}

		case "tab":
			m.pendingTabID++
			m.countBuffer = ""
			m.activeTab = (m.activeTab + 1) % 2
			m.clampOffsets()

		case "up", "k":
			m.pendingTabID++
			count := m.getAndResetCount()
			if m.activeTab == 0 {
				m.cursor = maxInt(0, m.cursor-count)
			} else {
				m.favCursor = maxInt(0, m.favCursor-count)
			}
			m.clampOffsets()

		case "down", "j":
			m.pendingTabID++
			count := m.getAndResetCount()
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.cursor+count)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favCursor+count)
				}
			}
			m.clampOffsets()

		case "J":
			m.pendingTabID++
			count := m.getAndResetCount()
			for i := 0; i < count; i++ {
				m.jumpNextCountry()
			}

		case "K":
			m.pendingTabID++
			count := m.getAndResetCount()
			for i := 0; i < count; i++ {
				m.jumpPrevCountry()
			}

		case "H":
			m.pendingTabID++
			m.countBuffer = ""
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.offset)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favOffset)
				}
			}
			m.clampOffsets()

		case "M":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.offset+listH/2)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favOffset+listH/2)
				}
			}
			m.clampOffsets()

		case "L":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.offset+listH-1)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favOffset+listH-1)
				}
			}
			m.clampOffsets()

		case "ctrl+d":
			m.pendingTabID++
			m.countBuffer = ""
			half := maxInt(1, m.getListHeight()/2)
			if m.activeTab == 0 {
				m.cursor = minInt(len(allStations)-1, m.cursor+half)
			} else {
				favs := m.getFavoritesList()
				if len(favs) > 0 {
					m.favCursor = minInt(len(favs)-1, m.favCursor+half)
				}
			}
			m.clampOffsets()

		case "ctrl+u":
			m.pendingTabID++
			m.countBuffer = ""
			half := maxInt(1, m.getListHeight()/2)
			if m.activeTab == 0 {
				m.cursor = maxInt(0, m.cursor-half)
			} else {
				m.favCursor = maxInt(0, m.favCursor-half)
			}
			m.clampOffsets()

		case "pgup":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			if m.activeTab == 0 {
				m.cursor = maxInt(0, m.cursor-listH)
			} else {
				m.favCursor = maxInt(0, m.favCursor-listH)
			}
			m.clampOffsets()

		case "pgdown":
			m.pendingTabID++
			m.countBuffer = ""
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
			m.pendingTabID++
			if m.countBuffer != "" {
				targetLine := m.getAndResetCount() - 1
				if m.activeTab == 0 {
					m.cursor = minInt(len(allStations)-1, maxInt(0, targetLine))
				} else {
					favs := m.getFavoritesList()
					if len(favs) > 0 {
						m.favCursor = minInt(len(favs)-1, maxInt(0, targetLine))
					}
				}
			} else {
				if m.activeTab == 0 {
					m.cursor = 0
				} else {
					m.favCursor = 0
				}
			}
			m.clampOffsets()

		case "end", "G":
			m.pendingTabID++
			if m.countBuffer != "" {
				targetLine := m.getAndResetCount() - 1
				if m.activeTab == 0 {
					m.cursor = minInt(len(allStations)-1, maxInt(0, targetLine))
				} else {
					favs := m.getFavoritesList()
					if len(favs) > 0 {
						m.favCursor = minInt(len(favs)-1, maxInt(0, targetLine))
					}
				}
			} else {
				if m.activeTab == 0 {
					m.cursor = maxInt(0, len(allStations)-1)
				} else {
					favs := m.getFavoritesList()
					m.favCursor = maxInt(0, len(favs)-1)
				}
			}
			m.clampOffsets()

		case "f":
			m.pendingTabID++
			m.countBuffer = ""
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

		case " ", "enter":
			m.pendingTabID++
			m.countBuffer = ""
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
	colorPeach   = lipgloss.Color("#FAB387")
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
		titleStyle.Render("📻 INTERNET RADIO (HK • TW • JP • SG • MY)"),
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

				relDist := absInt(itemIdx - activeCursor)
				var relMarker string
				if isSelected {
					relMarker = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render(fmt.Sprintf("❯%2d ", relDist))
				} else {
					relMarker = lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf(" %2d ", relDist))
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
				case "MY":
					regionTag = lipgloss.NewStyle().Foreground(colorPeach).Bold(true).Render("[MY] ")
				case "HK":
					regionTag = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("[HK] ")
				default:
					r := st.Region
					if r == "" {
						r = "USER"
					}
					if len(r) > 4 {
						r = r[:4]
					}
					regionTag = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(fmt.Sprintf("[%s] ", padWidth(r, 2)))
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
				line := fmt.Sprintf("%s%s%s %s%s %s", relMarker, favStar, playBadge, regionTag, rowText, dialCol)
				listLines = append(listLines, line)
			} else {
				listLines = append(listLines, "")
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
		cardLine2 = lblDesc + lipgloss.NewStyle().Foreground(colorSubtext).Render("Select a station and press [Enter/Space] to play")
	}

	if m.statusMsg != "" {
		cardLine3 = lblNotice + lipgloss.NewStyle().Foreground(colorYellow).Render(fitWidth(m.statusMsg, maxInt(10, cardInnerWidth-12)))
	} else {
		cardLine3 = ""
	}

	nowPlayingInfo := fmt.Sprintf("%s\n%s\n%s", cardLine1, cardLine2, cardLine3)
	playerCard := nowPlayingBox.Width(cardInnerWidth).Height(3).Render(nowPlayingInfo)

	// 5. Help Footer
	footerText := "[Enter/Space] Play/Stop • [f] Fav • [j/k] Move • [J/K] Country Jump • [Tab] Tab • [q] Quit"
	var footer string
	if m.countBuffer != "" {
		countBadge := lipgloss.NewStyle().Background(colorMauve).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(fmt.Sprintf(" Count: %s ", m.countBuffer))
		footer = lipgloss.JoinHorizontal(lipgloss.Center, countBadge, "  ", helpStyle.Render(footerText))
	} else {
		footer = helpStyle.Render(footerText)
	}

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
// Config & Stations Persistence
// -----------------------------------------------------------------------------

func ensureConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "iradio")
	_ = os.MkdirAll(dir, 0755)

	exampleFile := filepath.Join(dir, "stations.example.json")
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		exampleContent := `[
  {
    "id": "MYSTATION",
    "region": "TW",
    "name_zh": "自訂電台範例",
    "name_en": "My Custom Station",
    "dial": "Online",
    "desc": "Custom stream description",
    "stream_url": "https://example.com/stream.m3u8"
  }
]
`
		_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)
	}
	return dir
}

func loadCustomStations() []Station {
	dir := ensureConfigDir()
	candidates := []string{
		filepath.Join(dir, "stations.json"),
		filepath.Join(dir, "custom_stations.json"),
	}

	var loaded []Station
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var list []Station
		if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
			loaded = append(loaded, list...)
		}
	}
	return loaded
}

func initCustomStations() int {
	custom := loadCustomStations()
	if len(custom) == 0 {
		return 0
	}

	idxMap := make(map[string]int)
	for i, s := range allStations {
		idxMap[s.ID] = i
	}

	for i, s := range custom {
		if s.StreamURL == "" {
			continue
		}
		if s.ID == "" {
			s.ID = fmt.Sprintf("CUST%d", i+1)
		}
		if s.Region == "" {
			s.Region = "USER"
		}
		if s.NameZh == "" && s.NameEn != "" {
			s.NameZh = s.NameEn
		} else if s.NameEn == "" && s.NameZh != "" {
			s.NameEn = s.NameZh
		}

		if idx, exists := idxMap[s.ID]; exists {
			allStations[idx] = s
		} else {
			allStations = append(allStations, s)
			idxMap[s.ID] = len(allStations) - 1
		}
	}
	return len(custom)
}

func getFavFilePath() string {
	dir := ensureConfigDir()
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
