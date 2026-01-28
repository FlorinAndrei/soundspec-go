package render

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"git.sr.ht/~sbinet/gg"
	"github.com/FlorinAndrei/soundspec-go/config"
	"github.com/FlorinAndrei/soundspec-go/dsp"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

// Font sizes
const (
	fontSizeTitle = 16.0
	fontSizeLabel = 14.0
	fontSizeTick  = 12.0
)

var defaultFont *truetype.Font

func init() {
	var err error
	defaultFont, err = truetype.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}
}

func getFontFace(size float64) font.Face {
	return truetype.NewFace(defaultFont, &truetype.Options{
		Size: size,
		DPI:  72,
	})
}

// RenderSpectrogram creates a PNG image of the spectrogram with axes
func RenderSpectrogram(result *dsp.SpectrogramResult, title string, outputPath string, cfg config.Config) error {
	// Spectrogram area dimensions (from config)
	spectWidth := cfg.SpectrogramSize
	spectHeight := cfg.SpectrogramHeight()

	// Total image dimensions (spectrogram + margins)
	totalWidth := cfg.TotalImageWidth()
	totalHeight := cfg.TotalImageHeight()

	// Create the drawing context
	dc := gg.NewContext(totalWidth, totalHeight)

	// Fill background with white
	dc.SetColor(color.White)
	dc.Clear()

	// Create the spectrogram as an image and draw it
	spectImg := createSpectrogramImage(result, spectWidth, spectHeight)
	dc.DrawImage(spectImg, config.MarginLeft, config.MarginTop)

	// Draw border around spectrogram
	dc.SetColor(color.Black)
	dc.SetLineWidth(1)
	dc.DrawRectangle(float64(config.MarginLeft), float64(config.MarginTop), float64(spectWidth), float64(spectHeight))
	dc.Stroke()

	// Draw axes
	drawAxes(dc, result, cfg)

	// Draw title
	drawTitle(dc, title, totalWidth)

	// Draw grid lines
	drawGridLines(dc, result, cfg)

	// Save to file
	return dc.SavePNG(outputPath)
}

// createSpectrogramImage creates a raster image from the spectrogram data
func createSpectrogramImage(result *dsp.SpectrogramResult, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Find min/max values for color scaling
	minVal, maxVal := findMinMax(result.Data)
	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}

	numFreqs := len(result.Data)    // Number of frequency bins
	numTimes := len(result.Data[0]) // Number of time bins

	// For each pixel in the spectrogram area
	for py := 0; py < height; py++ {
		// Map pixel y to frequency bin (linear mapping since data is already log-spaced)
		// py=0 is top = high frequency = last index
		// py=height-1 is bottom = low frequency = first index
		freqIdx := int(float64(numFreqs-1) * float64(height-1-py) / float64(height-1))
		if freqIdx < 0 {
			freqIdx = 0
		}
		if freqIdx >= numFreqs {
			freqIdx = numFreqs - 1
		}

		for px := 0; px < width; px++ {
			// Map pixel x to time bin
			timeIdx := int(float64(numTimes-1) * float64(px) / float64(width-1))
			if timeIdx < 0 {
				timeIdx = 0
			}
			if timeIdx >= numTimes {
				timeIdx = numTimes - 1
			}

			// Get the value and map to color
			val := result.Data[freqIdx][timeIdx]
			normalized := (val - minVal) / valRange
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}

			col := Viridis(normalized)
			img.Set(px, py, col)
		}
	}

	return img
}

// drawAxes draws the axis labels and tick marks
func drawAxes(dc *gg.Context, result *dsp.SpectrogramResult, cfg config.Config) {
	spectWidth := cfg.SpectrogramSize
	spectHeight := cfg.SpectrogramHeight()

	dc.SetColor(color.Black)

	// Y-axis ticks (frequency, log scale)
	freqTicks := generateFrequencyTicks(result.FreqMin, result.FreqMax)
	for _, tick := range freqTicks {
		// Calculate y position (log scale)
		logMin := math.Log10(result.FreqMin)
		logMax := math.Log10(result.FreqMax)
		logVal := math.Log10(tick.value)
		t := (logVal - logMin) / (logMax - logMin)
		y := float64(config.MarginTop) + float64(spectHeight-1)*(1-t)

		// Draw tick mark
		dc.SetLineWidth(1)
		dc.DrawLine(float64(config.MarginLeft-5), y, float64(config.MarginLeft), y)
		dc.Stroke()

		// Draw label (right-aligned)
		dc.SetFontFace(getFontFace(fontSizeTick))
		labelWidth, _ := dc.MeasureString(tick.label)
		dc.DrawString(tick.label, float64(config.MarginLeft-8)-labelWidth, y+4)
	}

	// X-axis ticks (time, linear scale)
	xMin := 0.0
	xMax := 1.0
	if len(result.Times) > 0 {
		xMin = result.Times[0]
		xMax = result.Times[len(result.Times)-1]
	}

	timeTicks := generateTimeTicks(xMin, xMax)
	for _, tick := range timeTicks {
		// Calculate x position
		t := (tick.value - xMin) / (xMax - xMin)
		x := float64(config.MarginLeft) + float64(spectWidth-1)*t

		// Draw tick mark
		dc.SetLineWidth(1)
		dc.DrawLine(x, float64(config.MarginTop+spectHeight), x, float64(config.MarginTop+spectHeight+5))
		dc.Stroke()

		// Draw label (centered)
		dc.SetFontFace(getFontFace(fontSizeTick))
		labelWidth, _ := dc.MeasureString(tick.label)
		dc.DrawString(tick.label, x-labelWidth/2, float64(config.MarginTop+spectHeight+20))
	}

	// Y-axis label (rotated 90 degrees)
	dc.SetFontFace(getFontFace(fontSizeLabel))
	dc.Push()
	labelText := "Frequency [Hz]"
	labelWidth, labelHeight := dc.MeasureString(labelText)
	// Position for vertical text on left side
	labelX := 15.0
	labelY := float64(config.MarginTop + spectHeight/2)
	dc.RotateAbout(-math.Pi/2, labelX, labelY)
	dc.DrawString(labelText, labelX-labelWidth/2, labelY+labelHeight/3)
	dc.Pop()

	// X-axis label
	dc.SetFontFace(getFontFace(fontSizeLabel))
	labelText = "Time [s]"
	labelWidth, _ = dc.MeasureString(labelText)
	dc.DrawString(labelText, float64(config.MarginLeft+spectWidth/2)-labelWidth/2, float64(config.MarginTop+spectHeight+42))
}

