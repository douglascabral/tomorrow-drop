package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

// The desktop window opens at 80% of the free screen width and 95% of its
// height, so the lists have room. Phones fill the screen and keep their own size.
func TestDesktopWindowOpensAt80PercentWidthAnd95PercentHeight(t *testing.T) {
	got := initialWindowSize(false, fyne.NewSize(1600, 900), true)
	if want := fyne.NewSize(1280, 855); got != want {
		t.Fatalf("initialWindowSize = %v, want %v", got, want)
	}
}

func TestWindowKeepsTheDefaultSizeWhenTheScreenIsUnknown(t *testing.T) {
	got := initialWindowSize(false, fyne.Size{}, false)
	if got != defaultWindowSize {
		t.Fatalf("initialWindowSize = %v, want %v", got, defaultWindowSize)
	}
}

func TestSmallScreenNeverShrinksTheWindowBelowTheDefault(t *testing.T) {
	got := initialWindowSize(false, fyne.NewSize(480, 700), true)
	if got.Width < defaultWindowSize.Width || got.Height < defaultWindowSize.Height {
		t.Fatalf("initialWindowSize = %v, below the default %v", got, defaultWindowSize)
	}
}

func TestPhoneWindowIgnoresTheScreenArea(t *testing.T) {
	got := initialWindowSize(true, fyne.NewSize(1600, 900), true)
	if got != defaultWindowSize {
		t.Fatalf("initialWindowSize = %v, want %v", got, defaultWindowSize)
	}
}
