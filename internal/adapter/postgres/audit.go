package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditRepo writes the append-only audit log.
//
// PROMPT §3: every staff write touching money, prices, customer type, credits
// or package expiry writes a row with actor, action, entity, before, after, IP
// and timestamp. The table refuses UPDATE and DELETE at the database level, so
// this is a one-way record rather than a table someone can tidy up.
type AuditRepo struct{ db *gorm.DB }

func NewAuditRepo(db *gorm.DB) *AuditRepo { return &AuditRepo{db: db} }

// Entry is one audited action.
type Entry struct {
	ActorID    *uuid.UUID
	ActorEmail string
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Before     any
	After      any
	Reason     string
	IP         string
	UserAgent  string
}

// Write appends an audit row.
//
// It takes a gorm.DB so the caller can pass a transaction: an audit row that
// commits when the change it describes rolled back is worse than no audit row,
// because it asserts something that never happened.
func (r *AuditRepo) Write(ctx context.Context, tx *gorm.DB, e Entry) error {
	db := tx
	if db == nil {
		db = r.db
	}

	before, err := marshalOrNil(e.Before)
	if err != nil {
		return fmt.Errorf("postgres: audit before: %w", err)
	}
	after, err := marshalOrNil(e.After)
	if err != nil {
		return fmt.Errorf("postgres: audit after: %w", err)
	}

	return db.WithContext(ctx).Exec(`
		INSERT INTO audit_log
		  (id, actor_id, actor_email, action, entity_type, entity_id,
		   before_state, after_state, reason, ip, user_agent)
		VALUES (?,?,?,?,?,?,?::jsonb,?::jsonb,?,NULLIF(?,'')::inet,?)`,
		uuid.Must(uuid.NewV7()), e.ActorID, e.ActorEmail, e.Action, e.EntityType,
		e.EntityID, before, after, e.Reason, e.IP, e.UserAgent).Error
}

func marshalOrNil(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}
