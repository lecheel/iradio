// Package ui implements the Bubble Tea TUI for iradio.
package ui

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"iradio/internal/audio"
	"iradio/internal/config"
	"iradio/internal/eqcache"
	"iradio/internal/mpris"
	"iradio/internal/music"
	"iradio/internal/stations"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	// ~30 FPS tick so the LED spectrum and peaks interpolate fluidly.
	return tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// SpectrumMsg carries a real, ffmpeg/FFT-derived set of bar levels.
type SpectrumMsg struct {
	Bars []int
}

func waitForSpectrum(ch chan []int) tea.Cmd {
	return func() tea.Msg {
		bars := <-ch
		return SpectrumMsg{Bars: bars}
	}
}

// EQMode specifies the visual style of the LED equalizer.
type EQMode int

const (
	EQModeBar    EQMode = iota // solid vertical block columns (▄)
	EQModeDotBar               // filled columns of dots (•)
	EQModeDot                  // floating single dots (•)
	EQModeCircle               // floating round dots (●)
)

// Model is the top-level Bubble Tea application state.
type Model struct {
	width        int
	height       int
	activeTab    int // 0 = Stations, 1 = Favorites, 2 = Music Library
	artFrame     int
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
	player       *audio.Player
	mprisSvc     *mpris.Service
	eqDB         *eqcache.DB
	cachedEQ     *eqcache.CachedEQ
	forceLiveEQ  bool // when true, ignore the precomputed EQ cache
	playingIdx   int  // index in stations.All, or -1 if stopped
	isPlaying    bool
	isMusic      bool
	musicTracks  []music.MusicTrack
	musicPlaying int
	trackStart   time.Time
	trackElapsed time.Duration
	statusMsg    string
	bars         []int
	smoothBars   []float64 // smoothly decayed values for 30fps animation
	targetBars   []float64 // target values received from FFT or precomputed cache
	peaks        []float64 // QE-style peak-hold markers (bar-value space 0..6)
	countBuffer  string
	pendingTabID int
	showHelp     bool
	volumeStr    string
	realSpectrum bool
	eqMode       EQMode
}

// New builds the initial Model, loading favorites, hidden flags and the
// local music library from disk.
func New() *Model {
	favs := config.LoadBoolMap("favorites.json")
	hidden := config.LoadBoolMap("hidden.json")
	customCount := stations.InitCustom()
	musicList := music.LoadTracks()
	mFavs := config.LoadBoolMap("music_favorites.json")
	status := ""
	if customCount > 0 {
		status = fmt.Sprintf("Loaded %d custom station(s)", customCount)
	} else if len(musicList) > 0 {
		status = fmt.Sprintf("Scanned %d tracks from ~/Music", len(musicList))
	}

	savedEQMode := config.LoadInt("eq_mode.json", 0)
	if savedEQMode < 0 || savedEQMode > int(EQModeCircle) {
		savedEQMode = 0
	}

	eqDB, err := eqcache.Open()
	if err != nil {
		eqDB = nil // playback will fall back to realtime/simulated EQ
	} else if eqDB.Count() > 0 && status == "" {
		status = fmt.Sprintf("EQ cache: %d track(s) ready", eqDB.Count())
	}

	return &Model{
		eqDB:         eqDB,
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
		player:       &audio.Player{SpecChan: make(chan []int, 4)},
		realSpectrum: audio.HasFFmpeg(),
		playingIdx:   -1,
		isPlaying:    false,
		isMusic:      false,
		musicTracks:  musicList,
		musicPlaying: -1,
		statusMsg:    status,
		bars:         make([]int, 32),
		smoothBars:   make([]float64, 32),
		targetBars:   make([]float64, 32),
		peaks:        make([]float64, 32),
		countBuffer:  "",
		pendingTabID: 0,
		showHelp:     false,
		volumeStr:    config.SystemVolume(),
		eqMode:       EQMode(savedEQMode),
	}
}

// SetService attaches the MPRIS service to the model.
func (m *Model) SetService(s *mpris.Service) { m.mprisSvc = s }

// SetForceLiveEQ controls whether local track playback uses the precomputed
// EQ cache. When true, the cache is bypassed and every track is analysed in
// real time with the ffmpeg-driven FFT (equivalent to the --eq CLI flag).
func (m *Model) SetForceLiveEQ(v bool) { m.forceLiveEQ = v }

