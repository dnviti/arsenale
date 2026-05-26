package tunnelbroker

import (
	"testing"
	"time"
)

func TestTargetPortCandidatesDeduplicateAndDropInvalidPorts(t *testing.T) {
	got := targetPortCandidates(14822, []int{4822, 14822, 0, -1, 70000})
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

func TestTargetPortCandidatesLimitFallbackAttempts(t *testing.T) {
	got := targetPortCandidates(14822, []int{4822, 5822, 6822})
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

func TestIsValidTargetPortRequest(t *testing.T) {
	tests := []struct {
		name       string
		primary    int
		additional []int
		want       bool
	}{
		{name: "single valid primary", primary: 22, want: true},
		{name: "valid fallback", primary: 22, additional: []int{2222}, want: true},
		{name: "too many fallbacks", primary: 22, additional: []int{2222, 2223}, want: false},
		{name: "out of range primary", primary: 70000, want: false},
		{name: "all invalid", primary: 0, additional: []int{70000}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTargetPortRequest(tt.primary, tt.additional); got != tt.want {
				t.Fatalf("isValidTargetPortRequest(%d, %#v) = %t, want %t", tt.primary, tt.additional, got, tt.want)
			}
		})
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
