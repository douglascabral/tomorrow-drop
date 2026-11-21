package main

import (
	"context"
	"fmt"
	"image/color"
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/settings"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/ui"
)

// header is the top bar of a sub-screen: back button and title.
func (g *gui) header(title string) fyne.CanvasObject {
	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), g.showHome)
	name := newLabelStyled(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewBorder(nil, nil, back, nil, name)
}

// card wraps content in a rounded-looking surface.
func card(objs ...fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorSurface())
	bg.CornerRadius = 12
	return container.NewStack(bg, container.NewPadded(container.NewVBox(objs...)))
}

// showFull puts a screen in the window without the centered column: it uses
// the whole window, for screens that grow with it.
func (g *gui) showFull(content fyne.CanvasObject) {
	g.onUI(func() { g.win.SetContent(content) })
}

// show puts a screen in the window. On desktop the screen is a centered column
// of at most maxColumnWidth; on phones it takes the whole width.
func (g *gui) show(content fyne.CanvasObject) {
	if !g.mobile {
		content = container.New(&maxWidthLayout{max: maxColumnWidth}, content)
	}
	g.onUI(func() { g.win.SetContent(content) })
}

// scroll puts a screen in a scroller, so small windows never cut content.
func scroll(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewScroll(container.NewPadded(content))
}

