package eqcache

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"math/cmplx"
	"os/exec"
	"strconv"
	"time"
)

const (
	SampleRate = 22050
	ChunkSize  = 1024
	NumBars    = 32
)

// Compute reads the file via ffmpeg and returns the full cached EQ series.
// The hop is one ChunkSize, i.e. 1024/22050 ≈ 46.4 ms per frame.
func Compute(path string) (*CachedEQ, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-v", "quiet",
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
		bar := computeSpectrumBars(samples, SampleRate, NumBars)
		for _, v := range bar {
			bars = append(bars, byte(v))
		}
		frames++
	}
	_ = cmd.Wait()

	return &CachedEQ{
		DurationMs: frames * hopMs,
		HopMs:      hopMs,
		NumBars:    NumBars,
		Bars:       bars,
	}, nil
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

func computeSpectrumBars(samples []float64, sampleRate int, numBars int) []int {
	n := len(samples)
	data := make([]complex128, n)
	for i, s := range samples {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		data[i] = complex(s*w, 0)
	}
	fft(data)

	mags := make([]float64, n/2)
	for i := range mags {
		mags[i] = cmplx.Abs(data[i])
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
		peak := 0.0
		for i := i0; i < i1 && i < len(mags); i++ {
			if mags[i] > peak {
				peak = mags[i]
			}
		}
		db := 20 * math.Log10(peak+1e-6)
		level := (db + 60) / 10
		if level < 0 {
			level = 0
		}
		if level > 6 {
			level = 6
		}
		bars[b] = int(level)
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
