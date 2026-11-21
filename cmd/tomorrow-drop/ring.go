package main

import (
	"image"
	"image/color"
	"math"
)

// ringImage draws a circular progress ring: the accent color fills clockwise
// from the top in proportion to fraction, the rest is the track color.
func ringImage(fraction float64, size int, accent, track color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := float64(size) / 2
	outer := c - 1
	inner := outer * 0.82
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - c
			dy := float64(y) + 0.5 - c
			r := math.Hypot(dx, dy)
			if r > outer || r < inner {
				continue
			}
			angle := math.Atan2(dx, -dy) // 0 at the top, increasing clockwise
			if angle < 0 {
				angle += 2 * math.Pi
			}
			if angle/(2*math.Pi) <= fraction {
				img.Set(x, y, accent)
			} else {
				img.Set(x, y, track)
			}
		}
	}
	return img
}
