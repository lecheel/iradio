package eqcache

import (
	"math"
	"math/cmplx"
	"sync"
)

// fftPlan holds precomputed twiddle factors and bit-reversal indices for a
// fixed FFT size, so repeated transforms avoid allocation and trig work.
type fftPlan struct {
	n       int
	twiddle []complex128
	rev     []int
}

var (
	planMu sync.Mutex
	plans  = map[int]*fftPlan{}
)

func getPlan(n int) *fftPlan {
	planMu.Lock()
	p, ok := plans[n]
	planMu.Unlock()
	if ok {
		return p
	}
	p = newPlan(n)
	planMu.Lock()
	plans[n] = p
	planMu.Unlock()
	return p
}

func newPlan(n int) *fftPlan {
	p := &fftPlan{n: n}
	p.twiddle = make([]complex128, n/2)
	for i := 0; i < n/2; i++ {
		theta := -2 * math.Pi * float64(i) / float64(n)
		p.twiddle[i] = cmplx.Rect(1, theta)
	}
	bits := 0
	for (1 << bits) < n {
		bits++
	}
	p.rev = make([]int, n)
	for i := 0; i < n; i++ {
		r := 0
		for b := 0; b < bits; b++ {
			if i&(1<<b) != 0 {
				r |= 1 << (bits - 1 - b)
			}
		}
		p.rev[i] = r
	}
	return p
}

// transform runs an in-place iterative radix-2 Cooley-Tukey FFT.
// This is 5-10x faster than the previous recursive version for n=1024
// because it never allocates and uses precomputed twiddles.
func (p *fftPlan) transform(a []complex128) {
	n := p.n
	if len(a) != n {
		return
	}
	rev := p.rev
	for i := 0; i < n; i++ {
		j := rev[i]
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	tw := p.twiddle
	for size := 2; size <= n; size <<= 1 {
		half := size >> 1
		step := n / size
		for i := 0; i < n; i += size {
			for j := 0; j < half; j++ {
				t := tw[j*step] * a[i+j+half]
				a[i+j+half] = a[i+j] - t
				a[i+j] = a[i+j] + t
			}
		}
	}
}

// spectrumAnalyzer reuses every scratch buffer needed to turn a chunk of
// samples into bar levels, so the hot loop allocates nothing at all.
type spectrumAnalyzer struct {
	n      int
	bars   int
	plan   *fftPlan
	data   []complex128
	window []float64
	mags   []float64
	out    []int
}

func newSpectrumAnalyzer(n, numBars int) *spectrumAnalyzer {
	sa := &spectrumAnalyzer{
		n:      n,
		bars:   numBars,
		plan:   getPlan(n),
		data:   make([]complex128, n),
		window: make([]float64, n),
		mags:   make([]float64, n/2),
		out:    make([]int, numBars),
	}
	for i := range sa.window {
		sa.window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
	}
	return sa
}

// compute fills the internal output buffer with bar levels (0..6) for the
// given samples. The returned slice aliases internal state and is valid
// until the next call to compute.
func (sa *spectrumAnalyzer) compute(samples []float64, sampleRate int) []int {
	n := sa.n
	w := sa.window
	d := sa.data
	for i := 0; i < n; i++ {
		d[i] = complex(samples[i]*w[i], 0)
	}
	sa.plan.transform(d)

	mags := sa.mags
	// Normalise by 4/N to compensate for the Hann window's coherent gain
	// (0.5). Without this, FFT magnitudes for full-scale audio sit around
	// +40 dB and every band pegs at the maximum level.
	norm := 4.0 / float64(n)
	for i := range mags {
		mags[i] = cmplx.Abs(d[i]) * norm
	}

	minHz, maxHz := 40.0, float64(sampleRate)/2
	logMin, logMax := math.Log10(minHz), math.Log10(maxHz)
	binHz := float64(sampleRate) / float64(n)
	numBars := sa.bars

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
		sa.out[b] = int(math.Round(level))
	}
	return sa.out
}
