package sshproxyapi

import (
	"context"
	"testing"

	"golang.org/x/crypto/ssh"
)

type fakeNewChannel struct {
	channelType   string
	rejectReason  ssh.RejectionReason
	rejectMessage string
	rejected      bool
}

func (f *fakeNewChannel) Accept() (ssh.Channel, <-chan *ssh.Request, error) {
	return nil, nil, nil
}

func (f *fakeNewChannel) Reject(reason ssh.RejectionReason, message string) error {
	f.rejected = true
	f.rejectReason = reason
	f.rejectMessage = message
	return nil
}

func (f *fakeNewChannel) ChannelType() string {
	return f.channelType
}

func (f *fakeNewChannel) ExtraData() []byte {
	return nil
}

func TestHandleProxyChannelDoesNotFinishTransportForUnsupportedChannel(t *testing.T) {
	t.Parallel()

	channel := &fakeNewChannel{channelType: "direct-tcpip"}
	handled := handleProxyChannel(context.Background(), channel, nil, nil)

	if handled {
		t.Fatal("unsupported channel marked the proxy session as handled")
	}
	if !channel.rejected {
		t.Fatal("unsupported channel was not rejected")
	}
	if channel.rejectReason != ssh.UnknownChannelType {
		t.Fatalf("reject reason = %v; want %v", channel.rejectReason, ssh.UnknownChannelType)
	}
}

func TestProxyStartBlockedWhilePaused(t *testing.T) {
	t.Parallel()

	control := newProxySessionControl(context.Background(), "sess-1", nil, nil)
	control.setPaused(true)

	if !proxyStartBlocked(false, control) {
		t.Fatal("start request was allowed while the proxy session was paused")
	}
	if !proxyStartBlocked(true, nil) {
		t.Fatal("second start request was allowed")
	}
}
