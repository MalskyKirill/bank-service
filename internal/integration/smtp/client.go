package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	mail "github.com/go-mail/mail/v2"
	"github.com/sirupsen/logrus"
)

type Client struct {
	enabled  bool
	host     string
	port     int
	username string
	password string
	from     string
	logger   *logrus.Logger
}

func NewClient(
	enabled bool,
	host string,
	port int,
	username string,
	password string,
	from string,
	logger *logrus.Logger,
) *Client {
	if port == 0 {
		port = 587
	}

	if strings.TrimSpace(from) == "" {
		from = username
	}

	return &Client{
		enabled:  enabled,
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		logger:   logger,
	}
}

func (c *Client) SendEmail(ctx context.Context, to string, subject string, htmlBody string) error {
	if !c.enabled {
		if c.logger != nil {
			c.logger.Infof("SMTP is disabled, email to %s skipped", to)
		}

		return nil
	}

	if strings.TrimSpace(c.host) == "" {
		return fmt.Errorf("smtp host is required")
	}

	if strings.TrimSpace(c.username) == "" {
		return fmt.Errorf("smtp username is required")
	}

	if strings.TrimSpace(c.password) == "" {
		return fmt.Errorf("smtp password is required")
	}

	if strings.TrimSpace(c.from) == "" {
		return fmt.Errorf("smtp from is required")
	}

	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("email recipient is required")
	}

	message := mail.NewMessage()
	message.SetHeader("From", c.from)
	message.SetHeader("To", to)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", htmlBody)

	dialer := mail.NewDialer(c.host, c.port, c.username, c.password)
	dialer.TLSConfig = &tls.Config{
		ServerName: c.host,
		MinVersion: tls.VersionTLS12,
	}

	done := make(chan error, 1)

	go func() {
		done <- dialer.DialAndSend(message)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	if c.logger != nil {
		c.logger.Infof("email sent to %s", to)
	}

	return nil
}
