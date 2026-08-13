package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stevenwilliam/healthy_catering/internal/adapter/postgres"
	"github.com/stevenwilliam/healthy_catering/internal/platform/apierror"
	"github.com/stevenwilliam/healthy_catering/internal/platform/id"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sanitize"
	"github.com/stevenwilliam/healthy_catering/internal/platform/security"
	"github.com/stevenwilliam/healthy_catering/internal/platform/sysparam"
)

// Auth is registration, login, refresh and verification.
//
// Every failure that could reveal whether an account exists returns the SAME
// error (99 §7). A login that says "no such user" for one address and "wrong
// password" for another is an account-enumeration oracle, and the whole point
// of hashing passwords is undone by leaking which addresses are worth attacking.
type Auth struct {
	users  *postgres.UserRepo
	audit  *postgres.AuditRepo
	params *sysparam.Store
	signer *security.TokenSigner
	now    func() time.Time

	refreshTTL time.Duration
	verifyTTL  time.Duration

	// mfa is optional: attached after construction, nil when the server has no
	// TOTP key configured.
	mfa *MFA
}

// AuthDeps wires the service.
type AuthDeps struct {
	Users      *postgres.UserRepo
	Audit      *postgres.AuditRepo
	Params     *sysparam.Store
	Signer     *security.TokenSigner
	Now        func() time.Time
	RefreshTTL time.Duration
	VerifyTTL  time.Duration
}

func NewAuth(d AuthDeps) *Auth {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.RefreshTTL == 0 {
		d.RefreshTTL = 30 * 24 * time.Hour
	}
	if d.VerifyTTL == 0 {
		d.VerifyTTL = 24 * time.Hour
	}
	return &Auth{
		users: d.Users, audit: d.Audit, params: d.Params, signer: d.Signer,
		now: d.Now, refreshTTL: d.RefreshTTL, verifyTTL: d.VerifyTTL,
	}
}

// ErrInvalidCredentials is the single answer to every failed login, whatever
// the underlying cause.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// Session is what a successful authentication returns.
type Session struct {
	AccessToken   string     `json:"access_token"`
	RefreshToken  string     `json:"refresh_token"`
	ExpiresIn     int        `json:"expires_in"`
	UserID        uuid.UUID  `json:"user_id"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty"`
	FullName      string     `json:"full_name"`
	Email         string     `json:"email"`
	Roles         []string   `json:"roles"`
	Permissions   []string   `json:"permissions"`
	EmailVerified bool       `json:"email_verified"`

	// MFARequired means this is NOT a session yet: the password was right and
	// a second factor is still owed. AccessToken is empty in that case, so a
	// client that ignores this field gets no working token rather than a
	// half-authenticated one.
	MFARequired bool   `json:"mfa_required,omitempty"`
	MFAToken    string `json:"mfa_token,omitempty"`
	MFAHint     string `json:"mfa_hint,omitempty"`
}

// RegisterInput is a customer registration. Every field is re-validated here
// even though the form validated it too — the form can be bypassed
// (CLAUDE.md §4).
type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
	Locale   string
	IP       string
	UA       string
}

// RegisterResult carries the verification token so the caller can mail it. The
// token is returned rather than sent here so the service has no dependency on
// the mailer, and so a test can assert on it.
type RegisterResult struct {
	UserID            uuid.UUID
	CustomerID        uuid.UUID
	VerificationToken string
	AlreadyRegistered bool
}

