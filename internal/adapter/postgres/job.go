package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobRepo is the Postgres-backed queue and the notification log.
//
// Postgres rather than Redis for the queue (docs/02 D-4): a job is enqueued in
// the same transaction as the state change that caused it, so a confirmation
// email for an order that rolled back cannot exist.
type JobRepo struct{ db *gorm.DB }

func NewJobRepo(db *gorm.DB) *JobRepo { return &JobRepo{db: db} }

// Job is one queued unit of work.
type Job struct {
	ID       uuid.UUID
	Kind     string
	Payload  json.RawMessage
	Attempts int
}

// Enqueue adds a job. A dedupe key collapses repeats: the expiry sweep runs
// hourly and must not send the same warning six times.
func (r *JobRepo) Enqueue(ctx context.Context, kind string, payload []byte,
	dedupe string, runAfter time.Time) error {

	var key any
	if dedupe != "" {
		key = dedupe
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO job (id, kind, payload, dedupe_key, run_after)
		VALUES (?, ?, ?::jsonb, ?, ?)
		ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL
		                          AND state IN ('PENDING','RUNNING') DO NOTHING`,
		uuid.Must(uuid.NewV7()), kind, string(payload), key, runAfter).Error
}

// Claim takes up to n pending jobs and marks them RUNNING.
//
// FOR UPDATE SKIP LOCKED is what lets a second worker start without the two
// fighting over the same rows — one takes what the other has not locked.
func (r *JobRepo) Claim(ctx context.Context, kind string, n int) ([]Job, error) {
	var jobs []Job
	err := r.db.WithContext(ctx).Raw(`
		UPDATE job SET state='RUNNING', attempts = attempts + 1, updated_at = now()
		 WHERE id IN (
		   SELECT id FROM job
		    WHERE kind = ? AND state = 'PENDING' AND run_after <= now()
		    ORDER BY run_after
		    LIMIT ?
		    FOR UPDATE SKIP LOCKED)
		 RETURNING id, kind, payload, attempts`, kind, n).Scan(&jobs).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: claim jobs: %w", err)
	}
	return jobs, nil
}

// Done marks a job complete.
func (r *JobRepo) Done(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE job SET state='DONE', updated_at=now() WHERE id=?`, id).Error
}

// Fail records an error and either reschedules with a backoff or gives up.
//
// A failed job stays visible as FAILED rather than vanishing: a message nobody
// can see was not sent is indistinguishable from one that was.
func (r *JobRepo) Fail(ctx context.Context, id uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE job
		   SET state = CASE WHEN attempts >= max_attempts THEN 'FAILED' ELSE 'PENDING' END,
		       run_after = now() + (interval '1 minute' * POWER(2, LEAST(attempts, 6))),
		       last_error = ?, updated_at = now()
		 WHERE id = ?`, truncateErr(reason), id).Error
}

// NotificationLog is one recorded send.
type NotificationLog struct {
	CustomerID    *uuid.UUID
	Channel       string
	Template      string
	Recipient     string
	Subject       string
	Locale        string
	ReferenceType string
	ReferenceID   *uuid.UUID
}

// LogNotification records an attempt before it is made, so a crash mid-send
// still leaves evidence.
func (r *JobRepo) LogNotification(ctx context.Context, n NotificationLog) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO notification_log
		  (id, customer_id, channel, template, recipient, subject, locale, state,
		   reference_type, reference_id)
		VALUES (?,?,?,?,?,?,?, 'QUEUED', NULLIF(?,''), ?)`,
		id, n.CustomerID, n.Channel, n.Template, n.Recipient, n.Subject,
		n.Locale, n.ReferenceType, n.ReferenceID).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: log notification: %w", err)
	}
	return id, nil
}

// MarkNotification records the outcome.
func (r *JobRepo) MarkNotification(ctx context.Context, id uuid.UUID, state, errMsg string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE notification_log
		   SET state = ?, error = NULLIF(?,''),
		       sent_at = CASE WHEN ? = 'SENT' THEN now() ELSE sent_at END
		 WHERE id = ?`, state, truncateErr(errMsg), state, id).Error
}

// PendingCount reports queue depth, for the dashboard and for tests.
func (r *JobRepo) PendingCount(ctx context.Context, kind string) (int, error) {
	var n int
	err := r.db.WithContext(ctx).Raw(
		`SELECT count(*) FROM job WHERE kind=? AND state='PENDING'`, kind).Scan(&n).Error
	return n, err
}

// truncateErr keeps a driver message from filling the column, and keeps the
// useful first line.
func truncateErr(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
