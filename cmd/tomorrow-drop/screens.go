package main

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/settings"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// header is the logo, name and slogan shown at the top of every screen.
func (g *gui) header(title string) fyne.CanvasObject {
	back := widget.NewButton(g.tr.T("gui.back"), g.showHome)
	name := widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	if title == "" {
		name = widget.NewLabelWithStyle(g.tr.T("app.nome"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	}
	return container.NewBorder(nil, nil, back, nil, container.NewVBox(container.NewCenter(logoImage(56)), name))
}

func (g *gui) show(content fyne.CanvasObject) {
	g.onUI(func() { g.win.SetContent(content) })
}

// showHome is the first screen (PRD §19.3): send, receive status, discovered
// devices, refresh, manual connection and settings.
func (g *gui) showHome() {
	g.refreshDevices()
	status := widget.NewLabel(g.tr.T("gui.home.receiving", g.tlsPort))

	sendBtn := widget.NewButton(g.tr.T("gui.home.send"), func() { g.showSend("", nil) })
	trustedBtn := widget.NewButton(g.tr.T("gui.home.trusted"), g.showTrusted)
	settingsBtn := widget.NewButton(g.tr.T("gui.home.settings"), g.showSettings)

	refreshBtn := widget.NewButton(g.tr.T("gui.home.refresh"), func() {
		if err := g.ctl.Refresh(); err != nil {
			g.notify(g.tr.T("gui.home.refresh"), g.tr.T(apperr.MessageKey(err)))
		}
	})

	manual := widget.NewEntry()
	manual.SetPlaceHolder(g.tr.T("gui.home.manual_placeholder"))
	connect := widget.NewButton(g.tr.T("gui.home.connect"), func() {
		ep, err := manualEndpoint(manual.Text)
		if err != nil {
			g.notify(g.tr.T("gui.home.manual"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.showSend(ep.String(), nil)
	})

	list := widget.NewList(
		func() int { return len(g.rows) },
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel(""), widget.NewLabel(""))
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			row := g.rows[i]
			labels := obj.(*fyne.Container).Objects
			labels[0].(*widget.Label).SetText(row.Name + "  ·  " + g.tr.T(row.StatusKey))
			labels[1].(*widget.Label).SetText(row.Address)
		},
	)
	list.OnSelected = func(i widget.ListItemID) {
		list.Unselect(i)
		g.showDevice(g.rows[i])
	}
	g.devicesList = list

	top := container.NewVBox(
		g.header(""),
		status,
		container.NewGridWithColumns(3, sendBtn, trustedBtn, settingsBtn),
		container.NewBorder(nil, nil, nil, refreshBtn, widget.NewLabelWithStyle(g.tr.T("gui.home.devices"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})),
	)
	bottom := container.NewVBox(widget.NewLabel(g.tr.T("gui.home.manual")), container.NewBorder(nil, nil, nil, connect, manual))
	g.show(container.NewBorder(top, bottom, nil, nil, list))
}

// refreshDevices rebuilds the device list from the ephemeral registry (task 022).
func (g *gui) refreshDevices() {
	g.rows = ui.DeviceRows(g.reg.List(time.Now()))
	g.onUI(func() {
		if g.devicesList != nil {
			g.devicesList.Refresh()
		}
	})
}

// showDevice offers the actions for one discovered device.
func (g *gui) showDevice(row ui.DeviceRow) {
	send := widget.NewButton(g.tr.T("gui.device.send"), func() { g.showSend(row.Address, nil) })
	pair := widget.NewButton(g.tr.T("gui.device.pair"), func() { g.pairWithAddress(row.Address) })
	body := container.NewVBox(widget.NewLabel(row.Name), widget.NewLabel(row.Address), widget.NewLabel(g.tr.T(row.StatusKey)), send, pair)
	g.onUI(func() {
		dialog.ShowCustom(g.tr.T("gui.device.title"), g.tr.T("gui.close"), body, g.win)
	})
}

// pairWithAddress starts pairing with a device the person chose.
func (g *gui) pairWithAddress(addr string) {
	ep, err := manualEndpoint(addr)
	if err != nil {
		g.notify(g.tr.T("gui.pair.title"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	go func() {
		s, err := transport.Dial(g.ctx, ep, g.id, g.store)
		if err != nil {
			g.notify(g.tr.T("gui.pair.title"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		defer func() { _ = s.Close() }()
		if s.State() == protocol.StateTrusted {
			g.notify(g.tr.T("gui.pair.title"), g.tr.T("gui.pair.already"))
			return
		}
		g.pairAsked(g.ctx, s)
	}()
}

// showSend is the send screen (PRD §19.3): choose files, write text, or send
// the clipboard by an explicit action (tasks 049 and 050).
func (g *gui) showSend(target string, preset *transfer.SendSpec) {
	to := widget.NewEntry()
	to.SetPlaceHolder(g.tr.T("gui.send.target_placeholder"))
	to.SetText(target)

	var files []string
	fileList := widget.NewLabel(g.tr.T("gui.send.no_files"))
	addFile := widget.NewButton(g.tr.T("gui.send.add_file"), func() {
		g.onUI(func() {
			dialog.ShowFileOpen(func(f fyne.URIReadCloser, err error) {
				if err != nil || f == nil {
					return
				}
				_ = f.Close()
				files = append(files, f.URI().Path())
				fileList.SetText(strings.Join(files, "\n"))
			}, g.win)
		})
	})

	text := widget.NewMultiLineEntry()
	text.SetPlaceHolder(g.tr.T("gui.send.text_placeholder"))

	progress := widget.NewProgressBar()
	progress.Hide()
	status := widget.NewLabel("")

	var cancel context.CancelFunc
	cancelBtn := widget.NewButton(g.tr.T("gui.send.cancel"), func() {
		if cancel != nil {
			cancel()
		}
	})
	cancelBtn.Hide()

	run := func(spec transfer.SendSpec) {
		dest := strings.TrimSpace(to.Text)
		if dest == "" {
			status.SetText(g.tr.T("gui.send.need_target"))
			return
		}
		ctx, c := context.WithCancel(g.ctx)
		cancel = c
		progress.Show()
		cancelBtn.Show()
		status.SetText(g.tr.T("gui.send.working"))
		go func() {
			defer c()
			err := g.sendTo(ctx, dest, spec, func(p transfer.Progress) {
				g.onUI(func() { progress.SetValue(ui.ProgressFraction(p)) })
			})
			g.onUI(func() {
				progress.Hide()
				cancelBtn.Hide()
			})
			if err != nil {
				g.onUI(func() { status.SetText(g.tr.T(apperr.MessageKey(err))) })
				return
			}
			g.onUI(func() { status.SetText(g.tr.T("gui.send.sent")) })
		}()
	}

	sendFiles := widget.NewButton(g.tr.T("gui.send.files"), func() {
		plan, err := ui.PlanFiles(files)
		if err != nil {
			status.SetText(g.tr.T(apperr.MessageKey(err)))
			return
		}
		paths := make([]string, 0, len(plan.Items))
		for _, it := range plan.Items {
			paths = append(paths, it.Path)
		}
		run(transfer.SendSpec{Kind: transfer.KindFile, Paths: paths})
	})
	sendText := widget.NewButton(g.tr.T("gui.send.text"), func() {
		if strings.TrimSpace(text.Text) == "" {
			status.SetText(g.tr.T("gui.send.nothing"))
			return
		}
		run(transfer.SendSpec{Kind: transfer.KindText, Text: text.Text})
	})
	clipboard := widget.NewButton(g.tr.T("gui.send.clipboard"), func() {
		g.previewClipboard(func(spec transfer.SendSpec) { run(spec) })
	})

	if preset != nil {
		run(*preset)
	}

	body := container.NewVBox(
		widget.NewLabel(g.tr.T("gui.send.target")), to,
		addFile, fileList, sendFiles,
		widget.NewSeparator(),
		text, sendText,
		widget.NewSeparator(),
		clipboard,
		progress, cancelBtn, status,
	)
	g.show(container.NewBorder(g.header(g.tr.T("gui.send.title")), nil, nil, nil, container.NewVScroll(body)))
}

// previewClipboard reads the clipboard only after the person chose to send it,
// and shows a preview before anything leaves the device (PRD §12.2).
func (g *gui) previewClipboard(send func(transfer.SendSpec)) {
	content := g.app.Clipboard().Content()
	p, err := ui.ClipboardPreview(content)
	if err != nil {
		g.notify(g.tr.T("gui.send.clipboard"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	kind := g.tr.T("cli.kind." + string(p.Kind))
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.send.clipboard_title", kind), g.tr.T("gui.send.send"), g.tr.T("gui.pair.cancel"),
			widget.NewLabel(p.Snippet), func(ok bool) {
				if ok {
					send(transfer.SendSpec{Kind: p.Kind, Text: p.Text})
				}
			}, g.win)
	})
}

// sendTo dials the destination and sends. A device that is not trusted is never
// sent to: the person is told to pair first (PRD §8.1).
func (g *gui) sendTo(ctx context.Context, dest string, spec transfer.SendSpec, progress func(transfer.Progress)) error {
	ep, err := g.endpointFor(dest)
	if err != nil {
		return err
	}
	g.ctl.SetSuspended(true)
	defer g.ctl.SetSuspended(false)
	s, err := transport.Dial(ctx, ep, g.id, g.store)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()
	if s.State() != protocol.StateTrusted {
		return fmt.Errorf("device not trusted: %w", apperr.ErrUnauthorized)
	}
	_, err = transfer.Send(ctx, s, spec, progress)
	return err
}

// endpointFor accepts IP:port, or the name of a discovered device.
func (g *gui) endpointFor(dest string) (netip.AddrPort, error) {
	if ep, err := manualEndpoint(dest); err == nil {
		return ep, nil
	}
	for _, d := range g.reg.List(time.Now()) {
		if strings.EqualFold(d.Name, dest) {
			return d.Addr, nil
		}
	}
	return netip.AddrPort{}, fmt.Errorf("no device named %s: %w", dest, apperr.ErrMalformed)
}

// showTrusted lists trusted devices with their full fingerprint (task 048).
func (g *gui) showTrusted() {
	peers, err := g.store.List()
	if err != nil {
		g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	rows := ui.TrustedRows(peers)
	items := make([]fyne.CanvasObject, 0, len(rows))
	for i := range rows {
		row := rows[i]
		inspect := widget.NewButton(g.tr.T("gui.trusted.inspect"), func() {
			g.inspectTrusted(peers, row.ID)
		})
		items = append(items, container.NewBorder(nil, nil, nil, inspect,
			widget.NewLabel(row.Name+"  ·  "+row.ID)))
	}
	if len(items) == 0 {
		items = append(items, widget.NewLabel(g.tr.T("gui.trusted.empty")))
	}
	g.show(container.NewBorder(g.header(g.tr.T("gui.trusted.title")), nil, nil, nil, container.NewVScroll(container.NewVBox(items...))))
}

// inspectTrusted shows the full fingerprint and removes trust on confirmation.
func (g *gui) inspectTrusted(peers []trust.Peer, id string) {
	for _, p := range peers {
		if p.Fingerprint.Hex()[:16] != id {
			continue
		}
		body := widget.NewLabel(p.Name + "\n\n" + p.Fingerprint.Grouped())
		g.onUI(func() {
			dialog.ShowCustomConfirm(g.tr.T("gui.trusted.inspect"), g.tr.T("gui.trusted.remove"), g.tr.T("gui.close"), body, func(ok bool) {
				if !ok {
					return
				}
				if err := g.store.Remove(p.Fingerprint); err != nil {
					g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
					return
				}
				g.showTrusted()
			}, g.win)
		})
		return
	}
}

// showSettings sets the language (PRD §19, docs/IDIOMAS.md). The choice takes
// effect the next time the application starts.
func (g *gui) showSettings() {
	options := []string{"auto", "pt-BR", "en"}
	current, err := settings.Load(g.cfgPath)
	if err != nil {
		current = settings.Default
	}
	picker := widget.NewSelect(options, func(choice string) {
		if err := settings.Save(g.cfgPath, settings.Settings{Language: choice}); err != nil {
			g.notify(g.tr.T("gui.settings.title"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.notify(g.tr.T("gui.settings.title"), g.tr.T("gui.settings.restart"))
	})
	picker.SetSelected(current.Language)
	body := container.NewVBox(widget.NewLabel(g.tr.T("gui.settings.language")), picker,
		widget.NewLabel(g.tr.T("gui.settings.about")))
	g.show(container.NewBorder(g.header(g.tr.T("gui.settings.title")), nil, nil, nil, body))
}
