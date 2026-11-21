package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"net/netip"
	"time"
	"unsafe"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/transport"
)

// theCore is the application state, created by tdStart.
var theCore *core

func goString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

// cString returns a C string the caller frees with tdFree.
func cString(s string) *C.char {
	return C.CString(s)
}

// tdStart opens the application. home holds the keys, the trusted devices and
// the staging files; docs is where received files are saved; name is the name
// announced to other devices; lang is the language. It returns an empty string
// on success, or the error to show.
//
//export tdStart
func tdStart(home, docs, name, lang *C.char) *C.char {
	c, err := newCore(goString(home), goString(docs), goString(name), goString(lang))
	if err != nil {
		return cString(err.Error())
	}
	theCore = c
	return cString("")
}

// tdSetAvailable turns discovery and the listener on (1) or off (0). It is called
// when the app comes to the foreground or goes to the background.
//
//export tdSetAvailable
func tdSetAvailable(on C.int) {
	if theCore != nil {
		theCore.setAvailable(on != 0)
	}
}

// tdState returns the state as JSON. The caller frees it with tdFree.
//
//export tdState
func tdState() *C.char {
	if theCore == nil {
		return cString("{}")
	}
	return cString(theCore.toJSON())
}

// tdPairStart starts pairing with the device at addr (IP:port). The question
// appears in the state as "pair".
//
//export tdPairStart
func tdPairStart(addr *C.char) {
	c := theCore
	if c == nil {
		return
	}
	a := goString(addr)
	ep, err := netip.ParseAddrPort(a)
	if err != nil {
		c.addJob("failed", c.tr.T("ios.pair.title"), c.tr.T(apperr.MessageKey(apperr.ErrMalformed)))
		return
	}
	suggested := c.nameAt(ep)
	go func() {
		ctx, cancel := contextWithTimeout(c.ctxOrBackground(), 2*time.Minute)
		defer cancel()
		s, err := transport.Dial(ctx, ep, c.id, c.store)
		if err != nil {
			c.addJob("failed", c.tr.T("ios.pair.title"), c.tr.T(apperr.MessageKey(err)))
			return
		}
		defer func() { _ = s.Close() }()
		if s.State() == protocol.StateTrusted {
			c.mu.Lock()
			c.rememberLocked(s.PeerKey(), ep)
			c.mu.Unlock()
			c.addJob("done", c.tr.T("ios.pair.title"), c.tr.T("ios.pair.already"))
			return
		}
		_ = c.runPairing(ctx, s, ep, false, suggested) //nolint:errcheck // the outcome is shown in the activity list
	}()
}

// tdPairConfirm answers the comparison of the fingerprints: 1 when they match.
//
//export tdPairConfirm
func tdPairConfirm(yes C.int) {
	c := theCore
	if c == nil {
		return
	}
	c.mu.Lock()
	p := c.pair
	c.mu.Unlock()
	if p != nil {
		select {
		case p.confirm <- yes != 0:
		default:
		}
	}
}

// tdPairName gives the name of the device that was paired.
//
//export tdPairName
func tdPairName(name *C.char) {
	c := theCore
	if c == nil {
		return
	}
	c.mu.Lock()
	p := c.pair
	c.mu.Unlock()
	if p != nil {
		select {
		case p.name <- goString(name):
		default:
		}
	}
}

// tdOfferAnswer accepts (1) or declines (0) the offer waiting for the person.
//
//export tdOfferAnswer
func tdOfferAnswer(yes C.int) {
	c := theCore
	if c == nil {
		return
	}
	c.mu.Lock()
	o := c.offer
	c.mu.Unlock()
	if o != nil {
		select {
		case o.answer <- yes != 0:
		default:
		}
	}
}

// tdSendFiles sends the files at the paths (a JSON array) to the device at addr.
// It returns the job id, shown in the state.
//
//export tdSendFiles
func tdSendFiles(addr, pathsJSON *C.char) C.int {
	c := theCore
	if c == nil {
		return 0
	}
	var paths []string
	if err := json.Unmarshal([]byte(goString(pathsJSON)), &paths); err != nil || len(paths) == 0 {
		return 0
	}
	j := c.addJob("running", c.tr.T("ios.send.files_title"), "")
	spec := transfer.SendSpec{Kind: transfer.KindFile, Paths: paths}
	go c.send(goString(addr), spec, j)
	return C.int(j.id)
}

// tdSendText sends a text to the device at addr. It returns the job id.
//
//export tdSendText
func tdSendText(addr, text *C.char) C.int {
	c := theCore
	if c == nil {
		return 0
	}
	j := c.addJob("running", c.tr.T("ios.send.text_title"), "")
	spec := transfer.SendSpec{Kind: transfer.KindText, Text: goString(text)}
	go c.send(goString(addr), spec, j)
	return C.int(j.id)
}

// tdRename gives a paired device a new name. fingerprintHex is its fingerprint.
//
//export tdRename
func tdRename(fingerprintHex, name *C.char) C.int {
	c := theCore
	if c == nil {
		return 1
	}
	peer, ok := c.pairedByHex(goString(fingerprintHex))
	if !ok {
		return 1
	}
	if err := c.store.Rename(peer.Fingerprint, goString(name)); err != nil {
		return 1
	}
	return 0
}

// tdUnpair stops trusting a paired device. fingerprintHex is its fingerprint.
//
//export tdUnpair
func tdUnpair(fingerprintHex *C.char) C.int {
	c := theCore
	if c == nil {
		return 1
	}
	peer, ok := c.pairedByHex(goString(fingerprintHex))
	if !ok {
		return 1
	}
	if err := c.unpair(peer); err != nil {
		return 1
	}
	return 0
}

// tdSearch asks the network for devices at once (the Refresh button).
//
//export tdSearch
func tdSearch() {
	if theCore != nil {
		theCore.searchNow()
	}
}

// tdDismissResult closes the outcome of the last operation.
//
//export tdDismissResult
func tdDismissResult() {
	if theCore != nil {
		theCore.mu.Lock()
		theCore.result = nil
		theCore.mu.Unlock()
	}
}

// tdDismissReceived closes the text that was received.
//
//export tdDismissReceived
func tdDismissReceived() {
	if theCore != nil {
		theCore.mu.Lock()
		theCore.received = nil
		theCore.mu.Unlock()
	}
}

// tdFree frees a string returned by this library.
//
//export tdFree
func tdFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}
