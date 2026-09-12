package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"iradio/internal/music"
	"iradio/internal/stations"
)

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
		row("t", "Toggle EQ display mode (Bar / Dot Bar / Dot / Circle)"),
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

func (m *Model) renderMainView() string {
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

	// If Music Tab is selected and not in Hidden mode, render the music view.
	if m.activeTab == 2 && !m.showHidden {
		return m.renderMusicView(header, tabsRow, contentWidth, listHeight)
	}

	// 3. Station List
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
				isThisPlaying := (m.isPlaying && m.playingIdx >= 0 && stations.All[m.playingIdx].ID == st.ID)

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

	// 4. Now Playing / EQ Panel
	var eqBar strings.Builder
	eqChars := []string{" ", " ", "▂", "▃", "▄", "▅", "▆", "▇"}
	if m.eqMode == EQModeDot || m.eqMode == EQModeDotBar {
		eqChars = []string{" ", "·", "•", "•", "•", "•", "•", "•"}
	} else if m.eqMode == EQModeCircle {
		eqChars = []string{" ", "·", "•", "•", "●", "●", "●", "●"}
	}
	for _, val := range m.bars {
		eqBar.WriteString(eqChars[val])
	}

	cardInnerWidth := maxInt(20, contentWidth-4)
	lblTitle := lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render(fitWidth("▶ Title:", 12))
	lblDesc := lipgloss.NewStyle().Foreground(colorSubtext).Render(fitWidth("  Describe:", 12))
	lblNotice := lipgloss.NewStyle().Foreground(colorYellow).Render(fitWidth("  Notice:", 12))

	var cardLine1, cardLine2, cardLine3 string
	if m.isPlaying && m.playingIdx >= 0 {
		curr := stations.All[m.playingIdx]
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
		footerText = "[Enter/Space] Play • [t] EQ Mode • [d] Unhide • [H/Esc] Exit Hidden • [?] Help • [q] Quit"
	} else {
		footerText = "[Enter/Space] Play • [t] EQ Mode • [f] Fav • [d] Hide • [H] Hidden • [Tab] Tab • [?] Help • [q] Quit"
	}

	var footer string
	if m.countBuffer != "" {
		countBadge := lipgloss.NewStyle().Background(colorMauve).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Render(fmt.Sprintf(" Count: %s ", m.countBuffer))
		footer = lipgloss.JoinHorizontal(lipgloss.Center, countBadge, "  ", helpStyle.Render(footerText))
	} else {
		footer = helpStyle.Render(footerText)
	}

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

func (m Model) renderMusicView(header, tabsRow string, contentWidth, listHeight int) string {
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

	// Left Panel: Library
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

	// Right Panels
	var curTrack music.MusicTrack
	if m.musicPlaying >= 0 && m.musicPlaying < len(m.musicTracks) {
		curTrack = m.musicTracks[m.musicPlaying]
	} else if m.musicCursor >= 0 && m.musicCursor < len(m.musicTracks) {
		curTrack = m.musicTracks[m.musicCursor]
	}

	// Now Box
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

	// Progress Box
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

	// Lyrics Box
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

	lyrHeight := 4
	lyricsBox := titledPanel("Lyrics", strings.Join(lyrLines, "\n"), rightWidth, lyrHeight, colorCyan)

	// LED spectrum analyzer
	used := lipgloss.Height(nowBox) + lipgloss.Height(progressBox) + lipgloss.Height(lyricsBox)
	artInner := (listHeight + 2) - used - 2
	if artInner < 6 {
		artInner = 6
	}

	spectrumRows := artInner - 4
	if spectrumRows < 3 {
		spectrumRows = 3
	}

	// Calculate how many spectrum columns fit across the full width of the right panel
	availBarCols := rightWidth - 4
	if availBarCols < 7 {
		availBarCols = 7
	}
	targetBars := (availBarCols + 1) / 2
	if targetBars < 4 {
		targetBars = 4
	}

	// Resample the 32 frequency bands across targetBars to seamlessly fill
	// the entire panel width when the terminal is enlarged or full screen.
	dispBars := make([]float64, targetBars)
	dispPeaks := make([]float64, targetBars)
	srcLen := len(m.bars)
	for b := 0; b < targetBars; b++ {
		var srcPos float64
		if targetBars > 1 {
			srcPos = float64(b) * float64(srcLen-1) / float64(targetBars-1)
		}
		idx := int(srcPos)
		frac := srcPos - float64(idx)
		if idx >= srcLen-1 {
			dispBars[b] = float64(m.bars[srcLen-1])
			if srcLen-1 < len(m.peaks) {
				dispPeaks[b] = m.peaks[srcLen-1]
			}
		} else {
			dispBars[b] = float64(m.bars[idx])*(1.0-frac) + float64(m.bars[idx+1])*frac
			p0, p1 := 0.0, 0.0
			if idx < len(m.peaks) {
				p0 = m.peaks[idx]
			}
			if idx+1 < len(m.peaks) {
				p1 = m.peaks[idx+1]
			}
			dispPeaks[b] = p0*(1.0-frac) + p1*frac
		}
	}

	const barMax = 6
	const spectrumMinHz = 40.0
	const spectrumMaxHz = 11025.0

	peakVal, sumVal := 0.0, 0.0
	for _, v := range dispBars {
		if v > peakVal {
			peakVal = v
		}
		sumVal += v
	}
	avgVal := 0.0
	if len(dispBars) > 0 {
		avgVal = sumVal / float64(len(dispBars))
	}

	baseW := targetBars*2 - 1
	if baseW < 1 {
		baseW = 1
	}

	// Symmetrical stereo meters matched to the exact width of the spectrum baseline
	meterW := (baseW - 5) / 2
	if meterW < 6 {
		meterW = 6
	}
	renderMeter := func(level float64, c lipgloss.Color) string {
		filled := int(math.Round(level * float64(meterW) / float64(barMax)))
		if filled > meterW {
			filled = meterW
		}
		if filled < 0 {
			filled = 0
		}
		return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(colorSurface).Render(strings.Repeat("░", meterW-filled))
	}
	stereoLine := "  " +
		lipgloss.NewStyle().Foreground(colorSubtext).Bold(true).Render("L ") +
		renderMeter(avgVal, colorGreen) + " " +
		lipgloss.NewStyle().Foreground(colorSubtext).Bold(true).Render("R ") +
		renderMeter(peakVal, colorCyan)

	activeGlyph := "▄"
	offGlyph := "▄"
	peakGlyph := "▄"
	switch m.eqMode {
	case EQModeDotBar:
		activeGlyph = "•"
		offGlyph = "·"
		peakGlyph = "•"
	case EQModeDot:
		activeGlyph = "•"
		offGlyph = "·"
		peakGlyph = "•"
	case EQModeCircle:
		activeGlyph = "●"
		offGlyph = "·"
		peakGlyph = "●"
	}

	isBarStyle := (m.eqMode == EQModeBar || m.eqMode == EQModeDotBar)

	peakStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	var spectrumLines []string
	for row := 0; row < spectrumRows; row++ {
		rowFromBottom := spectrumRows - 1 - row
		ratio := 0.0
		if spectrumRows > 1 {
			ratio = float64(rowFromBottom) / float64(spectrumRows-1)
		}
		var rowColor lipgloss.Color
		switch {
		case ratio > 0.85:
			rowColor = colorPink
		case ratio > 0.60:
			rowColor = colorPeach
		case ratio > 0.35:
			rowColor = colorYellow
		default:
			rowColor = colorGreen
		}
		onStyle := lipgloss.NewStyle().Foreground(rowColor)
		offStyle := lipgloss.NewStyle().Foreground(colorSurface)

		var sb strings.Builder
		sb.WriteString("  ")
		for i := 0; i < targetBars; i++ {
			v := dispBars[i]
			filledHeight := int(math.Round(v * float64(spectrumRows) / float64(barMax)))

			peakRow := -1
			if dispPeaks[i] > 0 {
				peakRow = int(dispPeaks[i]*float64(spectrumRows)/float64(barMax) + 0.5)
				if peakRow > spectrumRows-1 {
					peakRow = spectrumRows - 1
				}
			}

			isPeak := (rowFromBottom == peakRow && peakRow >= 0)
			var isLevel bool
			if isBarStyle {
				// Filled column up to the current level
				isLevel = (filledHeight > rowFromBottom)
			} else {
				// Single floating dot mode
				isLevel = (filledHeight > 0 && rowFromBottom == filledHeight-1)
			}

			switch {
			case isPeak && (!isLevel || !isBarStyle):
				sb.WriteString(peakStyle.Render(peakGlyph))
			case isLevel:
				sb.WriteString(onStyle.Render(activeGlyph))
			default:
				sb.WriteString(offStyle.Render(offGlyph))
			}
			if i < targetBars-1 {
				sb.WriteString(" ")
			}
		}
		spectrumLines = append(spectrumLines, sb.String())
	}

	baseline := "  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(strings.Repeat("▀", baseW))

	freqPoints := []struct {
		hz    float64
		label string
	}{
		{50, "50"}, {100, "100"}, {250, "250"}, {500, "500"},
		{1000, "1k"}, {2000, "2k"}, {4000, "4k"}, {8000, "8k"}, {10000, "10k"},
	}
	logMin := math.Log10(spectrumMinHz)
	logMax := math.Log10(spectrumMaxHz)

	axis := []rune(strings.Repeat(" ", baseW))
	lastEnd := -1
	for _, fp := range freqPoints {
		if fp.hz < spectrumMinHz || fp.hz > spectrumMaxHz {
			continue
		}
		bin := int((math.Log10(fp.hz)-logMin)/(logMax-logMin)*float64(targetBars) + 0.5)
		if bin >= targetBars {
			continue
		}
		col := bin * 2
		if col <= lastEnd || col+len(fp.label) > baseW {
			continue
		}
		for i, r := range fp.label {
			axis[col+i] = r
		}
		lastEnd = col + len(fp.label)
	}
	freqLabel := "  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(string(axis))

	// Label the spectrum source so the user can tell at a glance whether
	// the LED bars come from the preloaded EQ cache, a live ffmpeg FFT,
	// or the simulated fallback.
	var status string
	var statusColor lipgloss.Color
	switch m.currentSpectrumSource() {
	case spectrumCached:
		status = "◉ CACHED EQ"
		statusColor = colorCyan
	case spectrumRealtime:
		status = "◉ LIVE FFT"
		statusColor = colorGreen
	case spectrumSimulated:
		status = "◉ SIMULATED"
		statusColor = colorPeach
	default:
		status = "◌ IDLE"
		statusColor = colorSubtext
	}

	modeTag := "BAR"
	switch m.eqMode {
	case EQModeDotBar:
		modeTag = "DOT BAR"
	case EQModeDot:
		modeTag = "DOT"
	case EQModeCircle:
		modeTag = "CIRCLE"
	}

	infoLine := "  " +
		lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(status) +
		lipgloss.NewStyle().Foreground(colorMauve).Bold(true).Render("  ["+modeTag+"]") +
		lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("  Peak %d/%d", int(math.Round(peakVal)), barMax))
	eqLines := make([]string, 0, artInner)
	eqLines = append(eqLines, stereoLine)
	eqLines = append(eqLines, spectrumLines...)
	eqLines = append(eqLines, baseline, freqLabel, infoLine)
	for len(eqLines) < artInner {
		eqLines = append(eqLines, "")
	}
	if len(eqLines) > artInner {
		eqLines = eqLines[:artInner]
	}

	eqColor := colorPeach
	if m.isPlaying {
		eqColors := []lipgloss.Color{colorPeach, colorCyan, colorGreen, colorMauve}
		eqColor = eqColors[(m.artFrame/5)%len(eqColors)]
	}
	artBox := titledPanel("LED Equalizer", strings.Join(eqLines, "\n"), rightWidth, artInner, eqColor)

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, nowBox, progressBox, lyricsBox, artBox)
	if h := lipgloss.Height(rightColumn); h < listHeight+2 {
		rightColumn += strings.Repeat("\n", listHeight+2-h)
	}
	mainSplit := lipgloss.JoinHorizontal(lipgloss.Top, libraryBox, " ", rightColumn)

	footerText := "[Enter/Space] Play/Pause • [t] EQ Mode • [f] Fav • [n/p] Next/Prev • [r] Rescan • [Tab] Switch • [?] Help • [q] Quit"
	footer := helpStyle.Render(footerText)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tabsRow,
		"",
		mainSplit,
		"",
		footer,
	)

	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

// View implements tea.Model.
func (m *Model) View() string {
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
