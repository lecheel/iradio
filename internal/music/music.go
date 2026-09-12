// Package music loads the local ~/Music library, ID3 metadata and LRC lyrics.
package music

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// LyricLine is a single timed line of an .lrc file.
type LyricLine struct {
	Time time.Duration
	Text string
}

// MusicTrack describes one local audio file.
type MusicTrack struct {
	Path     string
	Filename string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
	Lyrics   []LyricLine
}

// ParseLRCFile parses a standard .lrc file into timed lyric lines.
func ParseLRCFile(lrcPath string) []LyricLine {
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

// ProbeDuration uses ffprobe to determine the length of an audio file.
func ProbeDuration(path string) time.Duration {
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

// LoadTracks walks ~/Music and returns every supported audio file it finds.
func LoadTracks() []MusicTrack {
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
			lyrics = ParseLRCFile(lrcPath)
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