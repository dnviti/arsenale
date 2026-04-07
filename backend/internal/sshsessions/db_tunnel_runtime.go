package sshsessions

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func (t *activeDBTunnel) serve() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.setForwardError(err)
			return
		}
		t.touch()
		go t.forward(conn)
	}
}

func (t *activeDBTunnel) forward(localConn net.Conn) {
	targetAddr := net.JoinHostPort(t.TargetDBHost, strconv.Itoa(t.TargetDBPort))
	remoteConn, err := t.sshClient.Dial("tcp", targetAddr)
	if err != nil {
		t.setForwardError(err)
		_ = localConn.Close()
		return
	}

	go func() {
		_, _ = io.Copy(remoteConn, localConn)
		_ = remoteConn.Close()
	}()
	go func() {
		_, _ = io.Copy(localConn, remoteConn)
		_ = localConn.Close()
	}()
}

func dbTunnelSSHClientConfig(credentials resolvedCredentials) (*ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0, 2)
	if strings.TrimSpace(credentials.Password) != "" {
		authMethods = append(authMethods, ssh.Password(credentials.Password))
	}
	if strings.TrimSpace(credentials.PrivateKey) != "" {
		var (
			signer ssh.Signer
			err    error
		)
		if strings.TrimSpace(credentials.Passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(credentials.PrivateKey), []byte(credentials.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(credentials.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key for %s: %w", credentials.Username, err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return nil, errors.New("ssh credentials are required")
	}

	return &ssh.ClientConfig{
		User:            credentials.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}, nil
}

func buildDBTunnelConnectionString(dbType *string, host string, port int, username, password, dbName string) *string {
	host = strings.TrimSpace(host)
	if host == "" || port <= 0 {
		return nil
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))

	var value string
	userPass := ""
	if strings.TrimSpace(username) != "" && strings.TrimSpace(password) != "" {
		userPass = urlEncode(username) + ":" + urlEncode(password) + "@"
	} else if strings.TrimSpace(username) != "" {
		userPass = urlEncode(username) + "@"
	}
	db := ""
	if strings.TrimSpace(dbName) != "" {
		db = "/" + urlEncode(dbName)
	}

	switch normalized := strings.ToLower(strings.TrimSpace(valueOrEmpty(dbType))); normalized {
	case "postgresql", "postgres":
		value = "postgresql://" + userPass + address + db
	case "mysql", "mariadb":
		value = "mysql://" + userPass + address + db
	case "mongodb", "mongo":
		value = "mongodb://" + userPass + address + db
	case "redis":
		if strings.TrimSpace(password) != "" {
			value = "redis://:" + urlEncode(password) + "@" + address
		} else {
			value = "redis://" + address
		}
	case "mssql", "sqlserver":
		var builder strings.Builder
		builder.WriteString("Server=")
		builder.WriteString(host)
		builder.WriteString(",")
		builder.WriteString(strconv.Itoa(port))
		builder.WriteString(";")
		if strings.TrimSpace(dbName) != "" {
			builder.WriteString("Database=")
			builder.WriteString(dbName)
			builder.WriteString(";")
		}
		if strings.TrimSpace(username) != "" {
			builder.WriteString("User Id=")
			builder.WriteString(username)
			builder.WriteString(";")
		}
		if strings.TrimSpace(password) != "" {
			builder.WriteString("Password=")
			builder.WriteString(password)
			builder.WriteString(";")
		}
		value = builder.String()
	case "oracle":
		base := address
		if strings.TrimSpace(dbName) != "" {
			value = base + "/" + strings.TrimSpace(dbName)
		} else {
			value = base + "/ORCL"
		}
	default:
		value = address
	}

	return &value
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyString(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			trimmed := strings.TrimSpace(*value)
			return &trimmed
		}
	}
	return nil
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func urlEncode(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"!", "%21",
		"#", "%23",
		"$", "%24",
		"&", "%26",
		"'", "%27",
		"(", "%28",
		")", "%29",
		"*", "%2A",
		"+", "%2B",
		",", "%2C",
		"/", "%2F",
		":", "%3A",
		";", "%3B",
		"=", "%3D",
		"?", "%3F",
		"@", "%40",
		"[", "%5B",
		"]", "%5D",
	)
	return replacer.Replace(value)
}
