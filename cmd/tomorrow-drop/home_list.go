package main

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/ui"
)

// The home list has two sections: the paired devices, from the trust store
// with the address they last answered at, and the devices that discovery found
// and that are not paired. A paired device is never listed twice.

type homeKind int

const (
	homeHeader homeKind = iota
	homeDevice
	homeNote
)

// homeItem is one line of the home list.
type homeItem struct {
	kind  homeKind
	title string
	row   ui.DeviceRow
}

// homeItemParts are the widgets of one line of the home list: a section title,
// a note, or a device line. Only the one the line needs is shown.
type homeItemParts struct {
	header, note *widget.Label
	row          fyne.CanvasObject
}

// pairedRows lists the paired devices from the trust store. A device whose
// address is not known yet has no address, and says so.
func (g *gui) pairedRows() []ui.DeviceRow {
	peers, err := g.store.List()
	if err != nil {
		return nil
	}
	addrs := g.pairedAddresses()
	rows := make([]ui.DeviceRow, 0, len(peers))
	for _, p := range peers {
		row := ui.DeviceRow{Name: p.Name, Address: addrs[p.Fingerprint.Hex()], Trusted: true, StatusKey: "ui.status.trusted"}
		if row.Address == "" {
			row.StatusKey = "ui.status.no_address"
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

// buildHomeList arranges the paired devices and the found ones into sections.
// It is built again whenever the device list changes.
func (g *gui) buildHomeList() []homeItem {
	paired := g.pairedRows()
	pairedAt := map[string]bool{}
	for _, row := range paired {
		if row.Address != "" {
			pairedAt[row.Address] = true
		}
	}
	var found []ui.DeviceRow
	for _, row := range g.rows {
		if !row.Trusted && !pairedAt[row.Address] {
			found = append(found, row)
		}
	}
	// With no device at all the list is empty, and the screen explains it.
	if len(paired) == 0 && len(found) == 0 {
		return nil
	}
	var items []homeItem
	if len(paired) > 0 {
		items = append(items, homeItem{kind: homeHeader, title: g.tr.T("gui.home.trusted")})
		for _, row := range paired {
			items = append(items, homeItem{kind: homeDevice, row: row})
		}
	}
	items = append(items, homeItem{kind: homeHeader, title: g.tr.T("gui.home.devices")})
	if len(found) == 0 {
		items = append(items, homeItem{kind: homeNote, title: g.tr.T("gui.home.empty_title")})
	}
	for _, row := range found {
		items = append(items, homeItem{kind: homeDevice, row: row})
	}
	return items
}

// newHomeItem builds a line of the home list able to show any kind of item.
func (g *gui) newHomeItem() fyne.CanvasObject {
	header := newLabelStyled("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	note := newLabelStyled("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	note.Importance = widget.LowImportance
	parts := &homeItemParts{header: header, note: note, row: g.newDeviceRow()}
	wrap := container.NewStack(header, note, parts.row)
	g.homeParts[wrap] = parts
	return wrap
}

// fillHomeItem sets the line i of the home list on an object made by newHomeItem.
func (g *gui) fillHomeItem(i int, obj fyne.CanvasObject) {
	parts := g.homeParts[obj]
	if parts == nil || i < 0 || i >= len(g.homeList) {
		return
	}
	it := g.homeList[i]
	parts.header.Hide()
	parts.note.Hide()
	parts.row.Hide()
	switch it.kind {
	case homeHeader:
		parts.header.SetText(it.title)
		parts.header.Show()
	case homeNote:
		parts.note.SetText(it.title)
		parts.note.Show()
	case homeDevice:
		g.fillDeviceRow(parts.row, it.row)
		parts.row.Show()
	}
}

// openHomeItem reacts to a tap on line i of the home list. Section titles and
// notes do nothing.
func (g *gui) openHomeItem(i int) {
	if i < 0 || i >= len(g.homeList) || g.homeList[i].kind != homeDevice {
		return
	}
	g.openDevice(g.homeList[i].row)
}
