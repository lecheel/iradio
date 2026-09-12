package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

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
	URL    string
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
func (m *mprisPlayer) OpenUri(uri string) *dbus.Error {
	if m.prog != nil {
		m.prog.Send(MPRISActionMsg{Action: "open", URL: uri})
	}
	return nil
}

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

	reply, err := conn.RequestName("org.mpris.MediaPlayer2.iradio", dbus.NameFlagDoNotQueue)
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

func (m *MPRISService) UpdateMusic(status string, t *MusicTrack) {
	if !m.active || m.properties == nil {
		return
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "PlaybackStatus", dbus.MakeVariant(status))
	meta := map[string]dbus.Variant{}
	if t != nil {
		meta["mpris:trackid"] = dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/track/music"))
		meta["xesam:title"] = dbus.MakeVariant(t.Title)
		meta["xesam:artist"] = dbus.MakeVariant([]string{t.Artist})
		meta["xesam:album"] = dbus.MakeVariant(t.Album)
	}
	m.properties.Set("org.mpris.MediaPlayer2.Player", "Metadata", dbus.MakeVariant(meta))
}

// -----------------------------------------------------------------------------
// Local Music Library (~/Music) & Lyrics
// -----------------------------------------------------------------------------

type LyricLine struct {
	Time time.Duration
	Text string
}

type MusicTrack struct {
	Path     string
	Filename string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
	Lyrics   []LyricLine
}

func parseLRCFile(lrcPath string) []LyricLine {
	b, err := os.ReadFile(lrcPath)
	if err != nil {
		return nil
	}
	var lines []LyricLine
	re := regexp.MustCompile(`\[(\d+):(\d+(?:\.\d+)?)\](.*)`)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			min, _ := strconv.Atoi(m[1])
			sec, _ := strconv.ParseFloat(m[2], 64)
			d := time.Duration(min)*time.Minute + time.Duration(sec*float64(time.Second))
			text := strings.TrimSpace(m[3])
			lines = append(lines, LyricLine{Time: d, Text: text})
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Time < lines[j].Time
	})
	return lines
}

func probeDuration(path string) time.Duration {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err == nil {
		secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
		if err == nil && secs > 0 {
			return time.Duration(secs * float64(time.Second))
		}
	}
	return 0
}

func decodeID3Text(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	enc := data[0]
	raw := data[1:]
	switch enc {
	case 1: // UTF-16 with BOM
		if len(raw) < 2 {
			return ""
		}
		var isBigEndian bool
		if raw[0] == 0xFE && raw[1] == 0xFF {
			isBigEndian = true
			raw = raw[2:]
		} else if raw[0] == 0xFF && raw[1] == 0xFE {
			isBigEndian = false
			raw = raw[2:]
		}
		u16s := make([]uint16, len(raw)/2)
		for i := 0; i < len(u16s); i++ {
			if isBigEndian {
				u16s[i] = binary.BigEndian.Uint16(raw[i*2:])
			} else {
				u16s[i] = binary.LittleEndian.Uint16(raw[i*2:])
			}
		}
		return strings.TrimRight(string(utf16.Decode(u16s)), "\x00")
	default:
		return strings.TrimRight(string(raw), "\x00")
	}
}

func parseID3(path string) (title, artist, album string, dur time.Duration) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	head := make([]byte, 10)
	if _, err := io.ReadFull(f, head); err == nil && string(head[:3]) == "ID3" {
		size := int(head[6])<<21 | int(head[7])<<14 | int(head[8])<<7 | int(head[9])
		tagBuf := make([]byte, size)
		if _, err := io.ReadFull(f, tagBuf); err == nil {
			pos := 0
			for pos+10 <= len(tagBuf) {
				frameID := string(tagBuf[pos : pos+4])
				if tagBuf[pos] == 0 {
					break
				}
				frameSize := int(binary.BigEndian.Uint32(tagBuf[pos+4 : pos+8]))
				if frameSize <= 0 || pos+10+frameSize > len(tagBuf) {
					break
				}
				frameData := tagBuf[pos+10 : pos+10+frameSize]
				pos += 10 + frameSize

				switch frameID {
				case "TIT2":
					title = decodeID3Text(frameData)
				case "TPE1":
					artist = decodeID3Text(frameData)
				case "TALB":
					album = decodeID3Text(frameData)
				case "TLEN":
					txt := decodeID3Text(frameData)
					if ms, err := strconv.Atoi(strings.TrimSpace(txt)); err == nil && ms > 0 {
						dur = time.Duration(ms) * time.Millisecond
					}
				}
			}
		}
	}
	return
}

func loadMusicTracks() []MusicTrack {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	musicDir := filepath.Join(home, "Music")
	_ = os.MkdirAll(musicDir, 0755)

	var list []MusicTrack
	exts := map[string]bool{".mp3": true, ".flac": true, ".m4a": true, ".wav": true, ".ogg": true}

	_ = filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !exts[ext] {
			return nil
		}

		fname := filepath.Base(path)
		title, artist, album, dur := parseID3(path)

		cleanName := strings.TrimSuffix(fname, filepath.Ext(fname))
		if title == "" || artist == "" {
			if strings.Contains(cleanName, "_") {
				parts := strings.Split(cleanName, "_")
				if len(parts) >= 3 {
					if artist == "" {
						artist = parts[1]
					}
					if title == "" {
						title = strings.Join(parts[2:], " ")
					}
					if album == "" {
						album = parts[len(parts)-1]
					}
				} else if len(parts) == 2 {
					if artist == "" {
						artist = parts[0]
					}
					if title == "" {
						title = parts[1]
					}
				}
			} else if strings.Contains(cleanName, "-") {
				parts := strings.Split(cleanName, "-")
				if artist == "" {
					artist = strings.TrimSpace(parts[0])
				}
				if title == "" {
					title = strings.TrimSpace(parts[1])
				}
			}
		}
		if title == "" {
			title = cleanName
		}
		if artist == "" {
			artist = "Unknown Artist"
		}
		if album == "" {
			album = "Local Music"
		}

		lrcPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".lrc"
		var lyrics []LyricLine
		if _, err := os.Stat(lrcPath); err == nil {
			lyrics = parseLRCFile(lrcPath)
		}

		list = append(list, MusicTrack{
			Path:     path,
			Filename: fname,
			Title:    title,
			Artist:   artist,
			Album:    album,
			Duration: dur,
			Lyrics:   lyrics,
		})
		return nil
	})

	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Filename) < strings.ToLower(list[j].Filename)
	})

	return list
}

