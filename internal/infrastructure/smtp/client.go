package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"time"
)

// Client sends mail via SMTP (Stalwart prod or Mailpit dev).
type Client struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewFromEnv() (*Client, error) {
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		return nil, fmt.Errorf("SMTP_FROM is required")
	}
	return &Client{
		host: envOr("SMTP_HOST", "localhost"),
		port: envOr("SMTP_PORT", "10725"),
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASS"),
		from: from,
	}, nil
}

func (c *Client) Send(ctx context.Context, to, subject, text, html string) error {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)
	msg := BuildMIME(c.from, to, subject, text, html)

	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()

	// Submission servers (Stalwart on 587) refuse AUTH before STARTTLS. Upgrade
	// when the server offers it; Mailpit in dev does not, so the check keeps the
	// plain path working.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: c.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	// Only authenticate when the server advertises AUTH. Mailpit (tests and dev)
	// accepts mail without it; calling Auth anyway fails with "command not
	// implemented".
	if c.user != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", c.user, c.pass, c.host)); err != nil {
				return err
			}
		}
	}
	if err := client.Mail(c.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
