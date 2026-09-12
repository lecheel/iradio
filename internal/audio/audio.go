// Package audio handles playback of live streams and local tracks, plus the
// real-time FFT spectrum analyzer that drives the LED equalizer.
package audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/cmplx"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Player plays audio via mpv/ffplay and optionally drives a real FFT
// spectrum analyzer via ffmpeg.
type Player struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	SpecChan chan []int
}

// HasFFmpeg reports whether ffmpeg is available for real-spectrum decoding.
func HasFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
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

// PlayOption customises a Play call.
type PlayOption func(*playOptions)

type playOptions struct {
	skipAnalyzer bool
}

// WithoutAnalyzer disables the realtime ffmpeg spectrum analyzer. Use this
// when precomputed EQ frames (from internal/eqcache) will drive the UI
// instead, so we don't spin up a second ffmpeg decoding process at playtime.
func WithoutAnalyzer() PlayOption {
	return func(o *playOptions) { o.skipAnalyzer = true }
}

// Play starts playback of the given URL (stream or local file).
func (p *Player) Play(url string, opts ...PlayOption) error {
	var po playOptions
	for _, opt := range opts {
		opt(&po)
	}

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

	if !po.skipAnalyzer && p.SpecChan != nil && HasFFmpeg() {
		p.startSpectrumAnalyzer(ctx, url)
	}

	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

// Stop cancels the currently running playback process.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.drainSpecChan()
}

func (p *Player) drainSpecChan() {
	if p.SpecChan == nil {
		return
	}
	for {
		select {
		case <-p.SpecChan:
		default:
			return
		}
	}
}

// startSpectrumAnalyzer decodes the same source to raw mono PCM via ffmpeg
// and FFTs it in real time, pushing 0-6 scaled bar levels to SpecChan.
func (p *Player) startSpectrumAnalyzer(ctx context.Context, source string) {
	const sampleRate = 22050
	const chunkSize = 1024
	const numBars = 32

	args := []string{
		"-v", "quiet",
		"-nostdin",
	}

	isNetwork := strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
	if isNetwork {
		args = append(args,
			"-reconnect", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "5",
		)
	} else {
		// Read at native 1x playback rate for local files so ffmpeg decodes in real time
		args = append(args, "-re")
	}

	args = append(args,
		"-i", source,
		"-f", "f32le",
		"-ac", "1",
		"-ar", strconv.Itoa(sampleRate),
		"pipe:1",
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	go func() {
		defer cmd.Wait()

		chunkDuration := time.Duration(chunkSize) * time.Second / time.Duration(sampleRate)
		ticker := time.NewTicker(chunkDuration)
		defer ticker.Stop()

		buf := make([]byte, chunkSize*4)
		samples := make([]float64, chunkSize)
		for {
			if _, err := io.ReadFull(stdout, buf); err != nil {
				return
			}
			for i := 0; i < chunkSize; i++ {
				bits := binary.LittleEndian.Uint32(buf[i*4:])
				samples[i] = float64(math.Float32frombits(bits))
			}
			bars := computeSpectrumBars(samples, sampleRate, numBars)

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			select {
			case p.SpecChan <- bars:
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
}

func fft(a []complex128) {
	n := len(a)
	if n <= 1 {
		return
	}
	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = a[2*i]
		odd[i] = a[2*i+1]
	}
	fft(even)
	fft(odd)
	for k := 0; k < n/2; k++ {
		t := cmplx.Rect(1, -2*math.Pi*float64(k)/float64(n)) * odd[k]
		a[k] = even[k] + t
		a[k+n/2] = even[k] - t
	}
}

// computeSpectrumBars windows + FFTs a chunk of mono samples and bins the
// magnitude spectrum into numBars log-spaced bands scaled to 0-6.
func computeSpectrumBars(samples []float64, sampleRate int, numBars int) []int {
	n := len(samples)
	data := make([]complex128, n)
	for i, s := range samples {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		data[i] = complex(s*w, 0)
	}
	fft(data)

	mags := make([]float64, n/2)
	// Normalise by 4/N to compensate for the Hann window's coherent gain
	// (0.5). Without this, FFT magnitudes for full-scale audio sit around
	// +40 dB and every band pegs at the maximum level.
	norm := 4.0 / float64(n)
	for i := range mags {
		mags[i] = cmplx.Abs(data[i]) * norm
	}

	bars := make([]int, numBars)
	minHz, maxHz := 40.0, float64(sampleRate)/2
	logMin, logMax := math.Log10(minHz), math.Log10(maxHz)
	binHz := float64(sampleRate) / float64(n)

	for b := 0; b < numBars; b++ {
		f0 := math.Pow(10, logMin+(logMax-logMin)*float64(b)/float64(numBars))
		f1 := math.Pow(10, logMin+(logMax-logMin)*float64(b+1)/float64(numBars))
		i0 := maxInt(1, int(f0/binHz))
		i1 := minInt(len(mags)-1, int(f1/binHz))
		if i1 <= i0 {
			i1 = i0 + 1
		}
		sumSq := 0.0
		count := 0
		for i := i0; i < i1 && i < len(mags); i++ {
			sumSq += mags[i] * mags[i]
			count++
		}
		val := 0.0
		if count > 0 {
			val = math.Sqrt(sumSq / float64(count))
		}

		// Treble tilt compensation (equal-loudness / pink noise balance)
		centerHz := math.Sqrt(f0 * f1)
		tilt := math.Pow(centerHz/400.0, 0.22)
		val *= tilt

		db := 20 * math.Log10(val+1e-6)
		// Dynamic range window (-42..-2 dBFS) so bars bounce expressively
		// across all 0..6 levels rather than pegging at the maximum.
		const minDB = -42.0
		const maxDB = -2.0
		level := (db - minDB) * 6.0 / (maxDB - minDB)
		if level < 0 {
			level = 0
		}
		if level > 6 {
			level = 6
		}
		bars[b] = int(math.Round(level))
	}
	return bars
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
