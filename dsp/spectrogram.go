package dsp

import (
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"
)

// SpectrogramResult holds the computed spectrogram data
type SpectrogramResult struct {
	Data       [][]float64 // [frequency][time] - log10 magnitude values
	Freqs      []float64   // Frequency values for each row (Hz)
	Times      []float64   // Time values for each column (seconds)
	FreqMin    float64     // Actual minimum frequency used
	FreqMax    float64     // Actual maximum frequency used
	SampleRate int
}

// ComputeSpectrogram computes the spectrogram with logarithmic frequency axis
func ComputeSpectrogram(samples []float64, sampleRate int, freqMin, freqMax, freqRes int, spectrogramWidth, spectrogramHeight int) *SpectrogramResult {
	numSamples := len(samples)

	// Number of points per segment for desired frequency resolution
	// If freqRes = 1 Hz, then npts = sampleRate
	npts := sampleRate / freqRes
	if npts > numSamples {
		npts = numSamples
	}

	// Calculate overlap to achieve desired time resolution
	// num_segments = (num_samples - noverlap) / (npts - noverlap)
	// Solving for noverlap: noverlap = (spectrogramWidth * npts - numSamples) / (spectrogramWidth - 1)
	noverlap := int(math.Ceil(float64(spectrogramWidth*npts-numSamples) / float64(spectrogramWidth-1)))
	if noverlap < 0 {
		noverlap = 0
	}
	if noverlap >= npts {
		noverlap = npts - 1
	}

	// Compute the STFT
	freqs, times, magnitude := stft(samples, sampleRate, npts, noverlap)

	// Resample to logarithmic frequency axis
	targetFreqs := logspace(math.Log10(float64(freqMin)), math.Log10(float64(freqMax)), spectrogramHeight)
	logMagnitude, actualFreqs := resampleToLogFreq(magnitude, freqs, targetFreqs)

	// Apply log10 to magnitude values
	for i := range logMagnitude {
		for j := range logMagnitude[i] {
			val := logMagnitude[i][j]
			if val <= 0 {
				val = 1e-10 // Avoid log(0)
			}
			logMagnitude[i][j] = math.Log10(val)
		}
	}

	return &SpectrogramResult{
		Data:       logMagnitude,
		Freqs:      actualFreqs,
		Times:      times,
		FreqMin:    float64(freqMin),
		FreqMax:    float64(freqMax),
		SampleRate: sampleRate,
	}
}

// stft computes the Short-Time Fourier Transform
func stft(samples []float64, sampleRate, npts, noverlap int) (freqs, times []float64, magnitude [][]float64) {
	step := npts - noverlap
	numSegments := (len(samples) - noverlap) / step

	if numSegments <= 0 {
		numSegments = 1
	}

	// FFT setup
	fft := fourier.NewFFT(npts)
	numFreqs := npts/2 + 1

	// Create Hann window
	window := hannWindow(npts)

	// Initialize output
	magnitude = make([][]float64, numFreqs)
	for i := range magnitude {
		magnitude[i] = make([]float64, numSegments)
	}

	times = make([]float64, numSegments)
	freqs = make([]float64, numFreqs)

	// Compute frequency values
	for i := 0; i < numFreqs; i++ {
		freqs[i] = float64(i) * float64(sampleRate) / float64(npts)
	}

	// Process each segment
	segment := make([]float64, npts)
	for seg := 0; seg < numSegments; seg++ {
		start := seg * step
		end := start + npts

		// Handle edge case where we don't have enough samples
		if end > len(samples) {
			// Zero-pad the last segment
			for i := range segment {
				if start+i < len(samples) {
					segment[i] = samples[start+i] * window[i]
				} else {
					segment[i] = 0
				}
			}
		} else {
			// Apply window
			for i := 0; i < npts; i++ {
				segment[i] = samples[start+i] * window[i]
			}
		}

		// Compute FFT
		coeffs := fft.Coefficients(nil, segment)

		// Extract magnitude (only positive frequencies)
		for i := 0; i < numFreqs; i++ {
			magnitude[i][seg] = cmplx.Abs(coeffs[i])
		}

		// Time at center of segment
		times[seg] = float64(start+npts/2) / float64(sampleRate)
	}

	return freqs, times, magnitude
}

// hannWindow creates a Hann window of given length
func hannWindow(n int) []float64 {
	window := make([]float64, n)
	for i := 0; i < n; i++ {
		window[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return window
}

// logspace generates n logarithmically spaced values from 10^start to 10^stop
func logspace(start, stop float64, n int) []float64 {
	result := make([]float64, n)
	step := (stop - start) / float64(n-1)
	for i := 0; i < n; i++ {
		result[i] = math.Pow(10, start+float64(i)*step)
	}
	return result
}

// resampleToLogFreq resamples the spectrogram to logarithmic frequency bins
func resampleToLogFreq(magnitude [][]float64, freqs []float64, targetFreqs []float64) ([][]float64, []float64) {
	numTargetFreqs := len(targetFreqs)
	numTimes := len(magnitude[0])

	result := make([][]float64, numTargetFreqs)
	actualFreqs := make([]float64, numTargetFreqs)

	for i, targetFreq := range targetFreqs {
		// Find the closest frequency bin
		idx := findClosestIndex(freqs, targetFreq)
		actualFreqs[i] = freqs[idx]

		result[i] = make([]float64, numTimes)
		copy(result[i], magnitude[idx])
	}

	return result, actualFreqs
}

// findClosestIndex finds the index of the closest value to target in a sorted slice
func findClosestIndex(slice []float64, target float64) int {
	if len(slice) == 0 {
		return 0
	}

	// Binary search for insertion point
	left, right := 0, len(slice)-1

	for left < right {
		mid := (left + right) / 2
		if slice[mid] < target {
			left = mid + 1
		} else {
			right = mid
		}
	}

	// Check if previous index is closer
	if left > 0 {
		if math.Abs(slice[left-1]-target) < math.Abs(slice[left]-target) {
			return left - 1
		}
	}

	return left
}