func getSystemVolume() string {
	if out, err := exec.Command("amixer", "sget", "Master").Output(); err == nil {
		s := string(out)
		if idx := strings.Index(s, "["); idx != -1 {
			if end := strings.Index(s[idx:], "%]"); end != -1 {
				return s[idx+1 : idx+end+1]
			}
		}
	}
	if out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			if v, err := strconv.ParseFloat(fields[1], 64); err == nil {
				return fmt.Sprintf("%d%%", int(v*100))
			}
		}
	}
	return "63%"
}

func getMusicFavFilePath() string {
	dir := ensureConfigDir()
	return filepath.Join(dir, "music_favorites.json")
}

func loadMusicFavorites() map[string]bool {
	favs := make(map[string]bool)
	data, err := os.ReadFile(getMusicFavFilePath())
	if err == nil {
		_ = json.Unmarshal(data, &favs)
	}
	return favs
}

func saveMusicFavorites(favs map[string]bool) {
	data, err := json.MarshalIndent(favs, "", "  ")
	if err == nil {
		_ = os.WriteFile(getMusicFavFilePath(), data, 0644)
	}
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
	activeTab    int // 0 = Stations, 1 = Favorites, 2 = Music Library
	cursor       int
	favCursor    int
	hiddenCursor int
	musicCursor  int
	offset       int
	favOffset    int
	hiddenOffset int
	musicOffset  int
	favorites    map[string]bool
	hidden       map[string]bool
	musicFavs    map[string]bool
	showHidden   bool
	audio        *AudioPlayer
	mpris        *MPRISService
	playingIdx   int // index in allStations, or -1 if stopped
	isPlaying    bool
	isMusic      bool
	musicTracks  []MusicTrack
	musicPlaying int
	trackStart   time.Time
	trackElapsed time.Duration
	statusMsg    string
	bars         []int
	countBuffer  string
	pendingTabID int
	showHelp     bool
	volumeStr    string
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

func (m Model) getVisibleAllStations() []Station {
	var list []Station
	for _, st := range allStations {
		if !m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m Model) getHiddenList() []Station {
	var list []Station
	for _, st := range allStations {
		if m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m Model) getActiveStations() []Station {
	if m.showHidden {
		return m.getHiddenList()
	}
	if m.activeTab == 0 {
		return m.getVisibleAllStations()
	}
	return m.getFavoritesList()
}

func (m *Model) getActiveCursor() int {
	if m.showHidden {
		return m.hiddenCursor
	}
	if m.activeTab == 0 {
		return m.cursor
	}
	return m.favCursor
}

func (m *Model) setActiveCursor(c int) {
	if m.showHidden {
		m.hiddenCursor = c
	} else if m.activeTab == 0 {
		m.cursor = c
	} else {
		m.favCursor = c
	}
}

func (m *Model) getActiveOffset() int {
	if m.showHidden {
		return m.hiddenOffset
	}
	if m.activeTab == 0 {
		return m.offset
	}
	return m.favOffset
}

func (m *Model) jumpNextCountry() {
	stations := m.getActiveStations()
	if len(stations) == 0 {
		return
	}

	curIdx := m.getActiveCursor()
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
		m.setActiveCursor(targetIdx)
		m.clampOffsets()
	}
}

func (m *Model) jumpPrevCountry() {
	stations := m.getActiveStations()
	if len(stations) == 0 {
		return
	}

	curIdx := m.getActiveCursor()
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

	if targetIdx != -1 {
		m.setActiveCursor(targetIdx)
		m.clampOffsets()
	}
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
	allVisible := m.getVisibleAllStations()
	if len(allVisible) > 0 && m.cursor >= len(allVisible) {
		m.cursor = len(allVisible) - 1
	}
	m.offset = clampOffset(m.cursor, m.offset, len(allVisible), listH)

	favs := m.getFavoritesList()
	if len(favs) > 0 && m.favCursor >= len(favs) {
		m.favCursor = len(favs) - 1
	}
	m.favOffset = clampOffset(m.favCursor, m.favOffset, len(favs), listH)

	hiddenList := m.getHiddenList()
	if len(hiddenList) > 0 && m.hiddenCursor >= len(hiddenList) {
		m.hiddenCursor = len(hiddenList) - 1
	}
	m.hiddenOffset = clampOffset(m.hiddenCursor, m.hiddenOffset, len(hiddenList), listH)

	if len(m.musicTracks) > 0 && m.musicCursor >= len(m.musicTracks) {
		m.musicCursor = len(m.musicTracks) - 1
	}
	m.musicOffset = clampOffset(m.musicCursor, m.musicOffset, len(m.musicTracks), listH)
}

func initialModel() Model {
	favs := loadFavorites()
	hidden := loadHidden()
	customCount := initCustomStations()
	musicList := loadMusicTracks()
	mFavs := loadMusicFavorites()
	status := ""
	if customCount > 0 {
		status = fmt.Sprintf("Loaded %d custom station(s)", customCount)
	} else if len(musicList) > 0 {
		status = fmt.Sprintf("Scanned %d tracks from ~/Music", len(musicList))
	}

	return Model{
		activeTab:    0,
		cursor:       0,
		favCursor:    0,
		hiddenCursor: 0,
		musicCursor:  0,
		offset:       0,
		favOffset:    0,
		hiddenOffset: 0,
		musicOffset:  0,
		favorites:    favs,
		hidden:       hidden,
		musicFavs:    mFavs,
		showHidden:   false,
		audio:        &AudioPlayer{},
		playingIdx:   -1,
		isPlaying:    false,
		isMusic:      false,
		musicTracks:  musicList,
		musicPlaying: -1,
		statusMsg:    status,
		bars:         make([]int, 14),
		countBuffer:  "",
		pendingTabID: 0,
		showHelp:     false,
		volumeStr:    getSystemVolume(),
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
			if m.isMusic && m.musicPlaying >= 0 && m.musicPlaying < len(m.musicTracks) {
				m.trackElapsed = time.Since(m.trackStart)
				curTrack := m.musicTracks[m.musicPlaying]
				if curTrack.Duration > 0 && m.trackElapsed >= curTrack.Duration {
					m.playMusicTrack((m.musicPlaying + 1) % len(m.musicTracks))
				}
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
		case "open":
			if msg.URL != "" {
				for idx, s := range allStations {
					if s.StreamURL == msg.URL {
						m.playingIdx = idx
						err := m.audio.Play(s.StreamURL)
						if err != nil {
							m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
							m.isPlaying = false
						} else {
							m.isPlaying = true
							m.statusMsg = fmt.Sprintf("Playing live: [%s] %s", s.Region, s.NameEn)
							if m.mpris != nil {
								m.mpris.Update("Playing", &allStations[idx])
							}
						}
						break
					}
				}
			}
		}

	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q", "enter", " ":
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.audio.Stop()
			if m.mpris != nil {
				m.mpris.Update("Stopped", nil)
			}
			return m, tea.Quit

		case "?":
			m.pendingTabID++
			m.countBuffer = ""
			m.showHelp = true
			return m, nil

		case "esc":
			m.pendingTabID++
			m.countBuffer = ""
			if m.showHidden {
				m.showHidden = false
				m.statusMsg = ""
				m.clampOffsets()
			}

		case "f1":
			m.pendingTabID++
			m.countBuffer = ""
			m.showHidden = false
			m.activeTab = 0
			m.clampOffsets()

		case "f2":
			m.pendingTabID++
			m.countBuffer = ""
			m.showHidden = false
			m.activeTab = 1
			m.clampOffsets()

		case "f3":
			m.pendingTabID++
			m.countBuffer = ""
			m.showHidden = false
			m.activeTab = 2
			m.clampOffsets()

		case "r":
			if m.activeTab == 2 {
				m.musicTracks = loadMusicTracks()
				m.statusMsg = fmt.Sprintf("Rescanned ~/Music (%d tracks found)", len(m.musicTracks))
				m.clampOffsets()
				return m, nil
			}

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			m.pendingTabID++
			m.countBuffer += msg.String()

		case "0":
			if m.countBuffer != "" {
				m.countBuffer += "0"
			}

		case "tab":
			m.pendingTabID++
			m.countBuffer = ""
			if m.showHidden {
				m.showHidden = false
			} else {
				m.activeTab = (m.activeTab + 1) % 3
			}
			m.clampOffsets()

		case "up", "k":
			m.pendingTabID++
			count := m.getAndResetCount()
			if m.activeTab == 2 && !m.showHidden {
				m.musicCursor = maxInt(0, m.musicCursor-count)
			} else {
				c := m.getActiveCursor()
				m.setActiveCursor(maxInt(0, c-count))
			}
			m.clampOffsets()

		case "down", "j":
			m.pendingTabID++
			count := m.getAndResetCount()
			if m.activeTab == 2 && !m.showHidden {
				total := len(m.musicTracks)
				if total > 0 {
					m.musicCursor = minInt(total-1, m.musicCursor+count)
				}
			} else {
				c := m.getActiveCursor()
				total := len(m.getActiveStations())
				if total > 0 {
					m.setActiveCursor(minInt(total-1, c+count))
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
			m.showHidden = !m.showHidden
			if m.showHidden {
				m.statusMsg = "Viewing hidden stations. Press 'd' to restore, 'H' to exit"
			} else {
				m.statusMsg = ""
			}
			m.clampOffsets()

		case "M":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			off := m.getActiveOffset()
			total := len(m.getActiveStations())
			if total > 0 {
				m.setActiveCursor(minInt(total-1, off+listH/2))
			}
			m.clampOffsets()

		case "L":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			off := m.getActiveOffset()
			total := len(m.getActiveStations())
			if total > 0 {
				m.setActiveCursor(minInt(total-1, off+listH-1))
			}
			m.clampOffsets()

		case "ctrl+d":
			m.pendingTabID++
			m.countBuffer = ""
			half := maxInt(1, m.getListHeight()/2)
			c := m.getActiveCursor()
			total := len(m.getActiveStations())
			if total > 0 {
				m.setActiveCursor(minInt(total-1, c+half))
			}
			m.clampOffsets()

		case "ctrl+u":
			m.pendingTabID++
			m.countBuffer = ""
			half := maxInt(1, m.getListHeight()/2)
			c := m.getActiveCursor()
			m.setActiveCursor(maxInt(0, c-half))
			m.clampOffsets()

		case "pgup":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			c := m.getActiveCursor()
			m.setActiveCursor(maxInt(0, c-listH))
			m.clampOffsets()

		case "pgdown":
			m.pendingTabID++
			m.countBuffer = ""
			listH := m.getListHeight()
			c := m.getActiveCursor()
			total := len(m.getActiveStations())
			if total > 0 {
				m.setActiveCursor(minInt(total-1, c+listH))
			}
			m.clampOffsets()

		case "home", "g":
			m.pendingTabID++
			total := len(m.getActiveStations())
			if m.countBuffer != "" {
				targetLine := m.getAndResetCount() - 1
				if total > 0 {
					m.setActiveCursor(minInt(total-1, maxInt(0, targetLine)))
				}
			} else {
				m.setActiveCursor(0)
			}
			m.clampOffsets()

		case "end", "G":
			m.pendingTabID++
			total := len(m.getActiveStations())
			if m.countBuffer != "" {
				targetLine := m.getAndResetCount() - 1
				if total > 0 {
					m.setActiveCursor(minInt(total-1, maxInt(0, targetLine)))
				}
			} else {
				if total > 0 {
					m.setActiveCursor(total - 1)
				} else {
					m.setActiveCursor(0)
				}
			}
			m.clampOffsets()

		case "d":
			m.pendingTabID++
			m.countBuffer = ""
			target := m.getCurrentStation()
			if target != nil {
				if m.showHidden || m.hidden[target.ID] {
					delete(m.hidden, target.ID)
					saveHidden(m.hidden)
					m.statusMsg = fmt.Sprintf("Restored %s to station list", target.NameEn)
					m.clampOffsets()
				} else {
					m.hidden[target.ID] = true
					saveHidden(m.hidden)
					m.statusMsg = fmt.Sprintf("Hidden station: %s (Press 'H' to view hidden)", target.NameEn)
					if m.isPlaying && m.playingIdx >= 0 && allStations[m.playingIdx].ID == target.ID {
						m.audio.Stop()
						m.isPlaying = false
						m.playingIdx = -1
						if m.mpris != nil {
							m.mpris.Update("Stopped", nil)
						}
					}
					m.clampOffsets()
				}
			}

		case "f":
			m.pendingTabID++
			m.countBuffer = ""
			if m.activeTab == 2 && !m.showHidden {
				if m.musicCursor >= 0 && m.musicCursor < len(m.musicTracks) {
					t := m.musicTracks[m.musicCursor]
					if m.musicFavs[t.Path] {
						delete(m.musicFavs, t.Path)
						m.statusMsg = fmt.Sprintf("Removed from favorites: %s", t.Filename)
					} else {
						m.musicFavs[t.Path] = true
						m.statusMsg = fmt.Sprintf("Added to favorites: %s", t.Filename)
					}
					saveMusicFavorites(m.musicFavs)
				}
			} else {
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
			}

		case "n":
			if m.isMusic || m.activeTab == 2 {
				m.selectNextMusicTrack()
			} else {
				m.selectNextStation()
			}

		case "p":
			if m.isMusic || m.activeTab == 2 {
				m.selectPrevMusicTrack()
			} else {
				m.selectPrevStation()
			}

		case " ", "enter":
			m.pendingTabID++
			m.countBuffer = ""
			if m.activeTab == 2 && !m.showHidden {
				m.togglePlayMusic()
			} else {
				m.togglePlay()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) getCurrentStation() *Station {
	stations := m.getActiveStations()
	c := m.getActiveCursor()
	if c >= 0 && c < len(stations) {
		return &stations[c]
	}
	return nil
}

func (m Model) getFavoritesList() []Station {
	var list []Station
	for _, st := range allStations {
		if m.favorites[st.ID] && !m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m *Model) playMusicTrack(idx int) {
	if idx < 0 || idx >= len(m.musicTracks) {
		return
	}
	t := &m.musicTracks[idx]
	if t.Duration == 0 {
		go func(path string, index int) {
			d := probeDuration(path)
			if d > 0 && index < len(m.musicTracks) {
				m.musicTracks[index].Duration = d
			}
		}(t.Path, idx)
	}

	err := m.audio.Play(t.Path)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
		m.isPlaying = false
		m.isMusic = false
		return
	}

	m.isPlaying = true
	m.isMusic = true
	m.musicPlaying = idx
	m.playingIdx = -1
	m.trackStart = time.Now()
	m.trackElapsed = 0
	m.statusMsg = fmt.Sprintf("Playing track: %s", t.Filename)
	if m.mpris != nil {
		m.mpris.UpdateMusic("Playing", t)
	}
}

func (m *Model) togglePlayMusic() {
	if len(m.musicTracks) == 0 {
		return
	}
	if m.isPlaying && m.isMusic && m.musicPlaying == m.musicCursor {
		m.audio.Stop()
		m.isPlaying = false
		m.isMusic = false
		m.statusMsg = fmt.Sprintf("Stopped: %s", m.musicTracks[m.musicCursor].Filename)
		if m.mpris != nil {
			m.mpris.UpdateMusic("Stopped", nil)
		}
		return
	}
	m.playMusicTrack(m.musicCursor)
}

func (m *Model) selectNextMusicTrack() {
	if len(m.musicTracks) == 0 {
		return
	}
	next := 0
	if m.musicPlaying >= 0 {
		next = (m.musicPlaying + 1) % len(m.musicTracks)
	}
	m.musicCursor = next
	m.playMusicTrack(next)
}

func (m *Model) selectPrevMusicTrack() {
	if len(m.musicTracks) == 0 {
		return
	}
	prev := 0
	if m.musicPlaying >= 0 {
		prev = (m.musicPlaying - 1 + len(m.musicTracks)) % len(m.musicTracks)
	}
	m.musicCursor = prev
	m.playMusicTrack(prev)
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

	if m.isPlaying && !m.isMusic && m.playingIdx == targetIdx {
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
	m.isMusic = false
	m.musicPlaying = -1
	m.playingIdx = targetIdx
	m.statusMsg = fmt.Sprintf("Playing live: [%s] %s", target.Region, target.NameEn)
	if m.mpris != nil {
		m.mpris.Update("Playing", target)
	}
}

func (m *Model) selectNextStation() {
	stations := m.getActiveStations()
	if len(stations) == 0 {
		return
	}
	curr := -1
	if m.playingIdx >= 0 && m.playingIdx < len(allStations) {
		for i, st := range stations {
			if st.ID == allStations[m.playingIdx].ID {
				curr = i
				break
			}
		}
	}
	nextIdx := (curr + 1) % len(stations)
	st := &stations[nextIdx]
	for idx, s := range allStations {
		if s.ID == st.ID {
			m.playingIdx = idx
			break
		}
	}
	_ = m.audio.Play(st.StreamURL)
	m.isPlaying = true
	if m.mpris != nil {
		m.mpris.Update("Playing", st)
	}
}

func (m *Model) selectPrevStation() {
	stations := m.getActiveStations()
	if len(stations) == 0 {
		return
	}
	curr := -1
	if m.playingIdx >= 0 && m.playingIdx < len(allStations) {
		for i, st := range stations {
			if st.ID == allStations[m.playingIdx].ID {
				curr = i
				break
			}
		}
	}
	prevIdx := (curr - 1 + len(stations)) % len(stations)
	st := &stations[prevIdx]
	for idx, s := range allStations {
		if s.ID == st.ID {
			m.playingIdx = idx
			break
		}
	}
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

func overlayLine(bgLine, fgLine string, x int) string {
	fgWidth := lipgloss.Width(fgLine)
	targetRightCol := x + fgWidth

	var left strings.Builder
	var right strings.Builder

	curCol := 0
	inEsc := false
	var escSeq strings.Builder
	var lastActiveStyles strings.Builder

	runes := []rune(bgLine)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == '\x1b' {
			inEsc = true
			escSeq.Reset()
			escSeq.WriteRune(r)
			continue
		}

		if inEsc {
			escSeq.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEsc = false
				seq := escSeq.String()
				if curCol <= x {
					left.WriteString(seq)
				}
				if strings.HasSuffix(seq, "m") {
					if seq == "\x1b[0m" || seq == "\x1b[m" {
						lastActiveStyles.Reset()
					} else {
						lastActiveStyles.WriteString(seq)
					}
				}
				if curCol >= targetRightCol {
					right.WriteString(seq)
				}
			}
			continue
		}

		w := lipgloss.Width(string(r))

		if curCol+w <= x {
			left.WriteRune(r)
		} else if curCol < x {
			left.WriteString(strings.Repeat(" ", x-curCol))
		}

		if curCol >= targetRightCol {
			right.WriteRune(r)
		}

		curCol += w
	}

	leftWidth := lipgloss.Width(left.String())
	if leftWidth < x {
		left.WriteString(strings.Repeat(" ", x-leftWidth))
	}

	var sb strings.Builder
	sb.WriteString(left.String())
	sb.WriteString("\x1b[0m")
	sb.WriteString(fgLine)
	sb.WriteString("\x1b[0m")
	if right.Len() > 0 {
		sb.WriteString(lastActiveStyles.String())
		sb.WriteString(right.String())
		sb.WriteString("\x1b[0m")
	}

	return sb.String()
}

func overlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fLine := range fgLines {
		targetY := y + i
		if targetY < 0 || targetY >= len(bgLines) {
			continue
		}
		bgLines[targetY] = overlayLine(bgLines[targetY], fLine, x)
	}

	return strings.Join(bgLines, "\n")
}

func (m Model) renderHelpBox() string {
	boxWidth := minInt(68, maxInt(40, m.width-6))

	bgStyle := lipgloss.NewStyle().Background(colorSurface)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorMauve).
		Padding(0, 1).
		Render("📖 KEYBOARD SHORTCUTS & HELP")

	sectionStyle := bgStyle.Bold(true).Foreground(colorYellow)
	keyStyle := bgStyle.Bold(true).Foreground(colorCyan)
	descStyle := bgStyle.Foreground(lipgloss.Color("#CDD6F4"))
	dimStyle := bgStyle.Foreground(colorSubtext)

	row := func(key, desc string) string {
		k := fitWidth(key, 18)
		return bgStyle.Render("  ") + keyStyle.Render(k) + bgStyle.Render(" ") + descStyle.Render(desc)
	}

	lines := []string{
		title,
		"",
		sectionStyle.Render("── Navigation & Vim Motions ────────────────────────"),
		row("j / k, ↓ / ↑", "Move down / up (supports [count]j, e.g. 3j)"),
		row("J / K", "Jump to Next / Previous country section"),
		row("M / L", "Jump to Middle / Bottom of screen"),
		row("Ctrl+d / Ctrl+u", "Half page down / up (PgDn / PgUp)"),
		row("gg / G", "Jump to First / Last station"),
		"",
		sectionStyle.Render("── Controls & Playback ─────────────────────────────"),
		row("Enter / Space", "Play / Stop selected station / track"),
		row("n / p", "Next / Previous track or station"),
		row("f", "Toggle station / track in Favorites"),
		row("d", "Hide non-working radio station / restore"),
		row("H", "Toggle viewing hidden radio stations"),
		row("r", "Rescan ~/Music directory for audio files"),
		row("Tab / F1-F3", "Switch tabs: Stations, Favorites, Music"),
		row("? / Esc", "Toggle / close this Help popup"),
		row("q / Ctrl+c", "Quit player"),
		"",
		sectionStyle.Render("── Music & Custom Stations ─────────────────────────"),
		bgStyle.Render("  ") + dimStyle.Render("Music Dir: ") + descStyle.Render("~/Music (*.mp3, *.lrc)"),
		bgStyle.Render("  ") + dimStyle.Render("Radio Cfg: ") + descStyle.Render("~/.config/iradio/stations.json"),
		"",
		dimStyle.Render("Press [?] or [Esc] to return to player"),
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMauve).
		Background(colorSurface).
		Padding(0, 1).
		Width(boxWidth).
		Render(content)
}

func (m Model) renderMainView() string {
	contentWidth := maxInt(40, m.width-4)
	listHeight := m.getListHeight()

	// 1. Header
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("📻 INTERNET RADIO (HK • TW • JP • SG • MY)"),
		lipgloss.NewStyle().Foreground(colorSubtext).Render(" • MPRIS2 Enabled"),
	)

	// 2. Tabs
	var tabsRow string
	stationsToRender := m.getActiveStations()
	activeCursor := m.getActiveCursor()
	currentOffset := m.getActiveOffset()
	totalItems := len(stationsToRender)

	if m.showHidden {
		hiddenTab := activeTabStyle.Render(fmt.Sprintf("👁 Hidden Stations (%d) [H/Esc to exit]", totalItems))
		tabsRow = lipgloss.JoinHorizontal(lipgloss.Center, hiddenTab)
	} else {
		visibleAll := m.getVisibleAllStations()
		favsList := m.getFavoritesList()
		favCount := len(favsList)
		musicCount := len(m.musicTracks)

		var tab1, tab2, tab3 string
		if m.activeTab == 0 {
			tab1 = activeTabStyle.Render(fmt.Sprintf("1: All Stations (%d/%d) [F1]", m.cursor+1, len(visibleAll)))
			tab2 = inactiveTabStyle.Render(fmt.Sprintf("2: Favorites (%d) [F2]", favCount))
			tab3 = inactiveTabStyle.Render(fmt.Sprintf("3: Music (%d) [F3]", musicCount))
		} else if m.activeTab == 1 {
			currentFavPos := 0
			if favCount > 0 {
				currentFavPos = m.favCursor + 1
			}
			tab1 = inactiveTabStyle.Render(fmt.Sprintf("1: All Stations (%d) [F1]", len(visibleAll)))
			tab2 = activeTabStyle.Render(fmt.Sprintf("2: Favorites (%d/%d) [F2]", currentFavPos, favCount))
			tab3 = inactiveTabStyle.Render(fmt.Sprintf("3: Music (%d) [F3]", musicCount))
		} else {
			currentMusicPos := 0
			if musicCount > 0 {
				currentMusicPos = m.musicCursor + 1
			}
			tab1 = inactiveTabStyle.Render(fmt.Sprintf("1: All Stations (%d) [F1]", len(visibleAll)))
			tab2 = inactiveTabStyle.Render(fmt.Sprintf("2: Favorites (%d) [F2]", favCount))
			tab3 = activeTabStyle.Render(fmt.Sprintf("3: Music (%d/%d) [F3]", currentMusicPos, musicCount))
		}

		hiddenBadge := ""
		if len(m.hidden) > 0 {
			hiddenBadge = lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("  [%d hidden • 'H' to view]", len(m.hidden)))
		}

		scrollInfo := ""
		if m.activeTab != 2 && totalItems > listHeight {
			endIdx := minInt(totalItems, currentOffset+listHeight)
			scrollInfo = lipgloss.NewStyle().Foreground(colorSubtext).Render(
				fmt.Sprintf("  [Showing %d-%d of %d]", currentOffset+1, endIdx, totalItems),
			)
		} else if m.activeTab == 2 && musicCount > listHeight {
			endIdx := minInt(musicCount, m.musicOffset+listHeight)
			scrollInfo = lipgloss.NewStyle().Foreground(colorSubtext).Render(
				fmt.Sprintf("  [Showing %d-%d of %d]", m.musicOffset+1, endIdx, musicCount),
			)
		}
		tabsRow = lipgloss.JoinHorizontal(lipgloss.Center, tab1, " ", tab2, " ", tab3, hiddenBadge, scrollInfo)
	}

	// If Music Tab is selected and not in Hidden mode, render the reference Terminal Music Player view
	if m.activeTab == 2 && !m.showHidden {
		return m.renderMusicView(header, tabsRow, contentWidth, listHeight)
	}

	// 3. Station List (Strictly budgeted to listHeight lines for full-screen view)
	var listLines []string

	if totalItems == 0 {
		var msg1, msg2 string
		if m.showHidden {
			msg1 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  No hidden stations.")
			msg2 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  Press 'd' on any station in normal view to hide non-working stations.")
		} else if m.activeTab == 1 {
			msg1 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  No favorite stations added yet.")
			msg2 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  Press 'f' on any station in Tab 1 to add.")
		} else {
			msg1 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  All stations are currently hidden.")
			msg2 = lipgloss.NewStyle().Foreground(colorSubtext).Render("  Press 'H' to view hidden stations and 'd' to change back.")
		}
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

				hiddenTag := ""
				if m.showHidden {
					hiddenTag = lipgloss.NewStyle().Foreground(colorPeach).Bold(true).Render("[HIDDEN] ")
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
				line := fmt.Sprintf("%s%s%s %s%s%s %s", relMarker, favStar, playBadge, hiddenTag, regionTag, rowText, dialCol)
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
	var footerText string
	if m.showHidden {
		footerText = "[Enter/Space] Play • [d] Change Back (Unhide) • [H/Esc] Exit Hidden • [?] Help • [q] Quit"
	} else {
		footerText = "[Enter/Space] Play • [f] Fav • [d] Hide • [H] Hidden • [j/k] Move • [Tab] Tab • [?] Help • [q] Quit"
	}

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

// titledPanel renders a rounded-border box with the title embedded into the
// top-left of the border, mimicking a labeled frame.
func titledPanel(title, content string, width, height int, borderColor lipgloss.Color) string {
	st := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width)
	if height > 0 {
		st = st.Height(height)
	}
	box := st.Render(content)
	if title == "" {
		return box
	}
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	total := lipgloss.Width(lines[0])
	titleStr := " " + title + " "
	tw := lipgloss.Width(titleStr)
	fill := total - 3 - tw
	if fill < 0 {
		fill = 0
	}
	bStyle := lipgloss.NewStyle().Foreground(borderColor)
	tStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)
	lines[0] = bStyle.Render("╭─") + tStyle.Render(titleStr) + bStyle.Render(strings.Repeat("─", fill)+"╮")
	return strings.Join(lines, "\n")
}

func (m Model) renderMusicView(header, tabsRow string, contentWidth, listHeight int) string {
	// 1. Top Panel: Terminal Music Player
	playerState := "stopped"
	if m.isPlaying && m.isMusic {
		playerState = "playing"
	}
	topContent := fmt.Sprintf("State: %s • Volume: %s [System (ALSA)]", playerState, m.volumeStr)
	// Content width inside the box must be contentWidth-2 so the boxed output
	// (content + 2 border chars) fits exactly into contentWidth columns.
	topBox := titledPanel("Terminal Music Player", topContent, contentWidth-2, 0, colorCyan)

	// Layout Widths. The two side-by-side boxes plus a single space separator
	// must fit inside contentWidth. Each box consumes (innerWidth + 2) columns
	// because of its left and right border glyphs.
	innerWidth := contentWidth - 5
	if innerWidth < 40 {
		innerWidth = 40
	}
	leftWidth := (innerWidth * 48) / 100
	if leftWidth < 28 {
		leftWidth = 28
	}
	rightWidth := innerWidth - leftWidth
	if rightWidth < 24 {
		rightWidth = 24
	}

	// 2. Left Panel: Library
	var libLines []string
	totalTracks := len(m.musicTracks)

	if totalTracks == 0 {
		libLines = append(libLines, lipgloss.NewStyle().Foreground(colorSubtext).Render("  No tracks found in ~/Music"))
		libLines = append(libLines, lipgloss.NewStyle().Foreground(colorSubtext).Render("  Drop .mp3 files in ~/Music and press 'r'"))
		for len(libLines) < listHeight {
			libLines = append(libLines, "")
		}
	} else {
		for row := 0; row < listHeight; row++ {
			idx := m.musicOffset + row
			if idx < totalTracks {
				t := m.musicTracks[idx]
				isSelected := (idx == m.musicCursor)
				isThisPlaying := (m.isPlaying && m.isMusic && m.musicPlaying == idx)

				relDist := absInt(idx - m.musicCursor)
				relMarker := fmt.Sprintf(" %2d ", relDist)
				if isSelected {
					relMarker = lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render(fmt.Sprintf("»%2d ", relDist))
				}

				playSymbol := " "
				if isThisPlaying {
					playSymbol = "▶"
				}

				heartSymbol := "♡"
				if m.musicFavs[t.Path] {
					heartSymbol = "♥"
				}

				itemLead := fmt.Sprintf("%s%s%s ", relMarker, playSymbol, heartSymbol)
				availW := maxInt(10, leftWidth-lipgloss.Width(itemLead)-4)
				nameText := fitWidth(t.Filename, availW)

				if isSelected {
					libLines = append(libLines, itemLead+lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(nameText))
				} else {
					libLines = append(libLines, itemLead+lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Render(nameText))
				}
			} else {
				libLines = append(libLines, "")
			}
		}
	}
	libraryBox := titledPanel("Library", strings.Join(libLines, "\n"), leftWidth, listHeight, colorYellow)

	// 3. Right Panels
	var curTrack MusicTrack
	if m.musicPlaying >= 0 && m.musicPlaying < len(m.musicTracks) {
		curTrack = m.musicTracks[m.musicPlaying]
	} else if m.musicCursor >= 0 && m.musicCursor < len(m.musicTracks) {
		curTrack = m.musicTracks[m.musicCursor]
	}

	// 3a. Now Box
	var nowLines []string
	if totalTracks > 0 {
		labelStyle := lipgloss.NewStyle().Foreground(colorMauve).Bold(true)
		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
		avail := maxInt(10, rightWidth-10)

		nowLines = append(nowLines, labelStyle.Render("Track:  ")+valStyle.Render(fitWidth(curTrack.Filename, avail)))
		nowLines = append(nowLines, labelStyle.Render("Artist: ")+valStyle.Render(fitWidth(curTrack.Artist, avail)))
		nowLines = append(nowLines, labelStyle.Render("Album:  ")+valStyle.Render(fitWidth(curTrack.Album, avail)))
		dispIdx := m.musicCursor + 1
		if m.musicPlaying >= 0 {
			dispIdx = m.musicPlaying + 1
		}
		nowLines = append(nowLines, labelStyle.Render("Index:  ")+valStyle.Render(fmt.Sprintf("%d / %d", dispIdx, totalTracks)))
	} else {
		nowLines = append(nowLines, lipgloss.NewStyle().Foreground(colorSubtext).Render("No track loaded"))
	}
	nowBox := titledPanel("Now", strings.Join(nowLines, "\n"), rightWidth, 4, colorCyan)

	// 3b. Progress Box
	var progLines []string
	elapsedSecs := int(m.trackElapsed.Seconds())
	totalSecs := int(curTrack.Duration.Seconds())
	if !m.isPlaying || !m.isMusic {
		elapsedSecs = 0
	}
	timeText := fmt.Sprintf("%02d:%02d / %02d:%02d", elapsedSecs/60, elapsedSecs%60, totalSecs/60, totalSecs%60)
	if totalSecs <= 0 {
		timeText = fmt.Sprintf("%02d:%02d / --:--", elapsedSecs/60, elapsedSecs%60)
	}

	barWidth := maxInt(5, rightWidth-lipgloss.Width(timeText)-6)
	var progBar strings.Builder
	if totalSecs > 0 && barWidth > 2 {
		ratio := float64(elapsedSecs) / float64(totalSecs)
		if ratio > 1.0 {
			ratio = 1.0
		}
		filled := int(ratio * float64(barWidth))
		progBar.WriteString(strings.Repeat("━", filled))
		progBar.WriteString("█")
		if barWidth-filled-1 > 0 {
			progBar.WriteString(strings.Repeat("─", barWidth-filled-1))
		}
	} else {
		progBar.WriteString("█")
		if barWidth > 1 {
			progBar.WriteString(strings.Repeat("─", barWidth-1))
		}
	}
	progStyled := lipgloss.NewStyle().Foreground(colorGreen).Render(progBar.String())
	progLine := fmt.Sprintf("%s   %s", progStyled, lipgloss.NewStyle().Foreground(colorSubtext).Render(timeText))
	progLines = append(progLines, progLine)

	progressBox := titledPanel("Progress", strings.Join(progLines, "\n"), rightWidth, 1, colorGreen)

	// 3c. Lyrics Box — title line + exactly 3 lyric lines (previous, current, next)
	const maxLyricLines = 3
	var lyrLines []string
	titleLine := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(fitWidth(curTrack.Filename, rightWidth-4))
	lyrLines = append(lyrLines, titleLine)

	if len(curTrack.Lyrics) > 0 && m.isPlaying && m.isMusic {
		activeIdx := 0
		for i, line := range curTrack.Lyrics {
			if line.Time <= m.trackElapsed {
				activeIdx = i
			} else {
				break
			}
		}
		start := activeIdx - 1
		if start < 0 {
			start = 0
		}
		end := start + maxLyricLines
		if end > len(curTrack.Lyrics) {
			end = len(curTrack.Lyrics)
			start = end - maxLyricLines
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			text := fitWidth(curTrack.Lyrics[i].Text, rightWidth-4)
			if i == activeIdx {
				lyrLines = append(lyrLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(text))
			} else {
				lyrLines = append(lyrLines, lipgloss.NewStyle().Foreground(colorSubtext).Render(text))
			}
		}
	} else {
		status := "(No synchronized .lrc file found)"
		if len(curTrack.Lyrics) > 0 {
			status = curTrack.Lyrics[0].Text
		}
		lyrLines = append(lyrLines, lipgloss.NewStyle().Foreground(colorSubtext).Render(status))
	}

	// Fixed height: 1 title line + 3 lyric lines.
	lyrHeight := 4
	lyricsBox := titledPanel("Lyrics", strings.Join(lyrLines, "\n"), rightWidth, lyrHeight, colorCyan)

	// Pad the right column so its total height matches the library box
	// (library renders as listHeight content lines + 2 border rows).
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, nowBox, progressBox, lyricsBox)
	if h := lipgloss.Height(rightColumn); h < listHeight+2 {
		rightColumn += strings.Repeat("\n", listHeight+2-h)
	}
	mainSplit := lipgloss.JoinHorizontal(lipgloss.Top, libraryBox, " ", rightColumn)

	footerText := "[Enter/Space] Play/Pause • [f] Fav • [n/p] Next/Prev • [r] Rescan ~/Music • [Tab] Switch • [?] Help • [q] Quit"
	footer := helpStyle.Render(footerText)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tabsRow,
		"",
		topBox,
		mainSplit,
		"",
		footer,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	mainView := m.renderMainView()
	if !m.showHelp {
		return mainView
	}

	helpBox := m.renderHelpBox()
	boxWidth := lipgloss.Width(helpBox)
	boxHeight := lipgloss.Height(helpBox)

	x := maxInt(0, (m.width-boxWidth)/2)
	y := maxInt(0, (m.height-boxHeight)/2)

	return overlay(mainView, helpBox, x, y)
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

func getHiddenFilePath() string {
	dir := ensureConfigDir()
	return filepath.Join(dir, "hidden.json")
}

func loadHidden() map[string]bool {
	hidden := make(map[string]bool)
	data, err := os.ReadFile(getHiddenFilePath())
	if err == nil {
		_ = json.Unmarshal(data, &hidden)
	}
	return hidden
}

func saveHidden(hidden map[string]bool) {
	data, err := json.MarshalIndent(hidden, "", "  ")
	if err == nil {
		_ = os.WriteFile(getHiddenFilePath(), data, 0644)
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
