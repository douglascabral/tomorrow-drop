package main

import (
	"image/color"
	"math"
	"testing"
)

// The progress ring fills clockwise from the top in proportion to the fraction.
func TestRingFillsProportionally(t *testing.T) {
	accent := color.RGBA{R: 255, A: 255}
	track := color.RGBA{B: 255, A: 255}
	for _, fraction := range []float64{0, 0.25, 0.72, 1} {
		img := ringImage(fraction, 200, accent, track)
		var filled, total int
		for y := 0; y < 200; y++ {
			for x := 0; x < 200; x++ {
				c := img.At(x, y)
				r, _, b, a := c.RGBA()
				if a == 0 {
					continue
				}
				total++
				if r > b {
					filled++
				}
			}
		}
		if total == 0 {
			t.Fatalf("ring for %.2f drew nothing", fraction)
		}
		got := float64(filled) / float64(total)
		if math.Abs(got-fraction) > 0.03 {
			t.Errorf("ring for %.2f filled %.3f of its area", fraction, got)
		}
	}
}
