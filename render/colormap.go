package render

import (
	"image/color"

	"github.com/dim13/colormap"
)

// Viridis returns the viridis color for a value between 0 and 1
func Viridis(t float64) color.RGBA {
	// Clamp t to [0, 1]
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// colormap.Viridis is a color.Palette ([]color.Color) with 256 entries
	idx := int(t * float64(len(colormap.Viridis)-1))
	if idx >= len(colormap.Viridis) {
		idx = len(colormap.Viridis) - 1
	}

	c := colormap.Viridis[idx]
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}
