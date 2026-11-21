package main

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/storage"
	"tomorrow-drop/internal/trust"
)

// A device announces only a name and a port, and its name is not the one the
// person gave it when pairing. Who a device is becomes known when an
// authenticated connection shows its key, so the application remembers, for the
// addresses it has contacted, which paired device answered there.

// rememberPaired records the paired device whose key was seen at ep. It
// returns false when the key is not in the list of paired devices.
func (g *gui) rememberPaired(ep netip.AddrPort, key []byte) bool {
	peers, err := g.store.List()
	if err != nil {
		return false
	}
	for _, p := range peers {
		if bytes.Equal(p.Key, key) {
			g.pairedMu.Lock()
			g.paired[ep] = p
			g.pairedMu.Unlock()
			return true
		}
	}
	return false
}

// pairedAt returns the paired device remembered at ep.
func (g *gui) pairedAt(ep netip.AddrPort) (trust.Peer, bool) {
	g.pairedMu.Lock()
	defer g.pairedMu.Unlock()
	p, ok := g.paired[ep]
	return p, ok
}

// unpairDevice stops trusting a device and forgets where it was seen.
func (g *gui) unpairDevice(peer trust.Peer) error {
	if err := g.store.Remove(peer.Fingerprint); err != nil {
		return err
	}
	g.pairedMu.Lock()
	for ep, p := range g.paired {
		if bytes.Equal(p.Key, peer.Key) {
			delete(g.paired, ep)
			g.reg.MarkTrusted(ep, false)
			g.reg.Unpin(ep)
		}
	}
	g.pairedMu.Unlock()
	g.refreshDevices()
	return nil
}

// confirmUnpair asks before unpairing, since the people have to pair again to
// exchange files. after runs once the device is unpaired.
func (g *gui) confirmUnpair(peer trust.Peer, after func()) {
	g.onUI(func() {
		// The buttons say what they do ("Unpair", "Cancel") in the person's
		// language, instead of the toolkit's "Yes" and "No".
		dialog.ShowCustomConfirm(g.tr.T("gui.trusted.remove"), g.tr.T("gui.trusted.remove"), g.tr.T("gui.pair.cancel"),
			newLabel(g.tr.T("gui.trusted.remove_confirm", peer.Name)), func(ok bool) {
				if !ok {
					return
				}
				if err := g.unpairDevice(peer); err != nil {
					g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
					return
				}
				after()
			}, g.win)
	})
}

// renameDevice gives a paired device a new name. The list and the device screen
// show the new name at once.
func (g *gui) renameDevice(peer trust.Peer, name string) error {
	if err := g.store.Rename(peer.Fingerprint, name); err != nil {
		return err
	}
	g.pairedMu.Lock()
	for ep, p := range g.paired {
		if bytes.Equal(p.Key, peer.Key) {
			p.Name = name
			g.paired[ep] = p
		}
	}
	g.pairedMu.Unlock()
	g.refreshDevices()
	return nil
}

// askRename asks for a new name for a paired device.
func (g *gui) askRename(peer trust.Peer) {
	entry := widget.NewEntry()
	entry.SetText(peer.Name)
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.trusted.rename_title"), g.tr.T("gui.trusted.rename_save"), g.tr.T("gui.pair.cancel"),
			entry, func(ok bool) {
				if !ok {
					return
				}
				if err := g.renameDevice(peer, strings.TrimSpace(entry.Text)); err != nil {
					g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
				}
			}, g.win)
	})
}

// pairedHere reports whether the device is still paired here, so that a request
// to pair again from it is answered with an offer to unpair it.
func (g *gui) pairedHere(peer trust.Peer) bool {
	_, ok := g.pairedByKey(peer.Key)
	return ok
}

// askUnpairOnRequest handles a device that asks to pair again while it is still
// paired here: it has unpaired this device on its side. The person decides
// whether to unpair it here too, so that the pairing can be done.
func (g *gui) askUnpairOnRequest(peer trust.Peer) {
	g.onUI(func() {
		dialog.ShowConfirm(g.tr.T("gui.pair.again_title"), g.tr.T("gui.pair.again_body", peer.Name), func(ok bool) {
			if !ok {
				return
			}
			if err := g.unpairDevice(peer); err != nil {
				g.notify(g.tr.T("gui.trusted.title"), g.tr.T(apperr.MessageKey(err)))
				return
			}
			g.notify(g.tr.T("gui.trusted.title"), g.tr.T("gui.pair.again_done"))
		}, g.win)
	})
}

// pairedByKey returns the paired device with this key.
func (g *gui) pairedByKey(key []byte) (trust.Peer, bool) {
	peers, err := g.store.List()
	if err != nil {
		return trust.Peer{}, false
	}
	for _, p := range peers {
		if bytes.Equal(p.Key, key) {
			return p, true
		}
	}
	return trust.Peer{}, false
}

// The last address of each paired device is kept in the application's folder,
// by fingerprint. It is used only to show the paired name of a device that
// discovery finds at that address, before a connection has shown who it is. It
// never marks the device as paired: that still needs an authenticated connection.
const pairedAddressesFile = "paired-addresses.json"

func (g *gui) pairedAddresses() map[string]string {
	data, err := os.ReadFile(filepath.Join(g.home, pairedAddressesFile)) //nolint:gosec // G304: file in the application's folder
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func (g *gui) savePairedAddresses(m map[string]string) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = storage.WriteFileAtomic(filepath.Join(g.home, pairedAddressesFile), data, 0o600)
}

// rememberAddress records the address at which a paired device answered.
func (g *gui) rememberAddress(ep netip.AddrPort, key []byte) {
	fp, err := identity.FingerprintOf(key)
	if err != nil {
		return
	}
	m := g.pairedAddresses()
	m[fp.Hex()] = ep.String()
	g.savePairedAddresses(m)
}

// pairedNameAt returns the paired name of the device last seen at addr, if any.
func (g *gui) pairedNameAt(addr netip.AddrPort) (string, bool) {
	peers, err := g.store.List()
	if err != nil {
		return "", false
	}
	m := g.pairedAddresses()
	for _, p := range peers {
		if m[p.Fingerprint.Hex()] == addr.String() {
			return p.Name, true
		}
	}
	return "", false
}