// drawTitle draws the title at the top
func drawTitle(dc *gg.Context, title string, totalWidth int) {
	dc.SetColor(color.Black)
	dc.SetFontFace(getFontFace(fontSizeTitle))
	labelWidth, _ := dc.MeasureString(title)
	dc.DrawString(title, float64(totalWidth)/2-labelWidth/2, float64(config.MarginTop-10))
}

// drawGridLines draws grid lines on the spectrogram
func drawGridLines(dc *gg.Context, result *dsp.SpectrogramResult, cfg config.Config) {
	spectWidth := cfg.SpectrogramSize
	spectHeight := cfg.SpectrogramHeight()

	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.SetLineWidth(0.5)

	// Horizontal grid lines (frequency)
	freqTicks := generateFrequencyTicks(result.FreqMin, result.FreqMax)
	for _, tick := range freqTicks {
		logMin := math.Log10(result.FreqMin)
		logMax := math.Log10(result.FreqMax)
		logVal := math.Log10(tick.value)
		t := (logVal - logMin) / (logMax - logMin)
		y := float64(config.MarginTop) + float64(spectHeight-1)*(1-t)

		dc.DrawLine(float64(config.MarginLeft), y, float64(config.MarginLeft+spectWidth), y)
		dc.Stroke()
	}

	// Vertical grid lines (time)
	xMin := 0.0
	xMax := 1.0
	if len(result.Times) > 0 {
		xMin = result.Times[0]
		xMax = result.Times[len(result.Times)-1]
	}

	timeTicks := generateTimeTicks(xMin, xMax)
	for _, tick := range timeTicks {
		t := (tick.value - xMin) / (xMax - xMin)
		x := float64(config.MarginLeft) + float64(spectWidth-1)*t

		dc.DrawLine(x, float64(config.MarginTop), x, float64(config.MarginTop+spectHeight))
		dc.Stroke()
	}

}

type tick struct {
	value float64
	label string
}

// generateFrequencyTicks generates tick marks for a log frequency axis
// Ticks at 1-9 for each decade, but only powers of 10 are labeled
func generateFrequencyTicks(freqMin, freqMax float64) []tick {
	var ticks []tick

	startPow := math.Floor(math.Log10(freqMin))
	endPow := math.Ceil(math.Log10(freqMax))

	for pow := startPow; pow <= endPow; pow++ {
		base := math.Pow(10, pow)

		// Generate ticks at 1, 2, 3, ..., 9 times the base
		for mult := 1; mult <= 9; mult++ {
			val := base * float64(mult)
			if val >= freqMin && val <= freqMax {
				// Only label powers of 10 (mult == 1)
				label := ""
				if mult == 1 {
					label = formatFrequency(val)
				}
				ticks = append(ticks, tick{value: val, label: label})
			}
		}
	}

	return ticks
}

// generateTimeTicks generates tick marks for a linear time axis
func generateTimeTicks(tMin, tMax float64) []tick {
	var ticks []tick

	duration := tMax - tMin
	if duration <= 0 {
		return ticks
	}

	// Choose a nice interval
	interval := chooseNiceInterval(duration, 8)

	// Start from a nice round number
	start := math.Ceil(tMin/interval) * interval

	for t := start; t <= tMax; t += interval {
		label := fmt.Sprintf("%.1f", t)
		if interval >= 1 {
			label = fmt.Sprintf("%.0f", t)
		}
		ticks = append(ticks, tick{value: t, label: label})
	}

	return ticks
}

// chooseNiceInterval chooses a nice interval for the axis
func chooseNiceInterval(range_ float64, targetTicks int) float64 {
	rough := range_ / float64(targetTicks)
	magnitude := math.Pow(10, math.Floor(math.Log10(rough)))
	normalized := rough / magnitude

	var nice float64
	if normalized < 1.5 {
		nice = 1
	} else if normalized < 3 {
		nice = 2
	} else if normalized < 7 {
		nice = 5
	} else {
		nice = 10
	}

	return nice * magnitude
}

// formatFrequency formats a frequency value for display
func formatFrequency(freq float64) string {
	if freq >= 1000 {
		return fmt.Sprintf("%.0fk", freq/1000)
	}
	if freq >= 100 {
		return fmt.Sprintf("%.0f", freq)
	}
	if freq >= 10 {
		return fmt.Sprintf("%.0f", freq)
	}
	return fmt.Sprintf("%.1f", freq)
}

// findMinMax finds the minimum and maximum values in a 2D slice
func findMinMax(data [][]float64) (min, max float64) {
	if len(data) == 0 || len(data[0]) == 0 {
		return 0, 1
	}

	min = data[0][0]
	max = data[0][0]

	for _, row := range data {
		for _, val := range row {
			if !math.IsInf(val, 0) && !math.IsNaN(val) {
				if val < min {
					min = val
				}
				if val > max {
					max = val
				}
			}
		}
	}

	return min, max
}