// Register creates a customer account.
//
// When the address is already registered it returns AlreadyRegistered and no
// token, and the HANDLER still replies exactly as it would for a new account.
// That keeps registration from becoming an enumeration oracle while letting the
// caller send a "someone tried to register your address" mail instead.
func (a *Auth) Register(ctx context.Context, in RegisterInput) (RegisterResult, error) {
	email, err := sanitize.Email("email", in.Email, 254)
	if err != nil {
		return RegisterResult{}, validationFrom(err)
	}
	name, err := sanitize.Required("full_name", in.FullName, 120)
	if err != nil {
		return RegisterResult{}, validationFrom(err)
	}
	var phone *string
	if in.Phone != "" {
		p, err := sanitize.Phone("phone", in.Phone)
		if err != nil {
			return RegisterResult{}, validationFrom(err)
		}
		phone = &p
	}
	locale := "id-ID"
	if in.Locale != "" {
		locale, err = sanitize.Enum("locale", in.Locale, "id-ID", "en")
		if err != nil {
			return RegisterResult{}, validationFrom(err)
		}
	}
	if err := security.ValidatePassword(in.Password); err != nil {
		// The reason is useful — a customer needs to know WHY the password was
		// refused — but the internal "security:" package prefix is not, and
		// leaking package names to clients is what CLAUDE.md §4 forbids.
		return RegisterResult{}, apierror.Validation(
			"That password is not strong enough.",
			map[string]any{"password": passwordAdvice(err)})
	}

	taken, err := a.users.EmailTaken(ctx, email)
	if err != nil {
		return RegisterResult{}, apierror.Internal(err)
	}
	if taken {
		return RegisterResult{AlreadyRegistered: true}, nil
	}

	hash, err := security.HashPassword(in.Password)
	if err != nil {
		return RegisterResult{}, apierror.Internal(err)
	}

	userID, customerID, err := a.users.RegisterCustomer(ctx, postgres.User{
		Email: email, PasswordHash: hash, FullName: name, Phone: phone,
	}, locale)
	if err != nil {
		return RegisterResult{}, apierror.Internal(err)
	}

	token, err := id.Token(32)
	if err != nil {
		return RegisterResult{}, apierror.Internal(err)
	}
	if err := a.users.StoreVerificationToken(ctx, userID, "EMAIL_VERIFY",
		security.HashToken(token), a.verifyTTL); err != nil {
		return RegisterResult{}, apierror.Internal(err)
	}

	_ = a.audit.Write(ctx, nil, postgres.Entry{
		ActorID: &userID, ActorEmail: email, Action: "customer.register",
		EntityType: "customer", EntityID: &customerID, IP: in.IP, UserAgent: in.UA,
	})

	return RegisterResult{UserID: userID, CustomerID: customerID, VerificationToken: token}, nil
}

// LoginInput is a credential attempt.
type LoginInput struct {
	Email    string
	Password string
	IP       string
	UA       string
}

// Login verifies credentials and issues a session.
func (a *Auth) Login(ctx context.Context, in LoginInput) (Session, error) {
	email, err := sanitize.Email("email", in.Email, 254)
	if err != nil {
		// Even a malformed address gets the generic answer: telling a
		// scanner which addresses are well-formed is free information.
		return Session{}, apierror.Unauthorized("Email or password is incorrect.")
	}

	user, err := a.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			// Hash a dummy password anyway so a missing account does not
			// return measurably faster than a wrong password.
			security.VerifyPasswordDummy()
			return Session{}, apierror.Unauthorized("Email or password is incorrect.")
		}
		return Session{}, apierror.Internal(err)
	}

	now := a.now()
	if user.Locked(now) {
		return Session{}, apierror.TooManyRequests(
			"Too many failed attempts. Please try again later.")
	}
	if !user.IsActive {
		return Session{}, apierror.Unauthorized("Email or password is incorrect.")
	}

	ok, err := security.VerifyPassword(in.Password, user.PasswordHash)
	if err != nil {
		return Session{}, apierror.Internal(err)
	}
	if !ok {
		maxAttempts := a.params.Int(ctx, sysparam.KeyMaxLoginAttempts, 5)
		lockFor := a.params.Duration(ctx, sysparam.KeyLockoutDuration, 15*time.Minute)
		_ = a.users.RecordLoginFailure(ctx, user.ID, maxAttempts, lockFor)
		return Session{}, apierror.Unauthorized("Email or password is incorrect.")
	}

	if err := a.users.RecordLoginSuccess(ctx, user.ID); err != nil {
		return Session{}, apierror.Internal(err)
	}

	// A correct password is only STEP ONE for an account with a second factor.
	// Nothing is issued here but a challenge token, which grants no permissions
	// and expires in minutes.
	if a.mfa != nil && a.mfa.Configured() {
		enrolled, err := a.mfa.totp.Enrolled(ctx, user.ID)
		if err != nil {
			return Session{}, apierror.Internal(err)
		}
		if enrolled {
			challenge, err := a.signer.IssueMFAChallenge(user.ID, mfaChallengeTTL)
			if err != nil {
				return Session{}, apierror.Internal(err)
			}
			return Session{
				MFARequired:   true,
				MFAToken:      challenge,
				ExpiresIn:     int(mfaChallengeTTL.Seconds()),
				UserID:        user.ID,
				Email:         user.Email,
				MFAHint:       "Enter the six-digit code from your authenticator app.",
				EmailVerified: user.EmailVerifiedAt != nil,
			}, nil
		}
	}

	return a.issue(ctx, user, in.IP, in.UA, nil)
}

