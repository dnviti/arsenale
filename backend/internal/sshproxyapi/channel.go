package sshproxyapi

import (
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

func handleProxyChannel(newChannel ssh.NewChannel, target *ssh.Client) {
	if newChannel.ChannelType() != "session" {
		_ = newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		return
	}
	defer channel.Close()

	targetSession, err := target.NewSession()
	if err != nil {
		_ = channel.Close()
		return
	}
	defer targetSession.Close()

	stdin, err := targetSession.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := targetSession.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := targetSession.StderrPipe()
	if err != nil {
		return
	}

	go func() {
		_, _ = io.Copy(stdin, channel)
		_ = stdin.Close()
	}()
	go func() { _, _ = io.Copy(channel, stdout) }()
	go func() { _, _ = io.Copy(channel.Stderr(), stderr) }()

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
			ok := handleWindowChange(targetSession, req.Payload)
			if req.WantReply {
				_ = req.Reply(ok, nil)
			}
		case "shell":
			if startedFlag {
				_ = req.Reply(false, nil)
				continue
			}
			startAndWait(targetSession.Shell, req)
		case "exec":
			if startedFlag {
				_ = req.Reply(false, nil)
				continue
			}
			command := parseExecCommand(req.Payload)
			startAndWait(func() error { return targetSession.Start(command) }, req)
		case "signal":
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
