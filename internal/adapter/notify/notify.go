// Package notify sends transactional messages.
//
// Every channel sits behind one interface, with at least two implementations
// planned, so swapping a provider is an adapter change and never a reshape of
// the core flow (99 §9). Today: SMTP for email, WAHA for WhatsApp, with the
// Meta Cloud API as the documented swap-in.
package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Channel is one delivery route for a message.
type Channel string

const (
	Email    Channel = "EMAIL"
	WhatsApp Channel = "WHATSAPP"
)

// Message is a rendered notification, ready to send.
type Message struct {
	Channel   Channel
	Recipient string
	Subject   string
	Body      string
	HTML      string
	Template  string
	Locale    string
}

// Sender delivers a message. Implementations must be safe for concurrent use.
type Sender interface {
	Send(ctx context.Context, m Message) error
	Channel() Channel
}

// SMTPConfig is what the mailer needs. It comes from sys_parameters so Steven
// can change the relay without a deploy (D14/B7), with the env as an override.
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
	Timeout   time.Duration
}

// SMTPSender sends email.
type SMTPSender struct{ cfg SMTPConfig }

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Channel() Channel { return Email }

// Send delivers one email.
//
// The headers are built from validated values only. A recipient or subject
// containing CR or LF would let an attacker inject extra headers — a Bcc, a
// different From — so both are rejected rather than escaped.
func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	if err := noHeaderInjection("recipient", m.Recipient); err != nil {
		return err
	}
	if err := noHeaderInjection("subject", m.Subject); err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port))
	from := fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", m.Recipient)
	fmt.Fprintf(&b, "Subject: %s\r\n", m.Subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	if m.HTML != "" {
		boundary := "evermore-boundary-9f3a2b"
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, m.Body)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, m.HTML)
		fmt.Fprintf(&b, "--%s--\r\n", boundary)
	} else {
		fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", m.Body)
	}

	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("notify: dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("notify: smtp: %w", err)
	}
	defer client.Quit()

	if s.cfg.UseTLS {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("notify: starttls: %w", err)
		}
	}
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("notify: auth: %w", err)
		}
	}
	if err := client.Mail(s.cfg.FromEmail); err != nil {
		return fmt.Errorf("notify: from: %w", err)
	}
	if err := client.Rcpt(m.Recipient); err != nil {
		return fmt.Errorf("notify: rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("notify: data: %w", err)
	}
	if _, err := w.Write([]byte(b.String())); err != nil {
		return fmt.Errorf("notify: write: %w", err)
	}
	return w.Close()
}

// noHeaderInjection refuses CR/LF in a value that becomes a mail header.
func noHeaderInjection(field, v string) error {
	if strings.ContainsAny(v, "\r\n") {
		return fmt.Errorf("notify: %s contains a line break and would inject a header", field)
	}
	return nil
}

// WAHAConfig points at the self-hosted WhatsApp gateway (Steven, 2026-08-13).
type WAHAConfig struct {
	BaseURL string
	Session string
	APIKey  string
	Timeout time.Duration
}

// Multi fans a message out to several channels, per the customer's preferences.
type Multi struct {
	senders map[Channel]Sender
}

func NewMulti(senders ...Sender) *Multi {
	m := &Multi{senders: map[Channel]Sender{}}
	for _, s := range senders {
		if s != nil {
			m.senders[s.Channel()] = s
		}
	}
	return m
}

// Send routes a message to its channel.
func (m *Multi) Send(ctx context.Context, msg Message) error {
	s, ok := m.senders[msg.Channel]
	if !ok {
		return fmt.Errorf("notify: no sender for channel %s", msg.Channel)
	}
	return s.Send(ctx, msg)
}

// Has reports whether a channel is configured, so the caller can skip queuing
// a message nobody can deliver.
func (m *Multi) Has(c Channel) bool {
	_, ok := m.senders[c]
	return ok
}