// Stop shuts down audio playback and reports the stopped state over MPRIS.
func (m *Model) Stop() {
	m.player.Stop()
	if m.eqDB != nil {
		_ = m.eqDB.Close()
		m.eqDB = nil
	}
	if m.mprisSvc != nil {
		m.mprisSvc.Update("Stopped", nil)
	}
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

func (m Model) getVisibleAllStations() []stations.Station {
	var list []stations.Station
	for _, st := range stations.All {
		if !m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m Model) getHiddenList() []stations.Station {
	var list []stations.Station
	for _, st := range stations.All {
		if m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m Model) getFavoritesList() []stations.Station {
	var list []stations.Station
	for _, st := range stations.All {
		if m.favorites[st.ID] && !m.hidden[st.ID] {
			list = append(list, st)
		}
	}
	return list
}

func (m Model) getActiveStations() []stations.Station {
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
	list := m.getActiveStations()
	if len(list) == 0 {
		return
	}

	curIdx := m.getActiveCursor()
	if curIdx < 0 || curIdx >= len(list) {
		curIdx = 0
	}

	currentRegion := list[curIdx].Region
	targetIdx := -1

	for i := curIdx + 1; i < len(list); i++ {
		if list[i].Region != currentRegion {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		for i := 0; i < curIdx; i++ {
			if list[i].Region != currentRegion {
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
	list := m.getActiveStations()
	if len(list) == 0 {
		return
	}

	curIdx := m.getActiveCursor()
	if curIdx < 0 || curIdx >= len(list) {
		curIdx = 0
	}

	currentRegion := list[curIdx].Region

	prevRegionIdx := -1
	for i := curIdx - 1; i >= 0; i-- {
		if list[i].Region != currentRegion {
			prevRegionIdx = i
			break
		}
	}
	if prevRegionIdx == -1 {
		for i := len(list) - 1; i > curIdx; i-- {
			if list[i].Region != currentRegion {
				prevRegionIdx = i
				break
			}
		}
	}

	if prevRegionIdx == -1 {
		return
	}

	targetRegion := list[prevRegionIdx].Region
	targetIdx := prevRegionIdx
	for i := prevRegionIdx; i >= 0; i-- {
		if list[i].Region == targetRegion {
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

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), waitForSpectrum(m.player.SpecChan))
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffsets()

	case SpectrumMsg:
		if m.isPlaying {
			n := minInt(len(msg.Bars), len(m.targetBars))
			for i := 0; i < n; i++ {
				m.targetBars[i] = float64(msg.Bars[i])
			}
		}
		cmds = append(cmds, waitForSpectrum(m.player.SpecChan))

	case tickMsg:
		if m.isPlaying {
			m.artFrame++
			if m.isMusic && m.cachedEQ != nil {
				// Precomputed EQ frames from the sqlite cache: no ffmpeg
				// decoding happens at playtime, we just index by elapsed
				// time and copy into the target bar array.
				ms := int(time.Since(m.trackStart).Milliseconds())
				if bars := m.cachedEQ.BarsAt(ms); bars != nil {
					n := minInt(len(bars), len(m.targetBars))
					for i := 0; i < n; i++ {
						m.targetBars[i] = float64(bars[i])
					}
				}
			} else if !m.realSpectrum {
				for i := range m.targetBars {
					r := rand.Intn(100)
					switch {
					case r < 55:
						// hold
					case r < 75:
						m.targetBars[i] += 1
					case r < 95:
						m.targetBars[i] -= 1
					default:
						if rand.Intn(2) == 0 {
							m.targetBars[i] += 2
						} else {
							m.targetBars[i] -= 2
						}
					}
					if m.targetBars[i] < 0 {
						m.targetBars[i] = 0
					} else if m.targetBars[i] > 6 {
						m.targetBars[i] = 6
					}
				}
			}

			// Smooth interpolation: fast attack on peaks, graceful decay on falls.
			const attack = 0.70
			const decay = 0.28
			for i := range m.bars {
				target := m.targetBars[i]
				if target > m.smoothBars[i] {
					m.smoothBars[i] += (target - m.smoothBars[i]) * attack
				} else {
					m.smoothBars[i] -= decay
					if m.smoothBars[i] < target {
						m.smoothBars[i] = target
					}
				}
				if m.smoothBars[i] < 0 {
					m.smoothBars[i] = 0
				} else if m.smoothBars[i] > 6 {
					m.smoothBars[i] = 6
				}
				m.bars[i] = int(math.Round(m.smoothBars[i]))
			}

			if len(m.peaks) != len(m.bars) {
				m.peaks = make([]float64, len(m.bars))
			}
			const peakGravity = 0.08
			for i := range m.bars {
				cur := m.smoothBars[i]
				if cur >= m.peaks[i] {
					m.peaks[i] = cur
				} else {
					m.peaks[i] -= peakGravity
					if m.peaks[i] < cur {
						m.peaks[i] = cur
					}
				}
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
				m.smoothBars[i] = 0
				m.targetBars[i] = 0
				m.peaks[i] = 0
			}
		}
		cmds = append(cmds, tickCmd())

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
			m.Stop()
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
				m.musicTracks = music.LoadTracks()
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
					config.SaveBoolMap("hidden.json", m.hidden)
					m.statusMsg = fmt.Sprintf("Restored %s to station list", target.NameEn)
					m.clampOffsets()
				} else {
					m.hidden[target.ID] = true
					config.SaveBoolMap("hidden.json", m.hidden)
					m.statusMsg = fmt.Sprintf("Hidden station: %s (Press 'H' to view hidden)", target.NameEn)
					if m.isPlaying && m.playingIdx >= 0 && stations.All[m.playingIdx].ID == target.ID {
						m.player.Stop()
						m.isPlaying = false
						m.playingIdx = -1
						if m.mprisSvc != nil {
							m.mprisSvc.Update("Stopped", nil)
						}
					}
					m.clampOffsets()
				}
			}

		case "t":
			m.pendingTabID++
			m.countBuffer = ""
			m.eqMode = (m.eqMode + 1) % 4
			config.SaveInt("eq_mode.json", int(m.eqMode))
			switch m.eqMode {
			case EQModeBar:
				m.statusMsg = "EQ Mode: Block Bar (solid ▄ columns)"
			case EQModeDotBar:
				m.statusMsg = "EQ Mode: Dot Bar (filled • columns)"
			case EQModeDot:
				m.statusMsg = "EQ Mode: Dot (floating • dots)"
			case EQModeCircle:
				m.statusMsg = "EQ Mode: Circle (dot matrix ●)"
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
					config.SaveBoolMap("music_favorites.json", m.musicFavs)
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
					config.SaveBoolMap("favorites.json", m.favorites)
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

// spectrumSource describes which backend is currently driving the LED
// equalizer bars, so the UI can label it accurately.
type spectrumSource int

const (
	spectrumIdle      spectrumSource = iota // nothing playing
	spectrumCached                          // precomputed frames from eqcache
	spectrumRealtime                        // live ffmpeg -> FFT
	spectrumSimulated                       // fallback random walk
)

func (m *Model) currentSpectrumSource() spectrumSource {
	if !m.isPlaying {
		return spectrumIdle
	}
	if m.isMusic && m.cachedEQ != nil {
		return spectrumCached
	}
	if m.realSpectrum {
		return spectrumRealtime
	}
	return spectrumSimulated
}

func (m *Model) getCurrentStation() *stations.Station {
	list := m.getActiveStations()
	c := m.getActiveCursor()
	if c >= 0 && c < len(list) {
		return &list[c]
	}
	return nil
}

func (m *Model) resetBars() {
	for i := range m.bars {
		m.bars[i] = 0
		m.smoothBars[i] = 0
		m.targetBars[i] = 0
		m.peaks[i] = 0
	}
}

func (m *Model) playMusicTrack(idx int) {
	if idx < 0 || idx >= len(m.musicTracks) {
		return
	}
	t := &m.musicTracks[idx]

	// Look up precomputed EQ first. If we have it, we can skip both the
	// realtime ffmpeg FFT and the ffprobe duration probe. When --eq is in
	// effect we deliberately ignore the cache so the LED meter reflects
	// the file as decoded live.
	var cached *eqcache.CachedEQ
	if !m.forceLiveEQ && m.eqDB != nil {
		cached = m.eqDB.Lookup(t.Path)
	}
	m.cachedEQ = cached

	if t.Duration == 0 {
		if cached != nil && cached.DurationMs > 0 {
			t.Duration = time.Duration(cached.DurationMs) * time.Millisecond
		} else {
			go func(path string, index int) {
				d := music.ProbeDuration(path)
				if d > 0 && index < len(m.musicTracks) {
					m.musicTracks[index].Duration = d
				}
			}(t.Path, idx)
		}
	}

	var err error
	if cached != nil {
		err = m.player.Play(t.Path, audio.WithoutAnalyzer())
	} else {
		err = m.player.Play(t.Path)
	}
	if err != nil {
		m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
		m.isPlaying = false
		m.isMusic = false
		m.cachedEQ = nil
		m.resetBars()
		return
	}

	m.resetBars()
	m.isPlaying = true
	m.isMusic = true
	m.musicPlaying = idx
	m.playingIdx = -1
	m.trackStart = time.Now()
	m.trackElapsed = 0
	switch {
	case cached != nil:
		m.statusMsg = fmt.Sprintf("Playing track (cached EQ): %s", t.Filename)
	case m.forceLiveEQ:
		m.statusMsg = fmt.Sprintf("Playing track (live EQ): %s", t.Filename)
	default:
		m.statusMsg = fmt.Sprintf("Playing track: %s", t.Filename)
	}
	if m.mprisSvc != nil {
		m.mprisSvc.UpdateMusic("Playing", t)
	}
}

func (m *Model) togglePlayMusic() {
	if len(m.musicTracks) == 0 {
		return
	}
	if m.isPlaying && m.isMusic && m.musicPlaying == m.musicCursor {
		m.player.Stop()
		m.isPlaying = false
		m.isMusic = false
		m.cachedEQ = nil
		m.resetBars()
		m.statusMsg = fmt.Sprintf("Stopped: %s", m.musicTracks[m.musicCursor].Filename)
		if m.mprisSvc != nil {
			m.mprisSvc.UpdateMusic("Stopped", nil)
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
	for idx, s := range stations.All {
		if s.ID == target.ID {
			targetIdx = idx
			break
		}
	}

	if m.isPlaying && !m.isMusic && m.playingIdx == targetIdx {
		m.player.Stop()
		m.isPlaying = false
		m.resetBars()
		m.statusMsg = fmt.Sprintf("Stopped: %s", target.NameEn)
		if m.mprisSvc != nil {
			m.mprisSvc.Update("Stopped", nil)
		}
		return
	}

	err := m.player.Play(target.StreamURL)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Audio Error: %v", err)
		m.isPlaying = false
		m.resetBars()
		return
	}

	m.resetBars()
	m.isPlaying = true
	m.isMusic = false
	m.musicPlaying = -1
	m.playingIdx = targetIdx
	m.cachedEQ = nil
	m.statusMsg = fmt.Sprintf("Playing live: [%s] %s", target.Region, target.NameEn)
	if m.mprisSvc != nil {
		m.mprisSvc.Update("Playing", target)
	}
}

func (m *Model) selectNextStation() {
	list := m.getActiveStations()
	if len(list) == 0 {
		return
	}
	curr := -1
	if m.playingIdx >= 0 && m.playingIdx < len(stations.All) {
		for i, st := range list {
			if st.ID == stations.All[m.playingIdx].ID {
				curr = i
				break
			}
		}
	}
	nextIdx := (curr + 1) % len(list)
	st := &list[nextIdx]
	for idx, s := range stations.All {
		if s.ID == st.ID {
			m.playingIdx = idx
			break
		}
	}
	_ = m.player.Play(st.StreamURL)
	m.isPlaying = true
	m.isMusic = false
	m.cachedEQ = nil
	if m.mprisSvc != nil {
		m.mprisSvc.Update("Playing", st)
	}
}

func (m *Model) selectPrevStation() {
	list := m.getActiveStations()
	if len(list) == 0 {
		return
	}
	curr := -1
	if m.playingIdx >= 0 && m.playingIdx < len(stations.All) {
		for i, st := range list {
			if st.ID == stations.All[m.playingIdx].ID {
				curr = i
				break
			}
		}
	}
	prevIdx := (curr - 1 + len(list)) % len(list)
	st := &list[prevIdx]
	for idx, s := range stations.All {
		if s.ID == st.ID {
			m.playingIdx = idx
			break
		}
	}
	_ = m.player.Play(st.StreamURL)
	m.isPlaying = true
	m.isMusic = false
	m.cachedEQ = nil
	if m.mprisSvc != nil {
		m.mprisSvc.Update("Playing", st)
	}
}