// showHome is the first screen (layout 1): the device list with status, and
// access to manual pairing and settings. Send and receive are offered on the
// device screen, once a device is chosen from the list.
func (g *gui) showHome() {
	g.refreshDevices()
	// The address is read again on every visit, so the screen and the device
	// list follow a change of network.
	g.localAddrs = g.addrs()
	thisDevice := newLabelStyled(g.tr.T("gui.home.this_device", ui.LocalAddressText(g.localAddrs)), fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})

	refresh := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if err := g.ctl.Refresh(); err != nil {
			g.notify(g.tr.T("gui.home.refresh"), g.tr.T(apperr.MessageKey(err)))
		}
	})
	refresh.Importance = widget.LowImportance

	list := widget.NewList(
		func() int { return len(g.rows) },
		g.newDeviceRow,
		func(i widget.ListItemID, obj fyne.CanvasObject) { g.fillDeviceRow(obj, g.rows[i]) },
	)
	list.OnSelected = func(i widget.ListItemID) {
		list.Unselect(i)
		g.openDevice(g.rows[i])
	}
	g.devicesList = list

	manual := widget.NewButtonWithIcon(g.tr.T("gui.home.manual"), theme.ComputerIcon(), g.showManual)
	manual.Importance = widget.MediumImportance
	settingsBtn := widget.NewButtonWithIcon(g.tr.T("gui.home.settings"), theme.SettingsIcon(), g.showSettings)
	settingsBtn.Importance = widget.LowImportance

	head := container.NewVBox(
		container.NewCenter(logoImage(96)),
		newLabelStyled(g.tr.T("app.nome"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		newLabelStyled(g.tr.T("app.slogan"), fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		thisDevice,
		container.NewBorder(nil, nil, nil, refresh, newLabelStyled(g.tr.T("gui.home.devices"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})),
	)
	foot := container.NewVBox(manual, settingsBtn)
	g.show(container.NewBorder(container.NewPadded(head), container.NewPadded(foot), nil, nil, list))
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

// deviceRowParts are the labels of one device line, kept by reference so the
// list never depends on the order of the container's children.
type deviceRowParts struct {
	icon                  *widget.Icon
	name, address, status *widget.Label
}

// newDeviceRow builds one device line: name and address on the left, status on
// the right (layout 1). It records the labels for fillDeviceRow.
func (g *gui) newDeviceRow() fyne.CanvasObject {
	parts := &deviceRowParts{icon: widget.NewIcon(nil), name: newLabel(""), address: newLabel(""), status: widget.NewLabel("")}
	root := container.NewBorder(nil, nil, parts.icon, parts.status, container.NewVBox(parts.name, parts.address))
	g.rowParts[root] = parts
	return root
}

// fillDeviceRow sets the texts of a row built by newDeviceRow.
func (g *gui) fillDeviceRow(obj fyne.CanvasObject, row ui.DeviceRow) {
	parts := g.rowParts[obj]
	if parts == nil {
		return
	}
	parts.name.SetText(row.Name)
	parts.address.SetText(row.Address)
	parts.status.SetText(g.tr.T(row.StatusKey))
	// The row was laid out before its texts were known; lay it out again so the
	// status gets its full width.
	obj.Refresh()
	if row.Trusted {
		parts.status.Importance = widget.SuccessImportance
		parts.icon.SetResource(theme.ConfirmIcon())
		parts.icon.Show()
	} else {
		parts.status.Importance = widget.LowImportance
		parts.icon.SetResource(nil)
		parts.icon.Hide()
	}
}

// openDevice opens a trusted device (layout 5) or starts pairing with a new one (layout 2).
func (g *gui) openDevice(row ui.DeviceRow) {
	if row.Trusted {
		g.showTrustedDevice(row)
		return
	}
	g.connectTo(row)
}

// showTrustedDevice is the paired-device screen: actions and identity (layout 5).
func (g *gui) showTrustedDevice(row ui.DeviceRow) {
	fp, pairedAt := g.peerDetails(row.Address)
	name := newLabelStyled(row.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	addr := newLabelStyled(row.Address, fyne.TextAlignCenter, fyne.TextStyle{})
	badge := newLabelStyled(g.tr.T("ui.status.trusted"), fyne.TextAlignCenter, fyne.TextStyle{})
	badge.Importance = widget.SuccessImportance

	details := card(
		newLabel(g.tr.T("gui.device.paired_at", pairedAt)),
		newLabel(g.tr.T("gui.device.fingerprint")),
		newLabelStyled(fp, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
	)
	sendFiles := widget.NewButtonWithIcon(g.tr.T("gui.device.send_files"), theme.UploadIcon(), func() { g.showSend(row.Address, false) })
	sendFiles.Importance = widget.HighImportance
	sendText := widget.NewButtonWithIcon(g.tr.T("gui.device.send_text"), theme.DocumentIcon(), func() { g.showSend(row.Address, true) })
	receive := widget.NewButtonWithIcon(g.tr.T("gui.home.receive"), theme.DownloadIcon(), g.showReceive)
	remove := widget.NewButtonWithIcon(g.tr.T("gui.trusted.remove"), theme.DeleteIcon(), func() { g.removeTrustByAddress(row.Address) })
	remove.Importance = widget.DangerImportance

	body := container.NewVBox(
		container.NewCenter(canvas.NewImageFromResource(theme.ComputerIcon())),
		name, addr, container.NewCenter(badge),
		details,
		sendFiles, sendText, receive, remove,
	)
	g.show(container.NewBorder(g.header(g.tr.T("gui.device.title")), nil, nil, nil, scroll(body)))
}

// peerDetails returns the fingerprint and pairing date of the trusted device at addr.
func (g *gui) peerDetails(addr string) (string, string) {
	peers, err := g.store.List()
	if err != nil {
		return "", ""
	}
	ep, err := manualEndpoint(addr)
	if err != nil {
		return "", ""
	}
	for _, d := range g.reg.List(time.Now()) {
		if d.Addr == ep {
			for _, p := range peers {
				if p.Name == d.Name {
					return p.Fingerprint.Grouped(), p.AddedAt.Local().Format("02/01/2006 15:04")
				}
			}
		}
	}
	return "—", "—"
}

// removeTrustByAddress revokes trust in the device at addr after confirmation.
func (g *gui) removeTrustByAddress(addr string) {
	peers, err := g.store.List()
	if err != nil {
		g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	ep, err := manualEndpoint(addr)
	if err != nil {
		return
	}
	for _, d := range g.reg.List(time.Now()) {
		if d.Addr != ep {
			continue
		}
		for _, p := range peers {
			if p.Name != d.Name {
				continue
			}
			peer := p
			g.onUI(func() {
				dialog.ShowConfirm(g.tr.T("gui.trusted.remove"), g.tr.T("gui.trusted.remove_confirm", peer.Name), func(ok bool) {
					if !ok {
						return
					}
					if err := g.store.Remove(peer.Fingerprint); err != nil {
						g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
						return
					}
					g.showHome()
				}, g.win)
			})
			return
		}
	}
}

// showManual is the manual pairing screen: IP address and port (layout 4).
func (g *gui) showManual() {
	ip := widget.NewEntry()
	ip.SetPlaceHolder("192.168.1.42")
	port := widget.NewEntry()
	port.SetText(strconv.Itoa(defaultTLSPort))
	connect := widget.NewButtonWithIcon(g.tr.T("gui.home.connect"), theme.ConfirmIcon(), func() {
		ep, err := manualEndpoint(ip.Text + ":" + strings.TrimSpace(port.Text))
		if err != nil {
			g.notify(g.tr.T("gui.home.manual"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.connectTo(ui.DeviceRow{Name: ep.String(), Address: ep.String(), StatusKey: "ui.status.discovered"})
	})
	connect.Importance = widget.HighImportance
	body := card(
		newLabelStyled(g.tr.T("gui.manual.title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		newLabelStyled(g.tr.T("gui.manual.help"), fyne.TextAlignCenter, fyne.TextStyle{}),
		newLabel(g.tr.T("gui.manual.ip")), ip,
		newLabel(g.tr.T("gui.manual.port")), port,
		connect,
	)
	g.show(container.NewBorder(g.header(g.tr.T("gui.manual.title")), nil, nil, nil, scroll(body)))
}

// showSend is the send screen (layout 6): choose files or write text, then send.
// Clipboard sending keeps the explicit preview (PRD §12.2).
func (g *gui) showSend(target string, textMode bool) {
	var files []string
	fileList := container.NewVBox()
	summary := newLabel("")
	status := newLabel("")
	text := widget.NewMultiLineEntry()
	text.SetPlaceHolder(g.tr.T("gui.send.text_placeholder"))

	var refreshFiles func()
	refreshFiles = func() {
		fileList.Objects = nil
		var total int64
		for i, f := range files {
			idx := i
			info, _ := statSize(f)
			total += info
			remove := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
				files = append(files[:idx], files[idx+1:]...)
				refreshFiles()
			})
			remove.Importance = widget.LowImportance
			fileList.Add(card(container.NewBorder(nil, nil, widget.NewIcon(theme.DocumentIcon()), remove,
				container.NewVBox(newLabel(filepath.Base(f)), newLabel(ui.FormatBytes(info))))))
		}
		if len(files) > 0 {
			summary.SetText(g.tr.T("gui.send.summary", len(files), ui.FormatBytes(total)))
		} else {
			summary.SetText(g.tr.T("gui.send.no_files"))
		}
		fileList.Refresh()
	}

	pick := widget.NewButtonWithIcon(g.tr.T("gui.send.select_files"), theme.ContentAddIcon(), func() {
		g.onUI(func() {
			d := dialog.NewFileOpen(func(f fyne.URIReadCloser, err error) {
				if err != nil || f == nil {
					return
				}
				defer func() { _ = f.Close() }()
				if g.mobile {
					// A phone gives the content, not a path: copy it into the app.
					path, err := stagePicked(f, f.URI().Name(), filepath.Join(g.home, "picked"))
					if err != nil {
						g.notify(g.tr.T("gui.send.title"), g.tr.T(apperr.MessageKey(err)))
						return
					}
					files = append(files, path)
				} else {
					files = append(files, f.URI().Path())
				}
				refreshFiles()
			}, g.win)
			d.Resize(pickerSize(g.win.Canvas().Size()))
			d.Show()
		})
	})
	pick.Importance = widget.HighImportance

	dest := widget.NewEntry()
	dest.SetPlaceHolder(g.tr.T("gui.send.target_placeholder"))
	dest.SetText(target)
	destLine := newLabel(g.tr.T("gui.send.to", target))
	if target == "" {
		destLine.Hide()
	} else {
		dest.Hide()
	}

	// On a phone the send action is a labeled button across the screen; on
	// desktop it is a compact icon next to the summary.
	sendBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), nil)
	if g.mobile {
		sendBtn = widget.NewButtonWithIcon(g.tr.T("gui.send.send"), theme.MailSendIcon(), nil)
	}
	sendBtn.Importance = widget.HighImportance
	sendBtn.OnTapped = func() {
		if strings.TrimSpace(dest.Text) == "" {
			status.SetText(g.tr.T("gui.send.need_target"))
			return
		}
		var spec transfer.SendSpec
		if textMode {
			if strings.TrimSpace(text.Text) == "" {
				status.SetText(g.tr.T("gui.send.nothing"))
				return
			}
			spec = transfer.SendSpec{Kind: transfer.KindText, Text: text.Text}
		} else {
			plan, err := ui.PlanFiles(files)
			if err != nil {
				status.SetText(g.tr.T(apperr.MessageKey(err)))
				return
			}
			paths := make([]string, 0, len(plan.Items))
			for _, it := range plan.Items {
				paths = append(paths, it.Path)
			}
			spec = transfer.SendSpec{Kind: transfer.KindFile, Paths: paths}
		}
		g.startSend(dest.Text, spec, filepath.Base(firstOr(files, text.Text)))
	}

	hint := g.tr.T("gui.send.drop_hint")
	if g.mobile {
		hint = g.tr.T("gui.send.pick_hint_mobile")
	}
	clipboard := widget.NewButtonWithIcon(g.tr.T("gui.send.clipboard"), theme.ContentPasteIcon(), func() {
		g.previewClipboard(func(spec transfer.SendSpec) {
			if strings.TrimSpace(dest.Text) == "" {
				status.SetText(g.tr.T("gui.send.need_target"))
				return
			}
			g.startSend(dest.Text, spec, g.tr.T("gui.send.clipboard"))
		})
	})
	refreshFiles()
	title := g.tr.T("gui.send.title")
	if textMode {
		title = g.tr.T("gui.send.text_title")
	}
	if g.mobile {
		g.show(container.NewBorder(g.header(title), nil, nil, nil, scroll(g.sendPhoneBody(textMode, destLine, dest, target, text, pick, fileList, summary, sendBtn, status, clipboard, hint))))
		return
	}
	g.showFull(container.NewBorder(g.header(title), nil, nil, nil,
		g.sendDesktopBody(textMode, destLine, dest, target, text, pick, fileList, summary, sendBtn, status, clipboard, hint)))
}

// sendPhoneBody stacks the send screen for a phone; it scrolls as a whole.
func (g *gui) sendPhoneBody(textMode bool, destLine *widget.Label, dest *widget.Entry, target string, text *widget.Entry,
	pick *widget.Button, fileList *fyne.Container, summary *widget.Label, sendBtn *widget.Button, status *widget.Label,
	clipboard *widget.Button, hint string) fyne.CanvasObject {
	if textMode {
		return container.NewVBox(targetRow(g, destLine, dest, target), card(text), g.sendRow(sendBtn, status), clipboard)
	}
	return container.NewVBox(
		targetRow(g, destLine, dest, target),
		card(newLabelStyled(hint, fyne.TextAlignCenter, fyne.TextStyle{}), container.NewCenter(pick)),
		fileList,
		summary,
		g.sendRow(sendBtn, status),
		clipboard,
	)
}

// sendDesktopBody uses the whole window: the target and the drop area on top,
// the list of files or the text in the middle, where it takes the height that is
// left, and the send action and status at the bottom.
func (g *gui) sendDesktopBody(textMode bool, destLine *widget.Label, dest *widget.Entry, target string, text *widget.Entry,
	pick *widget.Button, fileList *fyne.Container, summary *widget.Label, sendBtn *widget.Button, status *widget.Label,
	clipboard *widget.Button, hint string) fyne.CanvasObject {
	top := targetRow(g, destLine, dest, target)
	if textMode {
		bottom := container.NewVBox(g.sendRow(sendBtn, status), clipboard)
		return container.NewBorder(top, bottom, nil, nil, card(text))
	}
	// The button keeps its own height: a centered box would stretch with the area.
	dropArea := card(newLabelStyled(hint, fyne.TextAlignCenter, fyne.TextStyle{}),
		container.NewHBox(layout.NewSpacer(), pick, layout.NewSpacer()))
	files := container.NewBorder(container.NewVBox(top, dropArea), nil, nil, nil, container.NewScroll(fileList))
	bottom := container.NewVBox(container.NewBorder(nil, nil, nil, sendBtn, summary), status, clipboard)
	return container.NewBorder(nil, bottom, nil, nil, files)
}

// sendRow is the bottom row of a send screen. On a phone the button spans the
// screen and the status text sits above it; on desktop they share one row.
func (g *gui) sendRow(sendBtn *widget.Button, status *widget.Label) fyne.CanvasObject {
	if g.mobile {
		return container.NewVBox(status, g.touch(sendBtn))
	}
	return container.NewBorder(nil, nil, nil, sendBtn, status)
}

// targetRow shows the recipient as text when it is already known, and as an
// input when the send starts from the home screen.
func targetRow(g *gui, line *widget.Label, input *widget.Entry, target string) fyne.CanvasObject {
	if target != "" {
		return card(line)
	}
	return container.NewVBox(newLabel(g.tr.T("gui.send.target")), input)
}

func firstOr(files []string, fallback string) string {
	if len(files) > 0 {
		return files[0]
	}
	return fallback
}

func statSize(path string) (int64, error) {
	plan, err := ui.PlanFiles([]string{path})
	if err != nil || len(plan.Items) == 0 {
		return 0, err
	}
	return plan.Items[0].Size, nil
}

// startSend runs one send and shows the progress screen (layout 7).
func (g *gui) startSend(dest string, spec transfer.SendSpec, label string) {
	ctx, done := g.beginOperation()
	title := g.tr.T("gui.progress.sending_to", dest)
	meter := newMeter(g, title, g.tr.T("gui.progress.preparing"), func() { done() })
	g.show(meter.screen)
	sendAgain := doneAction{label: g.tr.T("gui.done.send_again"), run: func() {
		g.showSend(dest, spec.Kind == transfer.KindText)
	}}
	go func() {
		defer done()
		start := time.Now()
		err := g.sendTo(ctx, dest, spec, func(p transfer.Progress) {
			meter.update(ui.ProgressFraction(p), label, p.TotalSent, p.TotalSize, time.Since(start))
		})
		if err != nil {
			if ctx.Err() != nil {
				g.showOutcome(false, g.tr.T("gui.progress.canceled"), "", nil)
				return
			}
			g.showOutcome(false, g.tr.T("gui.send.failed"), g.tr.T(apperr.MessageKey(err)), nil, sendAgain)
			return
		}
		g.showCompleted(g.tr.T("gui.send.done"), g.tr.T("gui.send.sent_to", dest), nil, sendAgain)
	}()
}

// meter is the progress screen: a ring with the percentage, the file, the
// speed and the time left.
type meter struct {
	g       *gui
	screen  fyne.CanvasObject
	ring    *canvas.Image
	percent *widget.Label
	file    *widget.Label
	detail  *widget.Label
	rate    *widget.Label
}

func newMeter(g *gui, title, subtitle string, cancel func()) *meter {
	m := &meter{g: g}
	m.ring = canvas.NewImageFromImage(ringImage(0, 220, colorPrimary(), colorTrack()))
	m.ring.SetMinSize(fyne.NewSize(220, 220))
	m.ring.FillMode = canvas.ImageFillContain
	m.percent = newLabelStyled("0%", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	m.file = newLabelStyled(subtitle, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	m.detail = newLabelStyled("", fyne.TextAlignCenter, fyne.TextStyle{})
	m.rate = newLabelStyled("", fyne.TextAlignCenter, fyne.TextStyle{})
	cancelBtn := widget.NewButtonWithIcon(g.tr.T("gui.send.cancel"), theme.CancelIcon(), cancel)
	cancelBtn.Importance = widget.MediumImportance
	ringBox := container.NewStack(m.ring, container.NewCenter(m.percent))
	m.screen = container.NewBorder(g.header(title), container.NewPadded(cancelBtn), nil, nil,
		scroll(container.NewVBox(container.NewCenter(ringBox), m.file, m.detail, m.rate)))
	return m
}

// update redraws the ring and labels. It is safe to call from any goroutine.
func (m *meter) update(fraction float64, label string, sent, total int64, elapsed time.Duration) {
	rate := 0.0
	if elapsed > 0 {
		rate = float64(sent) / elapsed.Seconds()
	}
	left := ""
	if rate > 0 && total > sent {
		left = fmtDuration(time.Duration(float64(total-sent)/rate) * time.Second)
	}
	m.g.onUI(func() {
		m.ring.Image = ringImage(fraction, 220, colorPrimary(), colorTrack())
		m.ring.Refresh()
		m.percent.SetText(fmt.Sprintf("%d%%", int(fraction*100)))
		m.file.SetText(label)
		m.detail.SetText(m.g.tr.T("gui.progress.bytes", ui.FormatBytes(sent), ui.FormatBytes(total)))
		m.rate.SetText(m.g.tr.T("gui.progress.rate", ui.FormatBytes(int64(rate)), left))
	})
}

func fmtDuration(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// showReceive is the waiting screen (layout 8). Receiving is always on while
// the application is in the foreground; this screen explains it.
func (g *gui) showReceive() {
	icon := canvas.NewImageFromResource(theme.DownloadIcon())
	icon.SetMinSize(fyne.NewSize(72, 72))
	info := card(
		newLabelStyled(g.tr.T("gui.receive.waiting_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		newLabelStyled(g.tr.T("gui.receive.waiting_text"), fyne.TextAlignCenter, fyne.TextStyle{}),
		newLabelStyled(g.tr.T("gui.receive.port", g.tlsPort), fyne.TextAlignCenter, fyne.TextStyle{Monospace: true}),
	)
	hint := newLabelStyled(g.tr.T("gui.receive.auto_hint"), fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	body := container.NewVBox(container.NewCenter(icon), info, hint)
	g.show(container.NewBorder(g.header(g.tr.T("gui.receive.screen_title")), nil, nil, nil, scroll(body)))
}

// showCompleted is the completion screen (layout 11).
// doneAction is a button of the completion screen.
type doneAction struct {
	label string
	run   func()
}

// showCompleted shows the result of a transfer with the actions given: a
// reception offers to receive again, a send offers to send again.
func (g *gui) showCompleted(title, subtitle string, files []string, next ...doneAction) {
	g.showOutcome(true, title, subtitle, files, next...)
}

// showOutcome shows the result of a transfer. A failed or cancelled transfer
// shows an error mark, never the success mark.
func (g *gui) showOutcome(ok bool, title, subtitle string, files []string, next ...doneAction) {
	icon := theme.ConfirmIcon()
	if !ok {
		icon = theme.ErrorIcon()
	}
	mark := canvas.NewImageFromResource(icon)
	mark.SetMinSize(fyne.NewSize(72, 72))
	items := container.NewVBox()
	for _, f := range files {
		items.Add(newLabel(filepath.Base(f)))
	}
	body := []fyne.CanvasObject{container.NewCenter(mark),
		newLabelStyled(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		newLabelStyled(subtitle, fyne.TextAlignCenter, fyne.TextStyle{})}
	if len(files) > 0 {
		body = append(body, card(items))
		open := widget.NewButtonWithIcon(g.tr.T("gui.done.open_folder"), theme.FolderOpenIcon(), func() { g.openFolder(filepath.Dir(files[0])) })
		open.Importance = widget.HighImportance
		body = append(body, open)
	}
	for _, a := range next {
		body = append(body, widget.NewButton(a.label, a.run))
	}
	g.show(container.NewBorder(g.header(g.tr.T("gui.done.title")), nil, nil, nil, scroll(container.NewVBox(body...))))
}

// openFolder opens a folder in the system file manager.
func (g *gui) openFolder(dir string) {
	u, err := url.Parse("file://" + dir)
	if err != nil {
		return
	}
	_ = g.app.OpenURL(u)
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
		items = append(items, card(newLabelStyled(row.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			newLabelStyled(row.Fingerprint, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})))
	}
	if len(items) == 0 {
		items = append(items, newLabel(g.tr.T("gui.trusted.empty")))
	}
	g.show(container.NewBorder(g.header(g.tr.T("gui.trusted.title")), nil, nil, nil, scroll(container.NewVBox(items...))))
}

// showSettings sets the language (docs/IDIOMAS.md). It applies on the next start.
func (g *gui) showSettings() {
	current, err := settings.Load(g.cfgPath)
	if err != nil {
		current = settings.Default
	}
	picker := languagePicker(current.Language, func(choice string) {
		if err := settings.Save(g.cfgPath, settings.Settings{Language: choice}); err != nil {
			g.notify(g.tr.T("gui.settings.title"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.notify(g.tr.T("gui.settings.title"), g.tr.T("gui.settings.restart"))
	})
	trusted := widget.NewButtonWithIcon(g.tr.T("gui.home.trusted"), theme.AccountIcon(), g.showTrusted)
	about := newLabel(g.tr.T("gui.settings.about"))
	about.Wrapping = fyne.TextWrapWord
	body := container.NewVBox(newLabel(g.tr.T("gui.settings.language")), g.touch(picker),
		trusted, about)
	g.show(container.NewBorder(g.header(g.tr.T("gui.settings.title")), nil, nil, nil, scroll(body)))
}

// languagePicker lists the languages with the current one selected. Only a
// choice made by the person calls onChoose: the selection set here does not.
func languagePicker(current string, onChoose func(string)) *widget.Select {
	picker := widget.NewSelect([]string{"auto", "pt-BR", "en"}, nil)
	picker.SetSelected(current)
	picker.OnChanged = onChoose
	return picker
}

// connectTo contacts a device the person chose. Trust is known only after the
// connection is authenticated (docs/DISCOVERY.md), so the device is contacted
// first: a trusted device opens its screen with the send and receive actions,
// and an unknown one goes to pairing.
func (g *gui) connectTo(row ui.DeviceRow) {
	ep, err := manualEndpoint(row.Address)
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
			g.reg.MarkTrusted(ep, true)
			g.refreshDevices()
			row.Trusted = true
			row.StatusKey = "ui.status.trusted"
			g.onUI(func() { g.showTrustedDevice(row) })
			return
		}
		g.pairAsked(g.ctx, s)
	}()
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
			newLabel(p.Snippet), func(ok bool) {
				if ok {
					send(transfer.SendSpec{Kind: p.Kind, Text: p.Text})
				}
			}, g.win)
	})
}

func colorSurface() color.Color { return theme.Color(theme.ColorNameInputBackground) }
func colorPrimary() color.Color { return theme.Color(theme.ColorNamePrimary) }
func colorTrack() color.Color   { return color.NRGBA{R: 0x2A, G: 0x27, B: 0x33, A: 0xFF} }
