package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"time"
)

// Client sends mail via SMTP (Mailcow prod or Mailpit dev).
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

func (c *Client) Send(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", c.from, to, subject, body))

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

	if c.user != "" {
		if err := client.Auth(smtp.PlainAuth("", c.user, c.pass, c.host)); err != nil {
			return err
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
