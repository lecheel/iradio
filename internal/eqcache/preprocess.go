package eqcache

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"time"
)

const (
	// SampleRate is the mono analysis sample rate.
	SampleRate = 22050
	// ChunkSize is the FFT window / hop size in samples.
	ChunkSize = 1024
	// NumBars is the number of log-spaced spectrum columns.
	NumBars = 32
)

// Compute reads the file via ffmpeg and returns the full cached EQ series.
// The hop is one ChunkSize, i.e. 1024/22050 ≈ 46.4 ms per frame.
//
// The FFT and windowing are provided by spectrumAnalyzer (see fft.go),
// which reuses all scratch buffers across frames so the hot loop performs
// zero allocations.
func Compute(path string) (*CachedEQ, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-v", "quiet",
		"-nostdin",
		"-i", path,
		"-f", "f32le",
		"-ac", "1",
		"-ar", strconv.Itoa(SampleRate),
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// ChunkSize and SampleRate are untyped constants, so the division
	// result must be stored in a variable before truncating to int.
	hopMsF := float64(ChunkSize) * 1000.0 / float64(SampleRate)
	hopMs := int(hopMsF)
	if hopMs <= 0 {
		hopMs = 1
	}

	analyzer := newSpectrumAnalyzer(ChunkSize, NumBars)

	var bars []byte
	buf := make([]byte, ChunkSize*4)
	samples := make([]float64, ChunkSize)
	frames := 0

	for {
		if _, err := io.ReadFull(stdout, buf); err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				break
			}
			_ = cmd.Wait()
			return nil, err
		}
		for i := 0; i < ChunkSize; i++ {
			bits := binary.LittleEndian.Uint32(buf[i*4:])
			samples[i] = float64(math.Float32frombits(bits))
		}
		bar := analyzer.compute(samples, SampleRate)
		for _, v := range bar {
			bars = append(bars, byte(v))
		}
		frames++
	}
	_ = cmd.Wait()

	if frames == 0 {
		return nil, fmt.Errorf("no audio frames decoded from %s", path)
	}

	return &CachedEQ{
		DurationMs: frames * hopMs,
		HopMs:      hopMs,
		NumBars:    NumBars,
		Bars:       bars,
	}, nil
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
