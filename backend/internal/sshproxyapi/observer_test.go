package sshproxyapi

import (
	"context"
	"testing"
	"time"
)

func TestProxyObserverHubReplaysRecentOutputAndBroadcastsClose(t *testing.T) {
	t.Parallel()

	hub := newProxyObserverHub()
	observed := hub.register("sess-1")
	observed.broadcastData([]byte("before\n"))

	snapshot, events, unsubscribe, ok := observed.subscribe()
	defer unsubscribe()
	if !ok {
		t.Fatal("subscribe() = false, want true")
	}
	if snapshot != "before\n" {
		t.Fatalf("snapshot = %q, want prior output", snapshot)
	}

	observed.broadcastData([]byte("after\n"))
	assertProxyObserverEvent(t, events, proxyObserverEvent{typ: "data", data: "after\n"})

	observed.close("SESSION_TERMINATED", "Session terminated by administrator")
	assertProxyObserverEvent(t, events, proxyObserverEvent{typ: "error", code: "SESSION_TERMINATED", message: "Session terminated by administrator"})

	if _, ok := hub.get("sess-1"); ok {
		t.Fatal("closed session remained observable")
	}
}

func TestProxySessionControlWaitUntilResumedStopsOnTerminate(t *testing.T) {
	t.Parallel()

	control := newProxySessionControl(context.Background(), "sess-1", nil, nil)
	control.setPaused(true)

	done := make(chan bool, 1)
	go func() {
		done <- control.waitUntilResumed()
	}()

	select {
	case resumed := <-done:
		t.Fatalf("waitUntilResumed() returned early with %v", resumed)
	case <-time.After(50 * time.Millisecond):
	}

	control.terminate("SESSION_TERMINATED", "terminated")
	select {
	case resumed := <-done:
		if resumed {
			t.Fatal("waitUntilResumed() = true after terminate, want false")
		}
	case <-time.After(time.Second):
		t.Fatal("waitUntilResumed() did not stop after terminate")
	}
}

func assertProxyObserverEvent(t *testing.T, events <-chan proxyObserverEvent, want proxyObserverEvent) {
	t.Helper()
	select {
	case got := <-events:
		if got != want {
			t.Fatalf("event = %#v, want %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for event %#v", want)
	}
}
