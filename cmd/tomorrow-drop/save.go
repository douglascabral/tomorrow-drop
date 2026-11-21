package main

import (
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/transfer"
)

// receiveDestination picks where received files are saved. Phones ask the
// platform through ask; desktops return nil so the receiver uses Downloads.
func receiveDestination(mobile bool, ask func(name string) (io.WriteCloser, error)) transfer.Destination {
	if !mobile {
		return nil
	}
	return transfer.WriterDestination{Open: ask}
}

// saveLocation asks the platform where to save one received file: the storage
// picker on Android and the Files export on iOS. The receiver calls it from its
// own goroutine and waits for the answer, so the question itself runs on the UI
// thread through fyne.Do.
func (g *gui) saveLocation(name string) (io.WriteCloser, error) {
	type answer struct {
		w   io.WriteCloser
		err error
	}
	asked := make(chan answer, 1)
	// The picker takes the app to the background; the reception must go on.
	g.pickingFile.Store(true)
	fyne.Do(func() {
		d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
			switch {
			case err != nil:
				asked <- answer{err: err}
			case w == nil:
				asked <- answer{err: apperr.ErrCanceled}
			default:
				asked <- answer{w: w}
			}
		}, g.win)
		d.SetFileName(name)
		d.Show()
	})

	select {
	case a := <-asked:
		g.pickingFile.Store(false)
		return a.w, a.err
	case <-g.ctx.Done():
		g.pickingFile.Store(false)
		// A writer may still arrive after the app stopped. Close it so the
		// platform does not keep a half-written file open.
		go func() {
			if a := <-asked; a.w != nil {
				_ = a.w.Close()
			}
		}()
		return nil, apperr.ErrCanceled
	}
}
