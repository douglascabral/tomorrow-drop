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

// centredTitleLayout lays out a back button and a title: the button at the left
// edge and the title centred in the whole width, not in what the button leaves.
// The title is given the width left when the room of the button is kept on both
// sides, so it never runs under the button; it centres itself in that space.
type centredTitleLayout struct{}

func (centredTitleLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	back, title := objs[0], objs[1]
	b := back.MinSize()
	back.Move(fyne.NewPos(0, (size.Height-b.Height)/2))
	back.Resize(b)
	avail := size.Width - 2*b.Width
	if avail < 0 {
		avail = 0
	}
	title.Move(fyne.NewPos(b.Width, 0))
	title.Resize(fyne.NewSize(avail, size.Height))
}

func (centredTitleLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	b, t := objs[0].MinSize(), objs[1].MinSize()
	// The title may wrap, so only the room of the button is asked of it.
	h := b.Height
	if t.Height > h {
		h = t.Height
	}
	return fyne.NewSize(2*b.Width+t.Width, h)
}
