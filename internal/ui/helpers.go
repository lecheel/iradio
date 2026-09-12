package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
