package tunnelbroker

import (
	"testing"
	"time"
)

func TestTargetPortCandidatesDeduplicateAndDropInvalidPorts(t *testing.T) {
	got := targetPortCandidates(14822, []int{4822, 14822, 0, -1})
	want := []int{14822, 4822}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d = %d, want %d: %#v", i, got[i], want[i], got)
		}
	}
}

func TestHandleCloseResolvesPendingOpen(t *testing.T) {
	broker := NewBroker(BrokerConfig{})
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	resolve := make(chan *streamConn, 1)
	conn := &tunnelConnection{
		broker:       broker,
		streams:      make(map[uint16]*streamConn),
		pendingOpens: map[uint16]*pendingOpen{7: &pendingOpen{resolve: resolve, timer: timer}},
	}

	broker.handleClose(conn, 7)

	select {
	case stream := <-resolve:
		if stream != nil {
			t.Fatalf("pending open resolved with stream %#v, want nil", stream)
		}
	case <-time.After(time.Second):
		t.Fatal("pending open was not resolved")
	}
	if _, ok := conn.pendingOpens[7]; ok {
		t.Fatal("pending open was not removed")
	}
}
