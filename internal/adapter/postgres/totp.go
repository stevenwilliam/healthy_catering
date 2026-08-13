package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrTOTPReplay means the code was already spent — see RecordUse.
var ErrTOTPReplay = errors.New("postgres: totp code already used")

// TOTPRepo persists staff second-factor enrolment (docs/03 Q-16).
type TOTPRepo struct{ db *gorm.DB }

func NewTOTPRepo(db *gorm.DB) *TOTPRepo { return &TOTPRepo{db: db} }

// TOTPRow is one user's enrolment.
type TOTPRow struct {
	UserID        uuid.UUID  `gorm:"column:user_id"`
	SecretCipher  string     `gorm:"column:secret_cipher"`
	ConfirmedAt   *time.Time `gorm:"column:confirmed_at"`
	LastUsedStep  *int64     `gorm:"column:last_used_step"`
	RecoveryCodes []byte     `gorm:"column:recovery_codes"`
}

// Codes decodes the stored hashes.
func (r TOTPRow) Codes() []string {
	var out []string
	if len(r.RecoveryCodes) > 0 {
		_ = json.Unmarshal(r.RecoveryCodes, &out)
	}
	return out
}

// Get returns the enrolment, or ErrNotFound.
func (r *TOTPRepo) Get(ctx context.Context, userID uuid.UUID) (TOTPRow, error) {
	var row TOTPRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT user_id, secret_cipher, confirmed_at, last_used_step, recovery_codes
		  FROM user_totp WHERE user_id = ?`, userID).Scan(&row).Error
	if err != nil {
		return TOTPRow{}, err
	}
	if row.UserID == uuid.Nil {
		return TOTPRow{}, ErrNotFound
	}
	return row, nil
}

// StartEnrolment stores an UNCONFIRMED secret, replacing any previous attempt.
//
// Unconfirmed on purpose: writing confirmed_at here would let a user enrol a
// secret they never proved they can read, and lock themselves out on the next
// sign-in. The row only counts once a code from the app has been verified.
func (r *TOTPRepo) StartEnrolment(ctx context.Context, userID uuid.UUID, cipher string) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO user_totp (user_id, secret_cipher, confirmed_at, last_used_step, recovery_codes)
		VALUES (?, ?, NULL, NULL, '[]'::jsonb)
		ON CONFLICT (user_id) DO UPDATE
		   SET secret_cipher = EXCLUDED.secret_cipher,
		       confirmed_at  = NULL,
		       last_used_step = NULL,
		       recovery_codes = '[]'::jsonb,
		       updated_at    = now()`, userID, cipher).Error
}

// Confirm marks enrolment complete and stores the recovery-code hashes.
//
// Guarded on confirmed_at IS NULL so a second confirm cannot silently mint a
// fresh set of recovery codes for an account someone else has open.
func (r *TOTPRepo) Confirm(ctx context.Context, userID uuid.UUID, step int64, hashes []string) error {
	payload, err := json.Marshal(hashes)
	if err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Exec(`
		UPDATE user_totp
		   SET confirmed_at = now(), last_used_step = ?, recovery_codes = ?::jsonb,
		       updated_at = now()
		 WHERE user_id = ? AND confirmed_at IS NULL`, step, string(payload), userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrWrongState
	}
	return nil
}

// RecordUse advances the replay watermark.
//
// The guard on last_used_step is what makes replay protection hold under
// CONCURRENCY: two simultaneous submissions of the same code both verify in Go,
// and only one of them lands here.
func (r *TOTPRepo) RecordUse(ctx context.Context, userID uuid.UUID, step int64) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE user_totp
		   SET last_used_step = ?, updated_at = now()
		 WHERE user_id = ?
		   AND (last_used_step IS NULL OR last_used_step < ?)`, step, userID, step)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTOTPReplay
	}
	return nil
}

// ConsumeRecoveryCode removes one code from the stored set.
//
// The removal is the whole point: a recovery code is single-use, and doing this
// in SQL means two parallel uses of the same code cannot both succeed.
func (r *TOTPRepo) ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, hash string) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE user_totp
		   SET recovery_codes = (
		         SELECT COALESCE(jsonb_agg(c), '[]'::jsonb)
		           FROM jsonb_array_elements(recovery_codes) AS c
		          WHERE c <> to_jsonb(?::text)),
		       updated_at = now()
		 WHERE user_id = ?
		   AND recovery_codes @> to_jsonb(?::text)`, hash, userID, hash)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Disable removes enrolment entirely.
func (r *TOTPRepo) Disable(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM user_totp WHERE user_id = ?`, userID).Error
}

// Enrolled reports whether the user has a CONFIRMED second factor.
func (r *TOTPRepo) Enrolled(ctx context.Context, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM user_totp
		 WHERE user_id = ? AND confirmed_at IS NOT NULL`, userID).Scan(&n).Error
	return n > 0, err
}
