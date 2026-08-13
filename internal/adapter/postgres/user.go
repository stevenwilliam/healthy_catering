package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a row does not exist. Callers map it to a
// generic message: auth errors never reveal whether an account exists (99 §7).
var ErrNotFound = errors.New("postgres: not found")

// UserRepo owns app_user, roles and the token tables.
type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// User is the authentication record.
type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	Phone           *string
	FullName        string
	IsActive        bool
	IsStaff         bool
	EmailVerifiedAt *time.Time
	LastLoginAt     *time.Time
	FailedAttempts  int
	LockedUntil     *time.Time
}

// Locked reports whether the account is in a lockout window.
func (u User) Locked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// Customer is the profile attached to a customer user.
type Customer struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CustomerTypeID  uuid.UUID
	OrganisationID  *uuid.UUID
	PreferredLocale string
}

// FindByEmail loads a user by address. The column is CITEXT, so the comparison
// is case-insensitive in the database as well as normalised in the application.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, phone, full_name, is_active, is_staff,
		       email_verified_at, last_login_at, failed_attempts, locked_until
		  FROM app_user WHERE email = ?`, email).Scan(&u).Error
	if err != nil {
		return User{}, fmt.Errorf("postgres: find user: %w", err)
	}
	if u.ID == uuid.Nil {
		return User{}, ErrNotFound
	}
	return u, nil
}

// FindByID loads a user by primary key.
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, phone, full_name, is_active, is_staff,
		       email_verified_at, last_login_at, failed_attempts, locked_until
		  FROM app_user WHERE id = ?`, id).Scan(&u).Error
	if err != nil {
		return User{}, fmt.Errorf("postgres: find user: %w", err)
	}
	if u.ID == uuid.Nil {
		return User{}, ErrNotFound
	}
	return u, nil
}

// RegisterCustomer creates the user, the customer profile and the customer role
// grant in ONE transaction.
//
// All three or none: a user row without a customer profile cannot order and
// cannot be repaired by the customer, and a half-registered account is the kind
// of thing that is only discovered at checkout.
func (r *UserRepo) RegisterCustomer(ctx context.Context, u User, locale string) (uuid.UUID, uuid.UUID, error) {
	var userID, customerID uuid.UUID

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userID = uuid.Must(uuid.NewV7())
		customerID = uuid.Must(uuid.NewV7())

		if err := tx.Exec(`
			INSERT INTO app_user (id, email, password_hash, phone, full_name, is_active, is_staff)
			VALUES (?,?,?,?,?,TRUE,FALSE)`,
			userID, u.Email, u.PasswordHash, u.Phone, u.FullName).Error; err != nil {
			return err
		}

		// The default customer type is is_system, so it cannot be deleted out
		// from under a registration.
		if err := tx.Exec(`
			INSERT INTO customer (id, user_id, customer_type_id, preferred_locale, pdp_consent_at)
			SELECT ?, ?, ct.id, ?, now()
			  FROM customer_type ct
			 WHERE ct.is_system AND ct.slug = 'customer-default'`,
			customerID, userID, locale).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO user_role (user_id, role_id)
			SELECT ?, id FROM role WHERE code = 'customer'`, userID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("postgres: register: %w", err)
	}
	return userID, customerID, nil
}

// EmailTaken reports whether an address is already registered.
//
// Used only by the registration path, and even there the API's reply is
// deliberately identical whether or not the address existed — otherwise
// registration becomes an account-enumeration oracle (99 §7).
func (r *UserRepo) EmailTaken(ctx context.Context, email string) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Raw(
		`SELECT count(*) FROM app_user WHERE email = ?`, email).Scan(&n).Error; err != nil {
		return false, fmt.Errorf("postgres: email taken: %w", err)
	}
	return n > 0, nil
}

// Permissions returns the union of the permissions granted by a user's roles.
func (r *UserRepo) Permissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT p.code
		  FROM user_role ur
		  JOIN role_permission rp ON rp.role_id = ur.role_id
		  JOIN permission p       ON p.id = rp.permission_id
		 WHERE ur.user_id = ?`, userID).Scan(&codes).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: permissions: %w", err)
	}
	return codes, nil
}

// Roles returns a user's role codes.
func (r *UserRepo) Roles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.code FROM user_role ur JOIN role r ON r.id = ur.role_id
		 WHERE ur.user_id = ? ORDER BY r.code`, userID).Scan(&codes).Error
	if err != nil {
		return nil, fmt.Errorf("postgres: roles: %w", err)
	}
	return codes, nil
}

// CustomerByUser loads the customer profile for a user.
func (r *UserRepo) CustomerByUser(ctx context.Context, userID uuid.UUID) (Customer, error) {
	var c Customer
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, customer_type_id, organisation_id, preferred_locale
		  FROM customer WHERE user_id = ?`, userID).Scan(&c).Error
	if err != nil {
		return Customer{}, fmt.Errorf("postgres: customer: %w", err)
	}
	if c.ID == uuid.Nil {
		return Customer{}, ErrNotFound
	}
	return c, nil
}

