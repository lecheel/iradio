// Package main is the entry point for iradio.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"iradio/internal/config"
	"iradio/internal/eqcache"
	"iradio/internal/mpris"
	"iradio/internal/music"
	"iradio/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "-help", "help":
			printUsage(os.Stdout)
			return
		case "--version", "-v", "-version":
			fmt.Printf("iradio %s\n", version)
			return
		case "--process_eq", "-process_eq":
			if err := runProcessEQ(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "process_eq: %v\n", err)
				os.Exit(1)
			}
			return
		default:
			if strings.HasPrefix(os.Args[1], "-") {
				fmt.Fprintf(os.Stderr, "iradio: unknown flag %q\n\n", os.Args[1])
				printUsage(os.Stderr)
				os.Exit(2)
			}
		}
	}

	config.WriteExampleStations()

	m := ui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())

	svc := mpris.Start(p)
	m.SetService(svc)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running player: %v\n", err)
		os.Exit(1)
	}

	m.Stop()
}

// version is the reported program version. Override at build time with
//
//	go build -ldflags "-X main.version=v1.2.3"
var version = "dev"

// printUsage writes the top-level CLI help text.
func printUsage(w io.Writer) {
	fmt.Fprintf(w, `iradio %s — terminal internet radio & local music player

USAGE:
  iradio [FLAGS]                 Launch the interactive TUI player
  iradio --process_eq [FILTER]   Precompute EQ data for local tracks

FLAGS:
  -h, --help           Show this help message and exit
  -v, --version        Print the program version and exit

SUBCOMMANDS:
  --process_eq [FILTER]
        Walk ~/Music, run ffmpeg + FFT over each supported audio file
        and store the resulting spectrum frames in the SQLite EQ cache
        (~/.cache/iradio/eq.sqlite). Once cached, playback renders the
        LED equalizer by reading frames directly, with no ffmpeg/FFT
        work and no ffprobe duration probe at playtime.

        FILTER is an optional case-insensitive substring matched against
        each track's filename; when given, only matching tracks are
        processed.

        Re-running only analyzes files that are new or whose mtime/size
        changed since the last run, so it is safe to run repeatedly.

INTERACTIVE TUI KEYBINDINGS:
  Enter / Space        Play / stop the selected station or track
  n / p                Next / previous track or station
  f                    Toggle station / track in Favorites
  d                    Hide (or restore) the selected radio station
  H                    Toggle viewing hidden radio stations
  r                    Rescan ~/Music for audio files
  Tab / F1-F3          Switch tabs: Stations, Favorites, Music
  j / k, ↑ / ↓         Move down / up (supports [count]j, e.g. 3j)
  J / K                Jump to next / previous country section
  gg / G               Jump to first / last item
  ? / Esc              Toggle / close the in-app help popup
  q / Ctrl+c           Quit

FILES:
  Config    ~/.config/iradio/   (favorites.json, hidden.json,
                                 music_favorites.json, stations.json)
  Cache     ~/.cache/iradio/    (eq.sqlite)
  Music     ~/Music/            (*.mp3, *.flac, *.m4a, *.wav, *.ogg,
                                 plus sidecar *.lrc lyric files)

EXAMPLES:
  iradio                         # launch the player
  iradio --process_eq            # cache EQ for every track in ~/Music
  iradio --process_eq jazz       # cache EQ only for tracks matching "jazz"

`, version)
}

// runProcessEQ is the implementation of the --process_eq subcommand. It
// precomputes FFT spectrum frames for the local music library and stores
// them in the sqlite EQ cache so playback can render the equalizer without
// running ffmpeg or an FFT at runtime.
func runProcessEQ(extra []string) error {
	fmt.Println("Preprocessing EQ data for tracks in ~/Music ...")

	tracks := music.LoadTracks()
	if len(tracks) == 0 {
		return fmt.Errorf("no tracks found in ~/Music")
	}

	db, err := eqcache.Open()
	if err != nil {
		return fmt.Errorf("open eq cache: %w", err)
	}
	defer db.Close()

	var filter string
	if len(extra) > 0 {
		filter = strings.ToLower(extra[0])
	}

	var processed, skipped, failed int
	for i, t := range tracks {
		if filter != "" && !strings.Contains(strings.ToLower(t.Filename), filter) {
			continue
		}
		if db.Lookup(t.Path) != nil {
			skipped++
			fmt.Printf("[%d/%d] skip (cached): %s\n", i+1, len(tracks), t.Filename)
			continue
		}
		fmt.Printf("[%d/%d] analyzing: %s\n", i+1, len(tracks), t.Filename)
		eq, err := eqcache.Compute(t.Path)
		if err != nil {
			failed++
			fmt.Printf("    error: %v\n", err)
			continue
		}
		if err := db.Store(t.Path, eq); err != nil {
			failed++
			fmt.Printf("    store error: %v\n", err)
			continue
		}
		processed++
		frames := 0
		if eq.NumBars > 0 {
			frames = len(eq.Bars) / eq.NumBars
		}
		fmt.Printf("    stored %d frames @ %d ms hop (%d bars/frame)\n",
			frames, eq.HopMs, eq.NumBars)
	}

	fmt.Printf("\nDone. processed=%d skipped=%d failed=%d (total %d)\n",
		processed, skipped, failed, len(tracks))
	return nil
}
