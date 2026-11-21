package main

import (
	"testing"
)

// Regression: after sending a file the completion screen offered "Receive
// again", although the person had sent it. The receive action belongs only to
// a completed reception.
func TestSentCompletionOffersNoReceiveAction(t *testing.T) {
	g := newTestGUI(t)
	g.showCompleted(g.tr.T("gui.send.done"), g.tr.T("gui.send.sent_to", "laptop"), nil)
	if contains(buttonLabels(g.win.Content()), g.tr.T("gui.done.again")) {
		t.Fatalf("a sent file offers receive again; buttons = %v", buttonLabels(g.win.Content()))
	}
}

func TestReceivedCompletionOffersReceiveAgain(t *testing.T) {
	g := newTestGUI(t)
	g.showCompleted(g.tr.T("gui.receive.done"), "1 file", []string{"a.pdf"}, g.receiveAgainAction())
	if !contains(buttonLabels(g.win.Content()), g.tr.T("gui.done.again")) {
		t.Fatalf("a received file lacks receive again; buttons = %v", buttonLabels(g.win.Content()))
	}
}

// A sent file offers to send again: the button opens the send screen.
func TestSentCompletionOffersSendAgain(t *testing.T) {
	g := newTestGUI(t)
	g.showCompleted(g.tr.T("gui.send.done"), g.tr.T("gui.send.sent_to", "laptop"), nil,
		doneAction{label: g.tr.T("gui.done.send_again"), run: func() { g.showSend("", false) }})
	button := findButton(g.win.Content(), g.tr.T("gui.done.send_again"))
	if button == nil {
		t.Fatalf("sent file lacks send again; buttons = %v", buttonLabels(g.win.Content()))
	}
	button.OnTapped()
	if !contains(buttonLabels(g.win.Content()), g.tr.T("gui.send.clipboard")) {
		t.Fatalf("send again did not open the send screen; texts = %v", labelTexts(g.win.Content()))
	}
}
