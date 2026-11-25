package main

import (
	"bytes"
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
	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/settings"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// header is the top bar of a sub-screen: back button and title.
func (g *gui) header(title string) fyne.CanvasObject {
	return g.headerWith(title, g.showHome)
}

// headerWith is header with its own action for the back button.
func (g *gui) headerWith(title string, onBack func()) fyne.CanvasObject {
	back := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), onBack)
	name := newLabelStyled(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.New(&centredTitleLayout{}, back, container.NewCenter(g.compact(name)))
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
	g.onUI(func() {
		g.win.SetOnDropped(nil)
		g.win.SetContent(content)
	})
}

// show puts a screen in the window. On desktop the screen is a centered column
// of at most maxColumnWidth; on phones it takes the whole width.
func (g *gui) show(content fyne.CanvasObject) {
	if !g.mobile {
		content = container.New(&maxWidthLayout{max: maxColumnWidth}, content)
	}
	g.onUI(func() {
		g.win.SetOnDropped(nil)
		g.win.SetContent(content)
	})
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
	list := widget.NewList(
		func() int { return len(g.homeList) },
		g.newHomeItem,
		g.fillHomeItem,
	)
	list.OnSelected = func(i widget.ListItemID) {
		list.Unselect(i)
		g.openHomeItem(i)
	}
	g.devicesList = list

	manual := widget.NewButtonWithIcon(g.tr.T("gui.home.manual"), theme.ComputerIcon(), g.showManual)
	manual.Importance = widget.MediumImportance
	settingsBtn := widget.NewButtonWithIcon(g.tr.T("gui.home.settings"), theme.SettingsIcon(), g.showSettings)
	settingsBtn.Importance = widget.LowImportance

	headItems := []fyne.CanvasObject{}
	if g.sharedWaiting() {
		g.pendingMu.Lock()
		headItems = append(headItems, g.shareBanner(g.share))
		g.pendingMu.Unlock()
	}
	head := container.NewVBox(append(headItems, g.homeHead(func() {
		if err := g.ctl.Refresh(); err != nil {
			g.notify(g.tr.T("gui.home.refresh"), g.tr.T(apperr.MessageKey(err)))
		}
	}))...)
	// The notice that the device is visible only while the app is open stays on
	// the screen, below the two buttons, out of the way of the list.
	buttons := fyne.CanvasObject(container.NewGridWithColumns(2, manual, settingsBtn))
	if g.mobile {
		buttons = container.NewVBox(manual, settingsBtn)
	}
	foot := container.NewVBox(
		buttons,
		newLabelStyled(g.tr.T("gui.home.visible_only_open"), fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)
	// With no device the list is not blank: it says it is looking, and what to do.
	hint := newLabelStyled(g.tr.T("gui.home.empty_hint"), fyne.TextAlignCenter, fyne.TextStyle{})
	hint.Importance = widget.LowImportance
	g.emptyHint = container.NewVBox(
		g.compact(newLabelStyled(g.tr.T("gui.home.empty_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		hint,
	)
	if len(g.homeList) > 0 {
		g.emptyHint.Hide()
	}
	middle := container.NewStack(list, container.NewBorder(container.NewPadded(g.emptyHint), nil, nil, nil))
	g.show(container.NewBorder(container.NewPadded(head), container.NewPadded(foot), nil, nil, middle))
}

// homeHead is the top of the home screen, kept short so the device list gets the
// room: the logo with the name and slogan and the refresh button, then this
// device's address with its fingerprint. On desktop the logo is above the text. refresh runs when the button is
// pressed.
func (g *gui) homeHead(refresh func()) fyne.CanvasObject {
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), refresh)
	refreshBtn.Importance = widget.LowImportance
	logo := logoImage(g.logoSize())
	// This device's fingerprint sits beside its address, always at hand: it is
	// what the other person compares when pairing. A tap shows the whole of it.
	address := g.tr.T("gui.home.this_device", ui.LocalAddressText(g.localAddrs))
	fingerprint := g.fingerprintButton()

	if !g.mobile {
		// The logo above the name and the slogan, centred in the whole width; the
		// refresh button sits over the right edge.
		centred := container.NewVBox(
			container.NewCenter(logo),
			newLabelStyled(g.tr.T("app.nome"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			newLabelStyled(g.tr.T("app.slogan"), fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		)
		brand := container.NewStack(centred, container.NewBorder(container.NewHBox(layout.NewSpacer(), refreshBtn), nil, nil, nil))
		// The address in the left corner and the fingerprint in the right one.
		// The address does not wrap: a wrapped label has almost no minimum width,
		// so a narrow window squeezed it under the button.
		addressLabel := newLabelStyled(address, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		addressLabel.Wrapping = fyne.TextWrapOff
		device := container.NewBorder(nil, nil, addressLabel, fingerprint)
		return container.NewVBox(brand, device)
	}

	// On a phone the logo is at the left edge, the text beside it, and the
	// fingerprint button is only an icon, to keep the address on one line.
	title := container.NewVBox(
		newLabelStyled(g.tr.T("app.nome"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newLabelStyled(g.tr.T("app.slogan"), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
	)
	brand := container.NewBorder(nil, nil, logo, refreshBtn, container.NewCenter(title))
	fingerprint.SetText("")
	device := container.NewBorder(nil, nil, nil, fingerprint, newLabelStyled(address, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
	return g.compact(container.NewVBox(brand, device))
}

// fingerprintButton opens the whole of this device's fingerprint. It shows no
// part of it: seeing only the start would invite people to compare only the start.
func (g *gui) fingerprintButton() *widget.Button {
	b := widget.NewButtonWithIcon(g.tr.T("gui.fingerprint.view"), theme.VisibilityIcon(), g.showFingerprint)
	b.Importance = widget.LowImportance
	return b
}

// fingerprintContent is what the fingerprint view shows: the whole fingerprint,
// in four short lines, with a button to copy it.
func (g *gui) fingerprintContent() fyne.CanvasObject {
	grouped := g.id.Fingerprint().Grouped()
	copyBtn := widget.NewButtonWithIcon(g.tr.T("gui.receive.copy"), theme.ContentCopyIcon(), func() {
		g.app.Clipboard().SetContent(grouped)
	})
	return container.NewVBox(
		newLabel(g.tr.T("gui.fingerprint.hint")),
		newLabelStyled(ui.FingerprintBlock(grouped), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
		copyBtn,
	)
}

// showFingerprint shows this device's whole fingerprint.
func (g *gui) showFingerprint() {
	g.onUI(func() {
		dialog.NewCustom(g.tr.T("gui.fingerprint.title"), g.tr.T("gui.close"), g.fingerprintContent(), g.win).Show()
	})
}

// logoSize is the size of the logo on the home screen: smaller on a phone, so
// the list gets the room.
func (g *gui) logoSize() float32 {
	if g.mobile {
		return 40
	}
	return 44
}

// refreshDevices rebuilds the device list from the ephemeral registry (task 022).
func (g *gui) refreshDevices() {
	devices := g.reg.List(time.Now())
	for i := range devices {
		// A device already shown by an authenticated connection keeps its paired
		// name, not the computer name it announces.
		// The name comes from the paired record, so the list reads the current name
		// at once. Trust is not shown until a connection proves it.
		if name, ok := g.pairedNameAt(devices[i].Addr); ok {
			devices[i].Name = name
		}
		if p, ok := g.pairedAt(devices[i].Addr); ok {
			devices[i].Name = p.Name
			devices[i].Trusted = true
		}
	}
	g.rows = ui.DeviceRows(devices)
	g.homeList = g.buildHomeList()
	g.onUI(func() {
		if g.devicesList != nil {
			g.devicesList.Refresh()
		}
		if g.emptyHint != nil {
			if len(g.homeList) == 0 {
				g.emptyHint.Show()
			} else {
				g.emptyHint.Hide()
			}
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
	parts := &deviceRowParts{icon: widget.NewIcon(nil), name: newNameLabel(""), address: newNameLabel(""), status: widget.NewLabel("")}
	inner := container.NewBorder(nil, nil, parts.icon, parts.status, container.NewVBox(parts.name, parts.address))
	// The text is compact, and the row keeps a height that is easy to tap.
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, 72))
	root := container.NewStack(spacer, container.NewPadded(g.compact(inner)))
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
	parts.address.SetText(hostOf(row.Address))
	parts.status.SetText(g.tr.T(row.StatusKey))
	// The row was laid out before its texts were known; lay it out again so the
	// status gets its full width.
	obj.Refresh()
	// A paired device is online (green check) or offline (red cross): there is no
	// grey presence state. A found device shows a neutral computer. Every line
	// keeps its icon, so the names line up.
	switch {
	case row.Trusted && row.Presence == discovery.PresenceOnline:
		parts.status.Importance = widget.SuccessImportance
		parts.icon.SetResource(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
	case row.Trusted:
		parts.status.Importance = widget.DangerImportance
		parts.icon.SetResource(theme.NewErrorThemedResource(theme.CancelIcon()))
	default:
		parts.status.Importance = widget.LowImportance
		parts.icon.SetResource(theme.ComputerIcon())
	}
	parts.icon.Show()
	// The importance is drawn only on a refresh. The list reuses rows, so without
	// this the status kept the colour of the device the row showed before.
	parts.status.Refresh()
}

// openDevice opens a trusted device (layout 5) or starts pairing with a new one (layout 2).
func (g *gui) openDevice(row ui.DeviceRow) {
	if row.Trusted {
		// A paired device that is offline, or has no known address, cannot be
		// contacted, so its screen opens from the trust list. That is where it is
		// unpaired, as on the iPhone and iPad.
		if row.Fingerprint != "" && (row.Address == "" || row.Presence != discovery.PresenceOnline) {
			g.showTrustedDevice(row)
			return
		}
		if row.Address == "" {
			g.notify(g.tr.T("gui.home.trusted"), g.tr.T("gui.home.no_address_hint"))
			return
		}
		// Something another app shared waits for a device: a paired one gets it now.
		if g.sharedWaiting() {
			g.sendShared(row.Address)
			return
		}
	}
	// Connecting first proves the device is still paired, and then shows its
	// actions (send, receive). A discovered device goes to pairing the same way.
	g.connectTo(row, false)
}

// showTrustedDevice is the paired-device screen: actions first, identity below
// (layout 5). Unpairing is a quiet action at the end, since it is rare and the
// people have to pair again.
func (g *gui) showTrustedDevice(row ui.DeviceRow) {
	fp, pairedAt := g.peerDetails(row)
	icon := canvas.NewImageFromResource(theme.ComputerIcon())
	icon.SetMinSize(fyne.NewSize(56, 56))
	icon.FillMode = canvas.ImageFillContain
	name := newNameLabel(row.Name)
	name.Alignment, name.TextStyle = fyne.TextAlignCenter, fyne.TextStyle{Bold: true}
	addr := newNameLabel(hostOf(row.Address))
	addr.Alignment, addr.TextStyle = fyne.TextAlignCenter, fyne.TextStyle{Monospace: true}
	addr.Importance = widget.LowImportance
	badge := newLabelStyled(g.tr.T("ui.status.trusted"), fyne.TextAlignCenter, fyne.TextStyle{})
	badge.Importance = widget.SuccessImportance

	sendFiles := widget.NewButtonWithIcon(g.tr.T("gui.device.send_files"), theme.UploadIcon(), func() { g.withAddress(row, func() { g.showSend(row.Address, false) }) })
	sendFiles.Importance = widget.HighImportance
	sendText := widget.NewButtonWithIcon(g.tr.T("gui.device.send_text"), theme.DocumentIcon(), func() { g.withAddress(row, func() { g.showSend(row.Address, true) }) })
	sendClipboard := widget.NewButtonWithIcon(g.tr.T("gui.send.clipboard"), theme.ContentPasteIcon(), func() {
		g.withAddress(row, func() {
			g.previewClipboard(func(spec transfer.SendSpec) {
				g.startSend(row.Address, spec, g.tr.T("gui.send.clipboard"))
			})
		})
	})
	receive := widget.NewButtonWithIcon(g.tr.T("gui.home.receive"), theme.DownloadIcon(), g.showReceive)

	details := card(g.compact(container.NewVBox(
		newLabel(g.tr.T("gui.device.paired_at", pairedAt)),
		newLabel(g.tr.T("gui.device.fingerprint")),
		newLabelStyled(ui.FingerprintBlock(fp), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
	)))
	remove := widget.NewButtonWithIcon(g.tr.T("gui.trusted.remove"), theme.DeleteIcon(), func() {
		if peer, ok := g.peerOf(row); ok {
			g.confirmUnpair(peer, g.showHome)
		}
	})
	remove.Importance = widget.LowImportance
	rename := widget.NewButtonWithIcon(g.tr.T("gui.trusted.rename"), theme.DocumentCreateIcon(), func() {
		if peer, ok := g.peerOf(row); ok {
			g.askRename(peer)
		}
	})
	rename.Importance = widget.LowImportance

	body := container.NewVBox(
		container.NewCenter(icon),
		g.compact(container.NewVBox(name, addr)),
		g.fingerprintButton(),
		g.compact(container.NewCenter(badge)),
		sendFiles, sendText, sendClipboard, receive,
		details,
		rename,
		remove,
	)
	g.show(container.NewBorder(g.header(g.tr.T("gui.device.title")), nil, nil, nil, scroll(body)))
}

// peerOf returns the paired device a device screen is about: found by its
// fingerprint in the trust list when the row has one (so an offline device
// works), otherwise by the address a connection showed it at.
func (g *gui) peerOf(row ui.DeviceRow) (trust.Peer, bool) {
	if row.Fingerprint != "" {
		peers, err := g.store.List()
		if err != nil {
			return trust.Peer{}, false
		}
		for _, p := range peers {
			if p.Fingerprint.Hex() == row.Fingerprint {
				return p, true
			}
		}
		return trust.Peer{}, false
	}
	ep, err := manualEndpoint(row.Address)
	if err != nil {
		return trust.Peer{}, false
	}
	return g.pairedAt(ep)
}

// peerDetails returns the fingerprint and pairing date of the paired device of row.
func (g *gui) peerDetails(row ui.DeviceRow) (string, string) {
	p, ok := g.peerOf(row)
	if !ok {
		return "—", "—"
	}
	return p.Fingerprint.Grouped(), p.AddedAt.Local().Format("02/01/2006 15:04")
}

// withAddress runs send only when the device has a known address; otherwise it
// says how to give it one.
func (g *gui) withAddress(row ui.DeviceRow, send func()) {
	if row.Address == "" {
		g.notify(g.tr.T("gui.home.trusted"), g.tr.T("gui.home.no_address_hint"))
		return
	}
	send()
}

// showManual is the manual pairing screen: IP address and port (layout 4).
func (g *gui) showManual() {
	ip := widget.NewEntry()
	ip.SetPlaceHolder("192.168.1.42")
	port := widget.NewEntry()
	port.SetText(strconv.Itoa(defaultTLSPort))
	connect := func() {
		ep, err := manualEndpoint(strings.TrimSpace(ip.Text) + ":" + strings.TrimSpace(port.Text))
		if err != nil {
			g.notify(g.tr.T("gui.home.manual"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.connectTo(ui.DeviceRow{Name: ep.String(), Address: ep.String(), StatusKey: "ui.status.discovered"}, true)
	}
	// Enter connects from either field.
	ip.OnSubmitted = func(string) { connect() }
	port.OnSubmitted = func(string) { connect() }
	connectBtn := widget.NewButtonWithIcon(g.tr.T("gui.home.connect"), theme.ConfirmIcon(), connect)
	connectBtn.Importance = widget.HighImportance
	body := card(
		newLabelStyled(g.tr.T("gui.manual.help"), fyne.TextAlignCenter, fyne.TextStyle{}),
		g.compact(newLabel(g.tr.T("gui.manual.ip"))), ip,
		g.compact(newLabel(g.tr.T("gui.manual.port"))), port,
		connectBtn,
	)
	// The card keeps its natural height instead of filling the screen.
	g.show(container.NewBorder(g.header(g.tr.T("gui.manual.title")), nil, nil, nil, scroll(container.NewVBox(body))))
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

	// updateSend enables Send only when there is a recipient and something to send.
	var updateSend func()
	var refreshFiles func()
	refreshFiles = func() {
		defer func() {
			if updateSend != nil {
				updateSend()
			}
		}()
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
				g.compact(container.NewVBox(newNameLabel(filepath.Base(f)), newLabel(ui.FormatBytes(info)))))))
		}
		if len(files) > 0 {
			summary.SetText(g.tr.T("gui.send.summary", g.countFiles(len(files)), ui.FormatBytes(total)))
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
	destLine := newNameLabel(g.tr.T("gui.send.to", g.displayName(target)))
	if target == "" {
		destLine.Hide()
	} else {
		dest.Hide()
	}

	// The send action always says what it does; on a phone it spans the screen.
	sendBtn := widget.NewButtonWithIcon(g.tr.T("gui.send.send"), theme.MailSendIcon(), nil)
	sendBtn.Importance = widget.HighImportance
	updateSend = func() {
		ready := strings.TrimSpace(dest.Text) != ""
		if textMode {
			ready = ready && strings.TrimSpace(text.Text) != ""
		} else {
			ready = ready && len(files) > 0
		}
		if ready {
			sendBtn.Enable()
		} else {
			sendBtn.Disable()
		}
	}
	text.OnChanged = func(string) { updateSend() }
	dest.OnChanged = func(string) { updateSend() }
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
	updateSend()
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
	if !textMode {
		// Files can be dropped on the window. It is queued after the screen
		// change, which clears the previous screen's handler.
		g.onUI(func() {
			g.win.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
				if added := droppedPaths(uris); len(added) > 0 {
					files = append(files, added...)
					refreshFiles()
				}
			})
		})
	}
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
		dropCard(newLabelStyled(hint, fyne.TextAlignCenter, fyne.TextStyle{}), pick),
		fileList,
		g.compact(summary),
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
	dropArea := dropCard(newLabelStyled(hint, fyne.TextAlignCenter, fyne.TextStyle{}), pick)
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
		return card(g.compact(line))
	}
	return container.NewVBox(g.compact(newLabel(g.tr.T("gui.send.target"))), input)
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
	title := g.tr.T("gui.progress.sending_to", g.displayName(dest))
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
		g.showCompleted(g.tr.T("gui.send.done"), g.tr.T("gui.send.sent_to", g.displayName(dest)), nil, sendAgain)
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
	m.file = newNameLabel(subtitle)
	m.file.Alignment, m.file.TextStyle = fyne.TextAlignCenter, fyne.TextStyle{Bold: true}
	m.detail = newLabelStyled("", fyne.TextAlignCenter, fyne.TextStyle{})
	m.rate = newLabelStyled("", fyne.TextAlignCenter, fyne.TextStyle{})
	cancelBtn := widget.NewButtonWithIcon(g.tr.T("gui.send.cancel"), theme.CancelIcon(), cancel)
	cancelBtn.Importance = widget.MediumImportance
	ringBox := container.NewStack(m.ring, container.NewCenter(m.percent))
	m.screen = container.NewBorder(g.header(title), container.NewPadded(cancelBtn), nil, nil,
		scroll(container.NewVBox(container.NewCenter(ringBox), g.compact(container.NewVBox(m.file, m.detail, m.rate)))))
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
		g.compact(newLabelStyled(g.tr.T("gui.receive.waiting_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		newLabelStyled(g.tr.T("gui.receive.waiting_text"), fyne.TextAlignCenter, fyne.TextStyle{}),
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
		items.Add(newNameLabel(filepath.Base(f)))
	}
	body := []fyne.CanvasObject{container.NewCenter(mark),
		g.compact(newLabelStyled(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		newLabelStyled(subtitle, fyne.TextAlignCenter, fyne.TextStyle{})}
	if len(files) > 0 {
		body = append(body, card(g.compact(items)))
		// A phone has no folder to open: the files were saved where the person chose.
		if !g.mobile {
			open := widget.NewButtonWithIcon(g.tr.T("gui.done.open_folder"), theme.FolderOpenIcon(), func() { g.openFolder(filepath.Dir(files[0])) })
			open.Importance = widget.HighImportance
			body = append(body, open)
		}
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

// showTrusted lists the paired devices. Each one can be unpaired from here, so
// it does not depend on the device being online.
func (g *gui) showTrusted() {
	peers, err := g.store.List()
	if err != nil {
		g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	items := make([]fyne.CanvasObject, 0, len(peers))
	for i := range peers {
		peer := peers[i]
		remove := widget.NewButtonWithIcon(g.tr.T("gui.trusted.remove"), theme.DeleteIcon(), func() { g.confirmUnpair(peer, g.showTrusted) })
		remove.Importance = widget.LowImportance
		rename := widget.NewButtonWithIcon(g.tr.T("gui.trusted.rename"), theme.DocumentCreateIcon(), func() { g.askRename(peer) })
		rename.Importance = widget.LowImportance
		fingerprint := newLabelStyled(ui.FingerprintBlock(peer.Fingerprint.Grouped()), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		fingerprint.Importance = widget.LowImportance
		items = append(items, card(
			g.compact(container.NewVBox(boldName(peer.Name), fingerprint)),
			container.NewHBox(rename, remove),
		))
	}
	if len(items) == 0 {
		items = append(items, g.compact(newLabel(g.tr.T("gui.trusted.empty"))))
	}
	g.show(container.NewBorder(g.header(g.tr.T("gui.trusted.title")), nil, nil, nil, scroll(container.NewVBox(items...))))
}

// showSettings sets the language (docs/LANGUAGES.md). It applies on the next start.
func (g *gui) showSettings() {
	current, err := settings.Load(g.cfgPath)
	if err != nil {
		current = settings.Default
	}
	picker := languagePicker(current.Language, g.tr.T("gui.settings.language_auto"), func(choice string) {
		if err := settings.Save(g.cfgPath, settings.Settings{Language: choice}); err != nil {
			g.notify(g.tr.T("gui.settings.title"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		g.notify(g.tr.T("gui.settings.title"), g.tr.T("gui.settings.restart"))
	})
	trusted := widget.NewButtonWithIcon(g.tr.T("gui.home.trusted"), theme.AccountIcon(), g.showTrusted)
	about := newLabel(g.tr.T("gui.settings.about"))
	about.Wrapping = fyne.TextWrapWord
	body := container.NewVBox(g.compact(newLabel(g.tr.T("gui.settings.language"))), g.touch(picker),
		trusted, about)
	g.show(container.NewBorder(g.header(g.tr.T("gui.settings.title")), nil, nil, nil, scroll(body)))
}

// language is a choice of the language picker: the code that is saved and the
// name people read. Each language is named in its own language, so it can be
// found even when the app is in a language the person does not read.
type language struct{ code, label string }

func languages(autoLabel string) []language {
	return []language{{"auto", autoLabel}, {"pt-BR", "Português (Brasil)"}, {"en", "English"}}
}

// languagePicker lists the languages by name with the current one selected.
// Only a choice made by the person calls onChoose, with the language code: the
// selection set here does not.
func languagePicker(current, autoLabel string, onChoose func(code string)) *widget.Select {
	langs := languages(autoLabel)
	labels := make([]string, len(langs))
	for i, l := range langs {
		labels[i] = l.label
	}
	picker := widget.NewSelect(labels, nil)
	for _, l := range langs {
		if l.code == current {
			picker.SetSelected(l.label)
		}
	}
	picker.OnChanged = func(label string) {
		for _, l := range langs {
			if l.label == label {
				onChoose(l.code)
			}
		}
	}
	return picker
}

// connectTo contacts a device the person chose. Trust is known only after the
// connection is authenticated (docs/DISCOVERY.md), so the device is contacted
// first: a trusted device opens its screen with the send and receive actions,
// and an unknown one goes to pairing. byAddress is true for a device the person
// typed as IP:port: once it is paired it stays in the list for this session.
func (g *gui) connectTo(row ui.DeviceRow, byAddress bool) {
	ep, err := manualEndpoint(row.Address)
	if err != nil {
		g.notify(g.tr.T("gui.pair.title"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	g.background(func() {
		s, err := transport.Dial(g.ctx, ep, g.id, g.store)
		if err != nil {
			g.notify(g.tr.T("gui.pair.title"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		defer func() { _ = s.Close() }()
		if s.State() == protocol.StateTrusted {
			if g.rememberPaired(ep, s.PeerKey()) {
				g.rememberAddress(ep, s.PeerKey())
				if p, ok := g.pairedAt(ep); ok {
					row.Name = p.Name
				}
			}
			g.markListed(ep, row.Name, byAddress)
			g.refreshDevices()
			row.Trusted = true
			row.StatusKey = "ui.status.trusted"
			g.onUI(func() { g.showTrustedDevice(row) })
			return
		}
		g.pairHint = row.Name
		paired := g.pairAsked(g.ctx, s, false)
		g.pairHint = ""
		if paired {
			// The person started this pairing, so go on to what comes next: the
			// device, with its send and receive actions.
			if g.rememberPaired(ep, s.PeerKey()) {
				g.rememberAddress(ep, s.PeerKey())
				if p, ok := g.pairedAt(ep); ok {
					row.Name = p.Name
				}
			}
			g.markListed(ep, row.Name, byAddress)
			g.refreshDevices()
			row.Trusted = true
			row.StatusKey = "ui.status.trusted"
			g.onUI(func() { g.showTrustedDevice(row) })
		}
	})
}

// markListed marks a device paired at ep as trusted in the list. A device
// connected to by address (byAddress) is added to the list even when no
// announcement reached this device, and stays there while the app runs.
func (g *gui) markListed(ep netip.AddrPort, name string, byAddress bool) {
	if byAddress {
		_ = g.reg.Pin(ep, name, time.Now())
	}
	g.reg.MarkTrusted(ep, true)
	// The connection just succeeded, so the device is online now.
	g.presence.Record(ep, true, time.Now())
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
	// Files copied in the file manager are sent first, as the files themselves.
	if paths := clipboardPaths(); len(paths) > 0 {
		g.previewClipboardPaths(paths, send)
		return
	}
	// On phones a copied file comes as its contents: it is copied into the app
	// first, then sent like a file the person chose.
	if files := clipboardContents(); len(files) > 0 {
		g.previewClipboardContents(files, send)
		return
	}
	// A picture on the clipboard (a screenshot, for example) is sent before text:
	// the text clipboard API cannot see it.
	if data, ext, ok := clipboardImage(); ok {
		g.previewClipboardImage(data, ext, send)
		return
	}
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

// dropGap is the room between the parts of the drop area and to its edges.
const dropGap = 14

// dropCard is the area where files are dropped or chosen: a line of text and the
// choose button, with room around them so they do not touch the edges.
func dropCard(hint, pick fyne.CanvasObject) fyne.CanvasObject {
	return card(
		vgap(dropGap),
		hint,
		vgap(dropGap),
		container.NewCenter(pick),
		vgap(dropGap),
	)
}

// vgap is an empty vertical space of the given height.
func vgap(h float32) fyne.CanvasObject {
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(0, h))
	return gap
}

// previewClipboardImage asks before sending a picture from the clipboard. The
// picture is sent as a file, like any file the person chose.
func (g *gui) previewClipboardImage(data []byte, ext string, send func(transfer.SendSpec)) {
	size := ui.FormatBytes(int64(len(data)))
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.send.clipboard_image_title", size), g.tr.T("gui.send.send"), g.tr.T("gui.pair.cancel"),
			newLabel(g.tr.T("gui.send.clipboard_image_body", size)), func(ok bool) {
				if !ok {
					return
				}
				spec, err := g.imageSpec(data, ext)
				if err != nil {
					g.notify(g.tr.T("gui.send.clipboard"), g.tr.T(apperr.MessageKey(err)))
					return
				}
				send(spec)
			}, g.win)
	})
}

// previewClipboardPaths asks before sending files copied on the clipboard. They
// are sent from where they are; nothing is copied.
func (g *gui) previewClipboardPaths(paths []string, send func(transfer.SendSpec)) {
	plan, err := ui.PlanFiles(paths)
	if err != nil {
		g.notify(g.tr.T("gui.send.clipboard"), g.tr.T(apperr.MessageKey(err)))
		return
	}
	names := make([]string, 0, len(plan.Items))
	for _, it := range plan.Items {
		names = append(names, it.Name)
	}
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.send.clipboard_files_title", g.countFiles(len(plan.Items)), ui.FormatBytes(plan.Total)),
			g.tr.T("gui.send.send"), g.tr.T("gui.pair.cancel"),
			newNameLabel(strings.Join(names, ", ")), func(ok bool) {
				if ok {
					send(transfer.SendSpec{Kind: transfer.KindFile, Paths: paths})
				}
			}, g.win)
	})
}

// previewClipboardContents asks before sending files copied on a phone. They are
// copied into the application's folder and sent from there.
func (g *gui) previewClipboardContents(files []sharedFile, send func(transfer.SendSpec)) {
	var paths []string
	var total int64
	var names []string
	for _, f := range files {
		if int64(len(f.data)) > limits.MaxFileBytes {
			g.notify(g.tr.T("gui.send.clipboard"), g.tr.T(apperr.MessageKey(apperr.ErrLimitExceeded)))
			return
		}
		path, err := stagePicked(bytes.NewReader(f.data), safeSharedName(f.name, f.mime), filepath.Join(g.home, "clipboard"))
		if err != nil {
			g.notify(g.tr.T("gui.send.clipboard"), g.tr.T(apperr.MessageKey(err)))
			return
		}
		paths = append(paths, path)
		total += int64(len(f.data))
		names = append(names, filepath.Base(path))
	}
	if len(paths) == 0 {
		return
	}
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.send.clipboard_files_title", g.countFiles(len(paths)), ui.FormatBytes(total)),
			g.tr.T("gui.send.send"), g.tr.T("gui.pair.cancel"),
			newNameLabel(strings.Join(names, ", ")), func(ok bool) {
				if ok {
					send(transfer.SendSpec{Kind: transfer.KindFile, Paths: paths})
				}
			}, g.win)
	})
}
