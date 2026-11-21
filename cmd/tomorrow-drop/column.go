package main

import (
	"fyne.io/fyne/v2"
)

// maxColumnWidth is the widest content of a desktop screen. Wider windows keep
// the content in the middle instead of stretching every control.
const maxColumnWidth = 960

// maxWidthLayout centers its single child in a column of at most max width. On
// a narrow window the child takes the whole width.
type maxWidthLayout struct {
	max float32
}

func (l *maxWidthLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	w := size.Width
	if w > l.max {
		w = l.max
	}
	x := (size.Width - w) / 2
	for _, o := range objs {
		o.Move(fyne.NewPos(x, 0))
		o.Resize(fyne.NewSize(w, size.Height))
	}
}

func (l *maxWidthLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var min fyne.Size
	for _, o := range objs {
		s := o.MinSize()
		if s.Width > min.Width {
			min.Width = s.Width
		}
		if s.Height > min.Height {
			min.Height = s.Height
		}
	}
	return min
}
