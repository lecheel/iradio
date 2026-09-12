// Package main is the entry point for iradio.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"iradio/internal/config"
	"iradio/internal/eqcache"
	"iradio/internal/mpris"
	"iradio/internal/music"
	"iradio/internal/ui"
)

func main() {
	args := os.Args[1:]
	forceLiveEQ := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h", "-help", "help":
			printUsage(os.Stdout)
			return
		case "--version", "-v", "-version":
			fmt.Printf("iradio %s\n", version)
			return
		case "--process_eq", "-process_eq":
			// Consumes everything after it (optional recreate flag and filename filter).
			if err := runProcessEQ(args[i+1:]); err != nil {
				fmt.Fprintf(os.Stderr, "process_eq: %v\n", err)
				os.Exit(1)
			}
			return
		case "--recreate_eq", "-recreate_eq":
			if err := runProcessEQ(append([]string{"--recreate"}, args[i+1:]...)); err != nil {
				fmt.Fprintf(os.Stderr, "recreate_eq: %v\n", err)
				os.Exit(1)
			}
			return
		case "--eq", "-eq":
			forceLiveEQ = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "iradio: unknown flag %q\n\n", args[i])
				printUsage(os.Stderr)
				os.Exit(2)
			}
		}
	}

	config.WriteExampleStations()

	m := ui.New()
	m.SetForceLiveEQ(forceLiveEQ)
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
  iradio [FLAGS]                 Launch the interactive TUI player
  iradio --process_eq [FLAGS]    Precompute EQ data for local tracks
  iradio --recreate_eq [FILTER]  Wipe and recompute EQ data for local tracks

FLAGS:
  -h, --help           Show this help message and exit
  -v, --version        Print the program version and exit
      --eq             Force live (realtime) FFT spectrum analysis

SUBCOMMANDS:
  --process_eq [--recreate] [FILTER]
        Walk ~/Music, run ffmpeg + FFT over each supported audio file
        and store the resulting spectrum frames in the SQLite EQ cache
        (~/.cache/iradio/eq.sqlite). Once cached, playback renders the
        LED equalizer by reading frames directly, with no ffmpeg/FFT
        work and no ffprobe duration probe at playtime.

        Options:
          --recreate, -f  Clear existing cached tracks and re-analyze
                          every audio file from scratch.
          FILTER          Optional case-insensitive substring matched
                          against each track's filename.

  --recreate_eq [FILTER]
        Shortcut for 'iradio --process_eq --recreate [FILTER]'.

EQ MODES:
  By default, local tracks use the precomputed cache from --process_eq
  when it is available (LED Equalizer status: "◉ CACHED EQ") and fall
  back to a live ffmpeg FFT for uncached tracks and radio streams
  ("◉ LIVE FFT"). Passing --eq disables the cache lookup entirely, so
  every played track is decoded live for its spectrum ("◉ LIVE FFT"),
  which is useful when the cache is stale, the source changes often,
  or you simply want the analyser to reflect the file as it is now.

INTERACTIVE TUI KEYBINDINGS:
  Enter / Space        Play / stop the selected station or track
  t                    Toggle LED equalizer mode (Bar / Dot Bar / Dot / Circle)
  n / p                Next / previous track or station
  f                    Toggle station / track in Favorites
  d                    Hide (or restore) the selected radio station
  H                    Toggle viewing hidden radio stations
  r                    Rescan ~/Music for audio files
  Tab / F1-F3          Switch tabs: Stations, Favorites, Music
  F5                   Toggle Music tab layout between 5:5 and 3:8 split

  Local tracks remember their last playback position: pausing, skipping or
  quitting stores the current offset (in ~/.config/iradio/last_positions.json)
  and the next play resumes from there. Finished tracks restart from 0:00.
  j / k, ↑ / ↓         Move down / up (supports [count]j, e.g. 3j)
  J / K                Jump to next / previous country section
  gg / G               Jump to first / last item
  ? / Esc              Toggle / close the in-app help popup
  q / Ctrl+c           Quit

  FILES:
    Config    ~/.config/iradio/   (favorites.json, hidden.json,
                                   music_favorites.json, eq_mode.json,
                                   last_tab.json, last_positions.json,
                                   last_music_track.txt, music_split.json,
                                   stations.json)
  Cache     ~/.cache/iradio/    (eq.sqlite)
  Music     ~/Music/            (*.mp3, *.flac, *.m4a, *.wav, *.ogg,
                                 plus sidecar *.lrc lyric files)

