package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/notify"
	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Notifier queues and delivers transactional messages.
//
// Sends go through the job table, not straight out of the request: an SMTP
// timeout must not fail a payment that already succeeded, and support has to be
// able to prove what was sent (PROMPT §11). The queue lives in Postgres so a
// message is enqueued in the SAME transaction as the state change that caused
// it — a confirmation email for an order that rolled back is worse than none.
type Notifier struct {
	jobs    *postgres.JobRepo
	multi   *notify.Multi
	tpl     *notify.Templates
	params  *sysparam.Store
	log     *slog.Logger
	tz      *time.Location
	baseURL string
}

// NotifierDeps wires the service.
type NotifierDeps struct {
	Jobs    *postgres.JobRepo
	Senders *notify.Multi
	Params  *sysparam.Store
	Log     *slog.Logger
	TZ      *time.Location
	BaseURL string
}

func NewNotifier(d NotifierDeps) *Notifier {
	return &Notifier{
		jobs: d.Jobs, multi: d.Senders, tpl: notify.NewTemplates(),
		params: d.Params, log: d.Log, tz: d.TZ, baseURL: d.BaseURL,
	}
}

// Queue enqueues one notification.
//
// dedupe makes a job idempotent: the expiry sweep runs hourly and must not send
// the same warning six times, so the same key collapses to one pending job.
func (n *Notifier) Queue(ctx context.Context, template, recipient, locale string,
	data map[string]any, dedupe string, refType string, refID *uuid.UUID) error {

	payload, err := json.Marshal(map[string]any{
		"template": template, "recipient": recipient, "locale": locale,
		"data": data, "reference_type": refType, "reference_id": refID,
	})
	if err != nil {
		return fmt.Errorf("notifier: payload: %w", err)
	}
	return n.jobs.Enqueue(ctx, "notification", payload, dedupe, time.Now())
}

// Run processes the queue until the context is cancelled.
func (n *Notifier) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := n.drain(ctx); err != nil {
				n.log.Error("notification worker", "error", err)
			}
		}
	}
}

// drain claims and delivers pending jobs.
func (n *Notifier) drain(ctx context.Context) error {
	jobs, err := n.jobs.Claim(ctx, "notification", 20)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err := n.deliver(ctx, j); err != nil {
			// Retried with a backoff; after max_attempts the job is FAILED and
			// stays visible rather than disappearing.
			n.log.Warn("notification failed, will retry",
				"job", j.ID, "attempts", j.Attempts, "error", err)
			_ = n.jobs.Fail(ctx, j.ID, err.Error())
			continue
		}
		_ = n.jobs.Done(ctx, j.ID)
	}
	return nil
}

func (n *Notifier) deliver(ctx context.Context, j postgres.Job) error {
	var p struct {
		Template      string         `json:"template"`
		Recipient     string         `json:"recipient"`
		Locale        string         `json:"locale"`
		Data          map[string]any `json:"data"`
		ReferenceType string         `json:"reference_type"`
		ReferenceID   *uuid.UUID     `json:"reference_id"`
	}
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return fmt.Errorf("notifier: bad payload: %w", err)
	}

	if !n.params.Bool(ctx, sysparam.KeyEmailEnabled, true) {
		n.log.Info("email disabled by settings; dropping", "template", p.Template)
		return nil
	}

	msg, err := n.tpl.Render(p.Template, p.Locale, p.Data)
	if err != nil {
		return err
	}
	msg.Channel = notify.Email
	msg.Recipient = p.Recipient

	logID, err := n.jobs.LogNotification(ctx, postgres.NotificationLog{
		Channel: string(notify.Email), Template: p.Template, Recipient: p.Recipient,
		Subject: msg.Subject, Locale: msg.Locale,
		ReferenceType: p.ReferenceType, ReferenceID: p.ReferenceID,
	})
	if err != nil {
		return err
	}

	if err := n.multi.Send(ctx, msg); err != nil {
		_ = n.jobs.MarkNotification(ctx, logID, "FAILED", err.Error())
		return err
	}
	return n.jobs.MarkNotification(ctx, logID, "SENT", "")
}

// ── The nine transactional messages (PROMPT §11) ────────────────────────────

// VerifyEmail sends the confirmation link.
func (n *Notifier) VerifyEmail(ctx context.Context, email, name, token, locale string) {
	n.queueQuiet(ctx, notify.TplVerifyEmail, email, locale, map[string]any{
		"Name": name,
		"URL":  n.baseURL + "/verify-email?token=" + token,
	}, "verify:"+token, "app_user", nil)
}

// OrderPlaced sends the payment instructions.
func (n *Notifier) OrderPlaced(ctx context.Context, email, name, locale string,
	orderID uuid.UUID, data map[string]any) {
	n.queueQuiet(ctx, notify.TplOrderPlaced, email, locale, data,
		"order-placed:"+orderID.String(), "customer_order", &orderID)
}

// PaymentVerified confirms the money arrived.
func (n *Notifier) PaymentVerified(ctx context.Context, email, name, locale string,
	orderID uuid.UUID, orderCode, extra string) {
	n.queueQuiet(ctx, notify.TplPaymentVerified, email, locale, map[string]any{
		"Name": name, "OrderCode": orderCode, "Extra": extra,
	}, "payment-verified:"+orderID.String(), "customer_order", &orderID)
}

// PaymentRejected explains what to fix.
func (n *Notifier) PaymentRejected(ctx context.Context, email, name, locale string,
	orderID uuid.UUID, orderCode, reason, deadline string) {
	n.queueQuiet(ctx, notify.TplPaymentRejected, email, locale, map[string]any{
		"Name": name, "OrderCode": orderCode, "Reason": reason, "Deadline": deadline,
	}, "", "customer_order", &orderID)
}

// queueQuiet enqueues and logs a failure rather than propagating it.
//
// A notification that cannot be QUEUED must not fail the business action that
// triggered it — the order is placed, the payment is verified, and the customer
// being told is a separate concern.
func (n *Notifier) queueQuiet(ctx context.Context, template, recipient, locale string,
	data map[string]any, dedupe, refType string, refID *uuid.UUID) {
	if recipient == "" {
		return
	}
	if err := n.Queue(ctx, template, recipient, locale, data, dedupe, refType, refID); err != nil {
		n.log.Error("could not queue notification",
			"template", template, "error", err)
	}
}
