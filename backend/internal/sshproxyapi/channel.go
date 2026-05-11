package sshproxyapi

import (
	"context"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

type proxyStreamKind int

const (
	proxyStreamInput proxyStreamKind = iota
	proxyStreamOutput
)

func handleProxyChannel(ctx context.Context, newChannel ssh.NewChannel, target *ssh.Client, control *proxySessionControl) bool {
	if newChannel.ChannelType() != "session" {
		_ = newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
		return false
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		return true
	}
	defer channel.Close()

	targetSession, err := target.NewSession()
	if err != nil {
		_ = channel.Close()
		return true
	}
	defer targetSession.Close()
	unregisterActive := control.registerActiveSSH(channel, targetSession)
	defer unregisterActive()

	stdin, err := targetSession.StdinPipe()
	if err != nil {
		return true
	}
	stdout, err := targetSession.StdoutPipe()
	if err != nil {
		return true
	}
	stderr, err := targetSession.StderrPipe()
	if err != nil {
		return true
	}

	go func() {
		copyProxyStream(ctx, stdin, channel, control, proxyStreamInput)
		_ = stdin.Close()
	}()
	go func() { copyProxyStream(ctx, channel, stdout, control, proxyStreamOutput) }()
	go func() { copyProxyStream(ctx, channel.Stderr(), stderr, control, proxyStreamOutput) }()

	var started sync.Once
	startedFlag := false
	startAndWait := func(start func() error, req *ssh.Request) {
		ok := false
		started.Do(func() {
			if err := start(); err == nil {
				startedFlag = true
				ok = true
				go waitForTargetSession(targetSession, channel)
			}
		})
		_ = req.Reply(ok, nil)
	}

	for req := range requests {
		switch req.Type {
		case "env":
			ok := handleEnvRequest(targetSession, req.Payload)
			_ = req.Reply(ok, nil)
		case "pty-req":
			ok := handlePTYRequest(targetSession, req.Payload)
			_ = req.Reply(ok, nil)
		case "window-change":
			if control != nil && control.isPaused() {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				continue
			}
			ok := handleWindowChange(targetSession, req.Payload)
			if req.WantReply {
				_ = req.Reply(ok, nil)
			}
		case "shell":
			if proxyStartBlocked(startedFlag, control) {
				_ = req.Reply(false, nil)
				continue
			}
			startAndWait(targetSession.Shell, req)
		case "exec":
			if proxyStartBlocked(startedFlag, control) {
				_ = req.Reply(false, nil)
				continue
			}
			command := parseExecCommand(req.Payload)
			startAndWait(func() error { return targetSession.Start(command) }, req)
		case "signal":
			if control != nil && control.isPaused() {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				continue
			}
			ok := handleSignalRequest(targetSession, req.Payload)
			if req.WantReply {
				_ = req.Reply(ok, nil)
			}
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
	return true
}

func proxyStartBlocked(started bool, control *proxySessionControl) bool {
	return started || (control != nil && control.isPaused())
}

func copyProxyStream(ctx context.Context, dst io.Writer, src io.Reader, control *proxySessionControl, kind proxyStreamKind) {
	buffer := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := src.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if kind == proxyStreamInput && control != nil && control.isPaused() {
				if err != nil {
					return
				}
				continue
			}
			if kind == proxyStreamOutput && control != nil && !control.waitUntilResumed() {
				return
			}
			if _, writeErr := dst.Write(chunk); writeErr != nil {
				return
			}
			if kind == proxyStreamOutput && control != nil {
				control.observeOutput(chunk)
			}
		}
		if err != nil {
			return
		}
	}
}

func handleEnvRequest(session *ssh.Session, payload []byte) bool {
	var req struct {
		Name  string
		Value string
	}
	if err := ssh.Unmarshal(payload, &req); err != nil {
		return false
	}
	return session.Setenv(req.Name, req.Value) == nil
}

func handlePTYRequest(session *ssh.Session, payload []byte) bool {
	var req struct {
		Term         string
		Columns      uint32
		Rows         uint32
		WidthPixels  uint32
		HeightPixels uint32
		Modes        string
	}
	if err := ssh.Unmarshal(payload, &req); err != nil {
		return false
	}
	cols := int(req.Columns)
	rows := int(req.Rows)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return session.RequestPty(req.Term, rows, cols, ssh.TerminalModes{}) == nil
}

func handleWindowChange(session *ssh.Session, payload []byte) bool {
	var req struct {
		Columns      uint32
		Rows         uint32
		WidthPixels  uint32
		HeightPixels uint32
	}
	if err := ssh.Unmarshal(payload, &req); err != nil {
		return false
	}
	return session.WindowChange(int(req.Rows), int(req.Columns)) == nil
}

func handleSignalRequest(session *ssh.Session, payload []byte) bool {
	var req struct {
		Signal string
	}
	if err := ssh.Unmarshal(payload, &req); err != nil {
		return false
	}
	return session.Signal(ssh.Signal(req.Signal)) == nil
}

func parseExecCommand(payload []byte) string {
	var req struct {
		Command string
	}
	_ = ssh.Unmarshal(payload, &req)
	return req.Command
}

func waitForTargetSession(session *ssh.Session, channel ssh.Channel) {
	err := session.Wait()
	status := uint32(0)
	if err != nil {
		status = 1
		if exitErr, ok := err.(*ssh.ExitError); ok {
			status = uint32(exitErr.ExitStatus())
		}
	}
	_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct {
		Status uint32
	}{Status: status}))
	_ = channel.Close()
}
