// Package notify sends transactional messages.
//
// Every channel sits behind one interface, with at least two implementations
// planned, so swapping a provider is an adapter change and never a reshape of
// the core flow (99 §9). Today: SMTP for email, WAHA for WhatsApp, with the
// Meta Cloud API as the documented swap-in.
package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
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

// WAHA sends WhatsApp messages through a self-hosted WAHA gateway.
//
// Unofficial channel: the number can be banned by WhatsApp, and the documented
// alternative is the Meta Cloud API, which swaps this type and nothing else
// (docs/02 D-11). The gateway on this host is SHARED with ruuma.
type WAHA struct {
	cfg    WAHAConfig
	client *http.Client
}

// NewWAHA builds the sender, or returns nil when it is not configured.
//
// nil rather than an error on purpose: Multi skips a channel it has no sender
// for, so an unconfigured gateway means WhatsApp messages are never queued
// rather than queued and permanently failing.
func NewWAHA(cfg WAHAConfig) *WAHA {
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return nil
	}
	if cfg.Session == "" {
		cfg.Session = "default"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &WAHA{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

func (w *WAHA) Channel() Channel { return WhatsApp }

// Send posts one text message.
func (w *WAHA) Send(ctx context.Context, m Message) error {
	to, err := waChatID(m.Recipient)
	if err != nil {
		return err
	}

	// WAHA takes the whole message as one text field, so the HTML body is not
	// used here — the plain-text body is the WhatsApp message.
	payload, err := json.Marshal(map[string]string{
		"session": w.cfg.Session,
		"chatId":  to,
		"text":    strings.TrimSpace(m.Body),
	})
	if err != nil {
		return fmt.Errorf("notify: waha encode: %w", err)
	}

	endpoint := strings.TrimRight(w.cfg.BaseURL, "/") + "/api/sendText"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("notify: waha request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", w.cfg.APIKey)

	res, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify: waha send: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return nil
	}
	// The body carries WAHA's reason — a dead session, an unknown chat id — and
	// the job queue records it, so bounded is enough to be useful in a log
	// without pasting an unbounded remote response into it.
	reason, _ := io.ReadAll(io.LimitReader(res.Body, 512))
	return fmt.Errorf("notify: waha refused with %d: %s",
		res.StatusCode, strings.TrimSpace(string(reason)))
}

// waChatID turns a stored phone number into WAHA's chat identifier.
//
// sanitize.Phone is the single definition of a valid Indonesian number, so the
// link, the stored contact and the message all agree — a "@c.us" suffix built
// from a hand-trimmed zero is how a message goes to the wrong person.
func waChatID(recipient string) (string, error) {
	normalised, err := sanitize.Phone("recipient", recipient)
	if err != nil {
		return "", fmt.Errorf("notify: waha recipient %q is not a usable number", recipient)
	}
	return strings.TrimPrefix(normalised, "+") + "@c.us", nil
}
