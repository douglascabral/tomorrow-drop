package main

import "testing"

// The app keeps no history: a finished operation leaves the list, and its
// outcome is shown once, until the person closes it.
func TestFinishedOperationsLeaveNoHistory(t *testing.T) {
	c, err := newCore(t.TempDir(), t.TempDir(), "iPad test", "en")
	if err != nil {
		t.Fatal(err)
	}
	j := c.addJob("running", "Sending files", "")
	if got := len(c.state().Jobs); got != 1 {
		t.Fatalf("running operations = %d, want 1", got)
	}
	c.finish(j, "done", "Sent.")
	st := c.state()
	if len(st.Jobs) != 0 {
		t.Fatalf("a finished operation is still listed: %+v", st.Jobs)
	}
	if st.Result == nil || st.Result.Message != "Sent." {
		t.Fatalf("the outcome is not shown: %+v", st.Result)
	}
	c.mu.Lock()
	c.result = nil
	c.mu.Unlock()
	if c.state().Result != nil {
		t.Fatalf("the outcome stays after it is closed")
	}
}

// A receive that is running is listed, so the person knows not to close the
// app. One that never got an offer leaves no trace at all.
func TestReceiveIsListedWhileRunningAndDroppedWithoutOffer(t *testing.T) {
	c, err := newCore(t.TempDir(), t.TempDir(), "iPad test", "en")
	if err != nil {
		t.Fatal(err)
	}
	j := c.addJob("running", "Receiving files", "")
	if got := len(c.state().Jobs); got != 1 {
		t.Fatalf("a running receive is not listed: %d", got)
	}
	c.dropJob(j)
	st := c.state()
	if len(st.Jobs) != 0 || st.Result != nil {
		t.Fatalf("a dropped operation left a trace: jobs=%+v result=%+v", st.Jobs, st.Result)
	}
}

// Leaving the screen stops the app from being found and cancels what runs. The
// person is told when they come back, and the cancelled operation does not
// overwrite that message when it ends.
func TestLeavingTheScreenInterruptsAndSaysSo(t *testing.T) {
	c, err := newCore(t.TempDir(), t.TempDir(), "iPad test", "en")
	if err != nil {
		t.Fatal(err)
	}
	j := c.addJob("running", "Sending files", "")
	c.setAvailable(false)
	st := c.state()
	if len(st.Jobs) != 0 {
		t.Fatalf("the running operation is still listed after leaving the screen: %+v", st.Jobs)
	}
	if st.Result == nil || st.Result.OK || st.Result.Message == "" {
		t.Fatalf("no interruption message: %+v", st.Result)
	}
	before := *st.Result
	c.finish(j, "failed", "connection lost")
	if got := *c.state().Result; got != before {
		t.Fatalf("the cancelled operation overwrote the message: %+v", got)
	}
}
