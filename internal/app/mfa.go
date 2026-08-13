package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// mfaChallengeTTL bounds the half-authenticated window.
//
// Five minutes is enough to fetch a phone and long enough to be annoying if
// leaked, which is why the token is single-purpose and role-less.
const mfaChallengeTTL = 5 * time.Minute

// MFA is staff second-factor enrolment and verification (docs/03 Q-16).
//
// Kitchen and courier roles are exempt — they sign in from shared phones on a
// service floor — so this applies to admin, finance and staff.
type MFA struct {
	totp   *postgres.TOTPRepo
	users  *postgres.UserRepo
	audit  *postgres.AuditRepo
	cipher *security.TOTPCipher
	issuer string
	now    func() time.Time
}

func NewMFA(t *postgres.TOTPRepo, u *postgres.UserRepo, a *postgres.AuditRepo,
	cipher *security.TOTPCipher, issuer string, now func() time.Time) *MFA {
	if now == nil {
		now = time.Now
	}
	if issuer == "" {
		issuer = "Evermore"
	}
	return &MFA{totp: t, users: u, audit: a, cipher: cipher, issuer: issuer, now: now}
}

// Configured reports whether 2FA can be used at all.
//
// Without TOTP_ENCRYPTION_KEY the secrets could only be stored in the clear,
// and storing them in the clear is worse than not offering the feature.
func (m *MFA) Configured() bool { return m != nil && m.cipher != nil }

// EnrolmentStarted is what the user needs to add the account to their app.
type EnrolmentStarted struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
	Message    string `json:"message"`
}

// Start issues a new secret. Enrolment is not complete until Confirm.
func (m *MFA) Start(ctx context.Context, ident Identity, by Actor) (EnrolmentStarted, error) {
	if !m.Configured() {
		return EnrolmentStarted{}, apierror.Conflict(apierror.CodeConflict,
			"Two-factor authentication is not configured on this server.")
	}

	// Re-enrolling is allowed — a replaced phone is the ordinary case — but it
	// throws away the old secret and the old recovery codes, so the user must
	// prove the new one works before the account depends on it.
	secret, err := security.NewTOTPSecret()
	if err != nil {
		return EnrolmentStarted{}, apierror.Internal(err)
	}
	cipher, err := m.cipher.Encrypt(secret)
	if err != nil {
		return EnrolmentStarted{}, apierror.Internal(err)
	}
	if err := m.totp.StartEnrolment(ctx, ident.UserID, cipher); err != nil {
		return EnrolmentStarted{}, apierror.Internal(err)
	}

	m.log(ctx, by, "mfa.enrol.start", ident.UserID, "")

	return EnrolmentStarted{
		Secret:     secret,
		OTPAuthURL: security.OTPAuthURL(m.issuer, by.Email, secret),
		Message: "Scan this in Google Authenticator or Authy, then enter the " +
			"six-digit code to finish. Nothing changes until you do.",
	}, nil
}

// EnrolmentConfirmed carries the recovery codes, shown exactly once.
type EnrolmentConfirmed struct {
	RecoveryCodes []string `json:"recovery_codes"`
	Message       string   `json:"message"`
}

// Confirm completes enrolment once a real code proves the app is set up.
func (m *MFA) Confirm(ctx context.Context, ident Identity, code string, by Actor) (EnrolmentConfirmed, error) {
	if !m.Configured() {
		return EnrolmentConfirmed{}, apierror.Conflict(apierror.CodeConflict,
			"Two-factor authentication is not configured on this server.")
	}

	row, err := m.totp.Get(ctx, ident.UserID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return EnrolmentConfirmed{}, apierror.Conflict(apierror.CodeConflict,
				"Start enrolment first — there is no pending secret to confirm.")
		}
		return EnrolmentConfirmed{}, apierror.Internal(err)
	}
	if row.ConfirmedAt != nil {
		return EnrolmentConfirmed{}, apierror.Conflict(apierror.CodeConflict,
			"Two-factor authentication is already switched on for this account.")
	}

	secret, err := m.cipher.Decrypt(row.SecretCipher)
	if err != nil {
		return EnrolmentConfirmed{}, apierror.Internal(err)
	}
	step, err := security.VerifyTOTP(secret, code, m.now(), row.LastUsedStep)
	if err != nil {
		return EnrolmentConfirmed{}, badCode(err)
	}

	plain, hashed, err := security.NewRecoveryCodes(8)
	if err != nil {
		return EnrolmentConfirmed{}, apierror.Internal(err)
	}
	if err := m.totp.Confirm(ctx, ident.UserID, step, hashed); err != nil {
		if errors.Is(err, postgres.ErrWrongState) {
			return EnrolmentConfirmed{}, apierror.Conflict(apierror.CodeConflict,
				"Two-factor authentication is already switched on for this account.")
		}
		return EnrolmentConfirmed{}, apierror.Internal(err)
	}

	m.log(ctx, by, "mfa.enrol.confirm", ident.UserID, "")

	return EnrolmentConfirmed{
		RecoveryCodes: plain,
		Message: "Two-factor authentication is on. Save these recovery codes " +
			"somewhere safe — each works once, and this is the only time they " +
			"are shown. Without them, a lost phone means a lost account.",
	}, nil
}

