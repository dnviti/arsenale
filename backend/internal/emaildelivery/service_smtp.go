package emaildelivery

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func sendSMTP(msg Message) error {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := emailFrom()
	addr := host + ":" + port

	raw, err := buildSMTPMessage(from, msg)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := loadSecretEnv("SMTP_PASS", "SMTP_PASS_FILE")
	var auth smtp.Auth
	if user != "" || pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	if port == "465" {
		return sendImplicitTLSSMTP(addr, host, auth, from, msg.To, raw)
	}
	return smtp.SendMail(addr, auth, from, []string{msg.To}, raw)
}

func sendImplicitTLSSMTP(addr, host string, auth smtp.Auth, from, to string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("dial smtp tls: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp finalize body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func buildSMTPMessage(from string, msg Message) ([]byte, error) {
	text := msg.Text
	if text == "" {
		text = msg.Subject
	}

	var body string
	headers := []string{
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
	}

	switch {
	case msg.HTML != "" && text != "":
		boundary := "arsenale-email-boundary"
		headers = append(headers, `Content-Type: multipart/alternative; boundary="`+boundary+`"`)
		body = "--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + text + "\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n\r\n" + msg.HTML + "\r\n" +
			"--" + boundary + "--\r\n"
	case msg.HTML != "":
		headers = append(headers, "Content-Type: text/html; charset=UTF-8")
		body = msg.HTML
	default:
		headers = append(headers, "Content-Type: text/plain; charset=UTF-8")
		body = text
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body), nil
}