EXAMPLES:
  iradio                         # launch the player
  iradio --eq                    # launch the player, force live FFT EQ
  iradio --process_eq            # cache EQ for every track in ~/Music
  iradio --process_eq --recreate # recompute and overwrite existing cache
  iradio --recreate_eq           # shortcut to recompute all tracks
  iradio --process_eq jazz       # cache EQ only for tracks matching "jazz"

`, version)
}

// runProcessEQ is the implementation of the --process_eq subcommand. It
// precomputes FFT spectrum frames for the local music library and stores
// them in the sqlite EQ cache so playback can render the equalizer without
// running ffmpeg or an FFT at runtime.
func runProcessEQ(extra []string) error {
	recreate := false
	var filter string

	for _, arg := range extra {
		lower := strings.ToLower(arg)
		if lower == "--recreate" || lower == "-recreate" || lower == "recreate" || lower == "--force" || lower == "-f" {
			recreate = true
		} else if !strings.HasPrefix(arg, "-") && filter == "" {
			filter = lower
		}
	}

	tracks := music.LoadTracks()
	if len(tracks) == 0 {
		return fmt.Errorf("no tracks found in ~/Music")
	}

	db, err := eqcache.Open()
	if err != nil {
		return fmt.Errorf("open eq cache: %w", err)
	}
	defer db.Close()

	if recreate {
		fmt.Println("Recreating EQ cache (clearing existing cached entries)...")
		if err := db.ClearAll(); err != nil {
			return fmt.Errorf("clear eq cache: %w", err)
		}
	} else {
		fmt.Println("Preprocessing EQ data for tracks in ~/Music ...")
	}

	// Build the work list, pruning cached and filtered-out tracks.
	type job struct {
		track music.MusicTrack
	}
	var work []job
	var skipped int
	for _, t := range tracks {
		if filter != "" && !strings.Contains(strings.ToLower(t.Filename), filter) {
			continue
		}
		if db.Lookup(t.Path) != nil {
			skipped++
			continue
		}
		work = append(work, job{t})
	}

	if len(work) == 0 {
		fmt.Printf("Nothing to do: %d track(s) already cached.\n", skipped)
		return nil
	}

	// Parallelize across cores. Each worker spawns its own ffmpeg and runs
	// the (allocation-free) FFT loop; sqlite writes are serialized back on
	// the main goroutine.
	workers := runtime.NumCPU()
	if workers > len(work) {
		workers = len(work)
	}
	if workers < 1 {
		workers = 1
	}

	fmt.Printf("Preprocessing %d track(s) with %d worker(s) (skipped %d cached)...\n\n",
		len(work), workers, skipped)

	type result struct {
		track music.MusicTrack
		eq    *eqcache.CachedEQ
		err   error
	}

	jobs := make(chan job)
	results := make(chan result, len(work))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				eq, err := eqcache.Compute(j.track.Path)
				results <- result{j.track, eq, err}
			}
		}()
	}
	go func() {
		for _, j := range work {
			jobs <- j
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var processed, failed int
	completed := 0
	start := time.Now()
	for r := range results {
		completed++
		if r.err != nil {
			failed++
			fmt.Printf("[%d/%d] FAILED %s: %v\n", completed, len(work), r.track.Filename, r.err)
			continue
		}
		if err := db.Store(r.track.Path, r.eq); err != nil {
			failed++
			fmt.Printf("[%d/%d] store error for %s: %v\n", completed, len(work), r.track.Filename, err)
			continue
		}
		processed++
		frames := 0
		if r.eq.NumBars > 0 {
			frames = len(r.eq.Bars) / r.eq.NumBars
		}
		fmt.Printf("[%d/%d] %s (%d frames @ %d ms)\n",
			completed, len(work), r.track.Filename, frames, r.eq.HopMs)
	}

	fmt.Printf("\nDone in %s. processed=%d skipped=%d failed=%d (total %d)\n",
		time.Since(start).Round(time.Millisecond),
		processed, skipped, failed, len(tracks))
	return nil
}