// Verify completes a login challenge and returns the step to record.
//
// A recovery code is accepted in place of a TOTP code and is consumed on use.
func (m *MFA) Verify(ctx context.Context, userID uuid.UUID, code string) error {
	if !m.Configured() {
		return apierror.Internal(security.ErrNoTOTPKey)
	}

	row, err := m.totp.Get(ctx, userID)
	if err != nil {
		return apierror.Unauthorized("That code is not correct.")
	}
	if row.ConfirmedAt == nil {
		return apierror.Unauthorized("That code is not correct.")
	}

	secret, err := m.cipher.Decrypt(row.SecretCipher)
	if err != nil {
		return apierror.Internal(err)
	}

	step, err := security.VerifyTOTP(secret, code, m.now(), row.LastUsedStep)
	if err == nil {
		// The database, not this process, decides who wins a race between two
		// submissions of the same code.
		if err := m.totp.RecordUse(ctx, userID, step); err != nil {
			return apierror.Unauthorized("That code has already been used.")
		}
		return nil
	}
	if errors.Is(err, security.ErrTOTPReplayed) {
		return apierror.Unauthorized("That code has already been used. Wait for the next one.")
	}

	// Fall back to a recovery code before giving up.
	if idx, ok := security.MatchRecoveryCode(code, row.Codes()); ok {
		if err := m.totp.ConsumeRecoveryCode(ctx, userID, row.Codes()[idx]); err != nil {
			return apierror.Unauthorized("That recovery code has already been used.")
		}
		return nil
	}
	return apierror.Unauthorized("That code is not correct.")
}

// Status describes the account's second factor.
func (m *MFA) Status(ctx context.Context, ident Identity) (map[string]any, error) {
	out := map[string]any{
		"available": m.Configured(),
		"required":  ident.RequiresTOTP(),
		"enabled":   false,
	}
	row, err := m.totp.Get(ctx, ident.UserID)
	if err == nil {
		out["enabled"] = row.ConfirmedAt != nil
		out["pending"] = row.ConfirmedAt == nil
		out["recovery_codes_left"] = len(row.Codes())
	}
	return out, nil
}

// Disable switches 2FA off, which a required role may not do.
//
// The password is re-checked because "turn off my second factor" is exactly
// what someone does with a borrowed, already-signed-in laptop.
func (m *MFA) Disable(ctx context.Context, ident Identity, password string, by Actor) error {
	if ident.RequiresTOTP() {
		return apierror.Forbidden(apierror.CodeForbidden,
			"Two-factor authentication is mandatory for this role and cannot be switched off.")
	}

	user, err := m.users.FindByID(ctx, ident.UserID)
	if err != nil {
		return apierror.Internal(err)
	}
	ok, err := security.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return apierror.Internal(err)
	}
	if !ok {
		return apierror.Unauthorized("That password is not correct.")
	}

	if err := m.totp.Disable(ctx, ident.UserID); err != nil {
		return apierror.Internal(err)
	}
	m.log(ctx, by, "mfa.disable", ident.UserID, "user requested")
	return nil
}

func badCode(err error) error {
	if errors.Is(err, security.ErrTOTPReplayed) {
		return apierror.Unauthorized("That code has already been used. Wait for the next one.")
	}
	return apierror.Unauthorized("That code is not correct. Check your phone's clock is set automatically.")
}

func (m *MFA) log(ctx context.Context, by Actor, action string, userID uuid.UUID, reason string) {
	_ = m.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &by.UserID, ActorEmail: by.Email, Action: action,
		EntityType: "user", EntityID: &userID, Reason: reason,
		IP: by.IP, UserAgent: by.UA,
	})
}