// KitchenScope returns the kitchen a staff user is scoped to, if any. NULL
// means all kitchens (docs/02 D-21). Enforced in the repository layer, not the
// handler, so a new query cannot forget it.
func (r *UserRepo) KitchenScope(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var out []uuid.UUID
	if err := r.db.WithContext(ctx).Raw(
		`SELECT kitchen_id FROM staff_profile WHERE user_id = ? AND kitchen_id IS NOT NULL`,
		userID).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("postgres: kitchen scope: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return &out[0], nil
}

// RecordLoginSuccess clears the failure counter and stamps the login.
func (r *UserRepo) RecordLoginSuccess(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE app_user
		   SET failed_attempts = 0, locked_until = NULL, last_login_at = now()
		 WHERE id = ?`, userID).Error
}

// RecordLoginFailure increments the counter and locks the account once it
// crosses the threshold. Progressive lockout with a documented unlock path
// (99 §7); the unlock path is the lockout expiring, or an admin clearing it.
func (r *UserRepo) RecordLoginFailure(ctx context.Context, userID uuid.UUID, max int, lockFor time.Duration) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE app_user
		   SET failed_attempts = failed_attempts + 1,
		       locked_until = CASE WHEN failed_attempts + 1 >= ?
		                           THEN now() + ?::interval
		                           ELSE locked_until END
		 WHERE id = ?`, max, fmt.Sprintf("%d seconds", int(lockFor.Seconds())), userID).Error
}

// StoreRefreshToken saves a hashed refresh token. The raw token is never
// stored: a database leak must not hand over live sessions (99 §7).
func (r *UserRepo) StoreRefreshToken(ctx context.Context, id, userID uuid.UUID, hash string,
	expiresAt time.Time, ua string, ip string, parent *uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO refresh_token (id, user_id, token_hash, parent_id, user_agent, ip, expires_at)
		VALUES (?,?,?,?,?,NULLIF(?,'')::inet,?)`,
		id, userID, hash, parent, ua, ip, expiresAt).Error
}

// ConsumeRefreshToken atomically finds a live token and revokes it, returning
// the owner.
//
// Rotation is the point: a refresh token is single-use, so replaying a stolen
// one after the legitimate client has already rotated finds it revoked. The
// UPDATE ... RETURNING makes the find-and-revoke one statement, so two
// concurrent replays cannot both succeed.
func (r *UserRepo) ConsumeRefreshToken(ctx context.Context, hash string) (uuid.UUID, uuid.UUID, error) {
	var row struct {
		ID     uuid.UUID
		UserID uuid.UUID
	}
	err := r.db.WithContext(ctx).Raw(`
		UPDATE refresh_token
		   SET revoked_at = now(), revoked_reason = 'rotated'
		 WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > now()
		 RETURNING id, user_id`, hash).Scan(&row).Error
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("postgres: consume refresh: %w", err)
	}
	if row.ID == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	return row.ID, row.UserID, nil
}

// RevokeAllRefreshTokens ends every session for a user — logout-everywhere, and
// what a privilege change or a password reset must trigger.
func (r *UserRepo) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE refresh_token SET revoked_at = now(), revoked_reason = ?
		 WHERE user_id = ? AND revoked_at IS NULL`, reason, userID).Error
}

// StoreVerificationToken saves a hashed, single-use, expiring token.
func (r *UserRepo) StoreVerificationToken(ctx context.Context, userID uuid.UUID,
	purpose, hash string, ttl time.Duration) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO verification_token (id, user_id, purpose, token_hash, expires_at)
		VALUES (?,?,?,?, now() + ?::interval)`,
		uuid.Must(uuid.NewV7()), userID, purpose, hash,
		fmt.Sprintf("%d seconds", int(ttl.Seconds()))).Error
}

// ConsumeVerificationToken marks a token used and returns its owner. Single-use
// is enforced by the same UPDATE ... RETURNING trick as refresh rotation.
func (r *UserRepo) ConsumeVerificationToken(ctx context.Context, purpose, hash string) (uuid.UUID, error) {
	var row struct{ UserID uuid.UUID }
	err := r.db.WithContext(ctx).Raw(`
		UPDATE verification_token
		   SET consumed_at = now()
		 WHERE token_hash = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > now()
		 RETURNING user_id`, hash, purpose).Scan(&row).Error
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres: consume verification: %w", err)
	}
	if row.UserID == uuid.Nil {
		return uuid.Nil, ErrNotFound
	}
	return row.UserID, nil
}

// ContactFor returns the address, name and locale to write to. Best-effort:
// a notification is never worth failing the action that triggered it.
func (r *UserRepo) ContactFor(ctx context.Context, userID uuid.UUID) (email, name, locale string) {
	var row struct {
		Email  string
		Name   string
		Locale string
	}
	_ = r.db.WithContext(ctx).Raw(`
		SELECT u.email::text AS email, u.full_name AS name,
		       COALESCE(c.preferred_locale, 'id-ID') AS locale
		  FROM app_user u LEFT JOIN customer c ON c.user_id = u.id
		 WHERE u.id = ?`, userID).Scan(&row).Error
	return row.Email, row.Name, row.Locale
}

// ContactForCustomer is ContactFor keyed by customer rather than user.
func (r *UserRepo) ContactForCustomer(ctx context.Context, customerID uuid.UUID) (email, name, locale string) {
	var row struct {
		Email  string
		Name   string
		Locale string
	}
	_ = r.db.WithContext(ctx).Raw(`
		SELECT u.email::text AS email, u.full_name AS name,
		       COALESCE(c.preferred_locale,'id-ID') AS locale
		  FROM customer c JOIN app_user u ON u.id = c.user_id
		 WHERE c.id = ?`, customerID).Scan(&row).Error
	return row.Email, row.Name, row.Locale
}

// MarkEmailVerified stamps the verification.
func (r *UserRepo) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE app_user SET email_verified_at = now() WHERE id = ? AND email_verified_at IS NULL`,
		userID).Error
}