// CompleteMFA exchanges a verified challenge for a real session.
func (a *Auth) CompleteMFA(ctx context.Context, challenge, code, ip, ua string) (Session, error) {
	if a.mfa == nil || !a.mfa.Configured() {
		return Session{}, apierror.Internal(security.ErrNoTOTPKey)
	}

	userID, err := a.signer.ParseMFAChallenge(challenge)
	if err != nil {
		// Expired or reused: send them back to the password step rather than
		// leaving them guessing at a code that can never work.
		return Session{}, apierror.Unauthorized("That sign-in attempt expired. Please sign in again.")
	}

	if err := a.mfa.Verify(ctx, userID, code); err != nil {
		return Session{}, err
	}

	user, err := a.users.FindByID(ctx, userID)
	if err != nil {
		return Session{}, apierror.Unauthorized("Please sign in again.")
	}
	if !user.IsActive {
		// The account may have been deactivated between the two steps.
		return Session{}, apierror.Unauthorized("Email or password is incorrect.")
	}
	return a.issue(ctx, user, ip, ua, nil)
}

// AttachMFA wires the second-factor service after construction, since the two
// services need each other.
func (a *Auth) AttachMFA(m *MFA) { a.mfa = m }

// Refresh rotates a refresh token and issues a new session.
func (a *Auth) Refresh(ctx context.Context, raw, ip, ua string) (Session, error) {
	tokenID, userID, err := a.users.ConsumeRefreshToken(ctx, security.HashToken(raw))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return Session{}, apierror.Unauthorized("Please sign in again.")
		}
		return Session{}, apierror.Internal(err)
	}
	user, err := a.users.FindByID(ctx, userID)
	if err != nil {
		return Session{}, apierror.Unauthorized("Please sign in again.")
	}
	if !user.IsActive {
		return Session{}, apierror.Unauthorized("Please sign in again.")
	}
	return a.issue(ctx, user, ip, ua, &tokenID)
}

// Logout revokes every session for the user. Logging out on a shared machine
// should not leave a live session on it (99 §7).
func (a *Auth) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := a.users.RevokeAllRefreshTokens(ctx, userID, "logout"); err != nil {
		return apierror.Internal(err)
	}
	return nil
}

// VerifyEmail consumes a verification token.
func (a *Auth) VerifyEmail(ctx context.Context, raw string) error {
	userID, err := a.users.ConsumeVerificationToken(ctx, "EMAIL_VERIFY", security.HashToken(raw))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return apierror.BadRequest(apierror.CodeValidation,
				"That verification link is invalid or has expired.")
		}
		return apierror.Internal(err)
	}
	if err := a.users.MarkEmailVerified(ctx, userID); err != nil {
		return apierror.Internal(err)
	}
	return nil
}

// Identity is the resolved caller, attached to the request context.
type Identity struct {
	UserID        uuid.UUID
	CustomerID    *uuid.UUID
	Email         string
	IsStaff       bool
	Roles         []string
	Permissions   security.Set
	KitchenID     *uuid.UUID
	EmailVerified bool
}

// RequiresTOTP reports whether ANY of the caller's roles makes 2FA mandatory
// (docs/03 Q-16).
//
// Any, not all: a user who is both a courier and an admin holds admin powers,
// and the weaker role must not be a way to sign in without the second factor.
func (i Identity) RequiresTOTP() bool {
	for _, r := range i.Roles {
		if security.Role(r).RequiresTOTP() {
			return true
		}
	}
	return false
}

