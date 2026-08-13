package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Token errors are deliberately coarse: a client learns that a token is
// unusable, never why (docs/12, A07).
var (
	ErrTokenInvalid = errors.New("security: token invalid")
	ErrTokenExpired = errors.New("security: token expired")
)

// Claims is the access-token payload. It carries the subject and role, but
// never the store list — store scope is resolved server-side on every request
// from staff_store_assignments (BR-2.7.9).
type Claims struct {
	jwt.RegisteredClaims
	SubjectType SubjectType `json:"typ"`
	Role        Role        `json:"role"`
	// Purpose is EMPTY for a normal access token. A token issued for a
	// narrower job — the half-authenticated state between a correct password
	// and a correct 2FA code — carries a purpose here, and RequireAuth refuses
	// anything with one. Without this, the challenge token handed out after
	// step one of login would itself be a working session, which is exactly
	// the thing 2FA exists to prevent.
	Purpose string `json:"pur,omitempty"`
}

// PurposeMFA marks the short-lived token that stands between a verified
// password and a verified second factor.
const PurposeMFA = "mfa"

// TokenSigner issues and verifies access tokens. PreviousKey allows a signing
// key to be rotated without invalidating live sessions (docs/09 §3).
type TokenSigner struct {
	key         []byte
	previousKey []byte
	issuer      string
	ttl         time.Duration
	now         func() time.Time
}

func NewTokenSigner(key, previousKey, issuer string, ttl time.Duration, now func() time.Time) *TokenSigner {
	if now == nil {
		now = time.Now
	}
	return &TokenSigner{
		key:         []byte(key),
		previousKey: []byte(previousKey),
		issuer:      issuer,
		ttl:         ttl,
		now:         now,
	}
}

// Issue returns a signed access token and its jti.
func (s *TokenSigner) Issue(subjectType SubjectType, subjectID uuid.UUID, role Role) (string, uuid.UUID, error) {
	jti := uuid.New()
	now := s.now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subjectID.String(),
			ID:        jti.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
		SubjectType: subjectType,
		Role:        role,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.key)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("security: sign: %w", err)
	}
	return signed, jti, nil
}

// IssueMFAChallenge returns a short-lived token that proves only that the
// password was correct.
//
// It carries no role, so even if the purpose check were somehow bypassed it
// would resolve to an account with nothing granted.
func (s *TokenSigner) IssueMFAChallenge(subjectID uuid.UUID, ttl time.Duration) (string, error) {
	now := s.now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subjectID.String(),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Purpose: PurposeMFA,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("security: sign challenge: %w", err)
	}
	return signed, nil
}

// ParseMFAChallenge accepts ONLY a challenge token and returns its subject.
func (s *TokenSigner) ParseMFAChallenge(raw string) (uuid.UUID, error) {
	claims, err := s.Parse(raw)
	if err != nil {
		return uuid.Nil, err
	}
	if claims.Purpose != PurposeMFA {
		// A full access token must not be usable to complete somebody's
		// pending 2FA challenge either — the check runs both ways.
		return uuid.Nil, ErrTokenInvalid
	}
	return uuid.Parse(claims.Subject)
}

// Parse verifies a token against the current key, then the previous key during
// rotation. The signing method is pinned to HS256, so "alg": "none" and RS/HS
// confusion attacks are rejected (docs/12, A02).
func (s *TokenSigner) Parse(raw string) (*Claims, error) {
	keyFunc := func(key []byte) jwt.Keyfunc {
		return func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrTokenInvalid
			}
			return key, nil
		}
	}

	parse := func(key []byte) (*Claims, error) {
		var claims Claims
		_, err := jwt.ParseWithClaims(raw, &claims, keyFunc(key),
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(s.issuer),
			jwt.WithTimeFunc(s.now),
		)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, ErrTokenExpired
			}
			return nil, ErrTokenInvalid
		}
		return &claims, nil
	}

	claims, err := parse(s.key)
	if err == nil {
		return claims, nil
	}
	if errors.Is(err, ErrTokenExpired) || len(s.previousKey) == 0 {
		return nil, err
	}
	return parse(s.previousKey)
}

// HashToken hashes a refresh token or verification token for storage. Refresh
// tokens are never stored in plaintext (BR-2.7.12).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
