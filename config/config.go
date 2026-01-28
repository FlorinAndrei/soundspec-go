package config

import "runtime"

// AspectRatio is the height/width ratio for the spectrogram (16:9)
const AspectRatio = 9.0 / 16.0

// Plot margins in pixels (added to the spectrogram area)
const (
	MarginLeft   = 80
	MarginRight  = 20
	MarginTop    = 40
	MarginBottom = 50
)

// Config holds all configuration parameters for spectrogram generation
type Config struct {
	// Input/Output
	Input     string
	Output    string
	Overwrite bool

	// Image - SpectrogramSize is the width of the spectrogram AREA (not including margins)
	SpectrogramSize int

	// Frequency
	FrequencyMin int
	FrequencyMax int
	FrequencyRes int

	// Processing
	Workers int

	// Other
	Quiet bool
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		Output:          ".",
		Overwrite:       false,
		SpectrogramSize: 1000,
		FrequencyMin:    10,
		FrequencyMax:    20000,
		FrequencyRes:    1,
		Workers:         runtime.NumCPU(),
		Quiet:           false,
	}
}

// SpectrogramHeight calculates the spectrogram AREA height based on width and aspect ratio
func (c Config) SpectrogramHeight() int {
	return int(float64(c.SpectrogramSize) * AspectRatio)
}

// TotalImageWidth returns the total image width (spectrogram + margins)
func (c Config) TotalImageWidth() int {
	return c.SpectrogramSize + MarginLeft + MarginRight
}

// TotalImageHeight returns the total image height (spectrogram + margins)
func (c Config) TotalImageHeight() int {
	return c.SpectrogramHeight() + MarginTop + MarginBottom
}