// Resolve loads the caller's authorization from the database on every request.
//
// Permissions are NOT read from the token. A role change, a deactivation or a
// kitchen re-scope must take effect immediately, and a token minted before the
// change would otherwise keep its old powers until it expired.
func (a *Auth) Resolve(ctx context.Context, userID uuid.UUID) (Identity, error) {
	user, err := a.users.FindByID(ctx, userID)
	if err != nil {
		return Identity{}, apierror.Unauthorized("Please sign in again.")
	}
	if !user.IsActive {
		return Identity{}, apierror.Unauthorized("Please sign in again.")
	}

	perms, err := a.users.Permissions(ctx, userID)
	if err != nil {
		return Identity{}, apierror.Internal(err)
	}
	roles, err := a.users.Roles(ctx, userID)
	if err != nil {
		return Identity{}, apierror.Internal(err)
	}

	ident := Identity{
		UserID: user.ID, Email: user.Email, IsStaff: user.IsStaff,
		Roles: roles, Permissions: security.NewSet(perms),
		EmailVerified: user.EmailVerifiedAt != nil,
	}

	if user.IsStaff {
		if k, err := a.users.KitchenScope(ctx, userID); err == nil {
			ident.KitchenID = k
		}
	} else if c, err := a.users.CustomerByUser(ctx, userID); err == nil {
		ident.CustomerID = &c.ID
	}
	return ident, nil
}

func (a *Auth) issue(ctx context.Context, user postgres.User, ip, ua string, parent *uuid.UUID) (Session, error) {
	roles, err := a.users.Roles(ctx, user.ID)
	if err != nil {
		return Session{}, apierror.Internal(err)
	}
	perms, err := a.users.Permissions(ctx, user.ID)
	if err != nil {
		return Session{}, apierror.Internal(err)
	}

	subject := security.SubjectCustomer
	primary := security.RoleCustomer
	if user.IsStaff {
		subject = security.SubjectStaff
		if len(roles) > 0 {
			primary = security.Role(roles[0])
		}
	}

	access, _, err := a.signer.Issue(subject, user.ID, primary)
	if err != nil {
		return Session{}, apierror.Internal(err)
	}

	refresh, err := id.Token(48)
	if err != nil {
		return Session{}, apierror.Internal(err)
	}
	if err := a.users.StoreRefreshToken(ctx, uuid.Must(uuid.NewV7()), user.ID,
		security.HashToken(refresh), a.now().Add(a.refreshTTL), ua, ip, parent); err != nil {
		return Session{}, apierror.Internal(err)
	}

	s := Session{
		AccessToken: access, RefreshToken: refresh, ExpiresIn: int((15 * time.Minute).Seconds()),
		UserID: user.ID, FullName: user.FullName, Email: user.Email,
		Roles: roles, Permissions: perms, EmailVerified: user.EmailVerifiedAt != nil,
	}
	if !user.IsStaff {
		if c, err := a.users.CustomerByUser(ctx, user.ID); err == nil {
			s.CustomerID = &c.ID
		}
	}
	return s, nil
}

// passwordAdvice turns an internal validation error into customer-facing
// guidance, in the customer's terms rather than the package's.
func passwordAdvice(err error) string {
	switch {
	case errors.Is(err, security.ErrPasswordTooShort):
		return fmt.Sprintf("Use at least %d characters.", security.MinPasswordLength)
	case errors.Is(err, security.ErrPasswordBreached):
		return "That password has appeared in a known data breach. Please choose another."
	case errors.Is(err, security.ErrPasswordTrivial):
		return "That password is too predictable. Mix in other words or characters."
	default:
		return "Please choose a longer, less predictable password."
	}
}

// validationFrom converts a sanitize rejection into the API error model.
func validationFrom(err error) error {
	var se *sanitize.Error
	if errors.As(err, &se) {
		return apierror.Validation("Please check the highlighted field.",
			map[string]any{se.Field: se.Reason})
	}
	return apierror.Validation(fmt.Sprintf("Invalid input: %v", err), nil)
}
