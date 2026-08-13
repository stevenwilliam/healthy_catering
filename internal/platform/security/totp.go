package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/stevenwilliam/healthy_catering/internal/platform/id"
)

// TOTP — RFC 6238, SHA-1, 6 digits, 30-second step.
//
// SHA-1 is not a security weakness here and is not negotiable: every
// authenticator app on a phone speaks exactly this dialect, and HMAC-SHA1 is
// unaffected by the SHA-1 collision work. Changing any of these constants
// produces codes that Google Authenticator silently rejects.
const (
	totpDigits = 6
	totpPeriod = 30 * time.Second
	// One step of tolerance either way, so a phone clock that is 20 seconds out
	// still works. Wider than this and a stolen code stays usable too long.
	totpSkewSteps = 1
)

var (
	ErrTOTPBadCode  = errors.New("security: incorrect code")
	ErrTOTPReplayed = errors.New("security: code already used")
	ErrNoTOTPKey    = errors.New("security: TOTP_ENCRYPTION_KEY is not configured")
)

// base32NoPad is the encoding every authenticator app expects for the secret.
var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a fresh 160-bit secret, base32-encoded.
func NewTOTPSecret() (string, error) {
	buf := make([]byte, 20) // 160 bits, the RFC 4226 recommendation
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("security: totp secret: %w", err)
	}
	return base32NoPad.EncodeToString(buf), nil
}

// totpAt computes the code for one step.
func totpAt(secret []byte, step int64) string {
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))

	mac := hmac.New(sha1.New, secret)
	mac.Write(counter[:])
	sum := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 §5.4).
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	mod := uint32(1)
	for i := 0; i < totpDigits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, value%mod)
}

// VerifyTOTP checks a code and returns the step it matched.
//
// The caller MUST persist the returned step and refuse anything less than or
// equal to it next time. Without that, a code shoulder-surfed or read off a
// phishing page stays valid for its whole 30-second window and can be replayed
// by whoever saw it — the check below only proves the code is current, not that
// it is being used for the first time.
func VerifyTOTP(secretB32, code string, now time.Time, lastUsedStep *int64) (int64, error) {
	secret, err := base32NoPad.DecodeString(strings.ToUpper(strings.TrimSpace(secretB32)))
	if err != nil {
		return 0, fmt.Errorf("security: totp secret: %w", err)
	}

	code = strings.Map(func(r rune) rune {
		// People read codes off a screen in two groups of three and type the
		// space. Refusing "123 456" is a support call, not security.
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, code)
	if len(code) != totpDigits {
		return 0, ErrTOTPBadCode
	}

	current := now.Unix() / int64(totpPeriod.Seconds())
	for delta := int64(-totpSkewSteps); delta <= totpSkewSteps; delta++ {
		step := current + delta
		// Constant-time: a timing difference here leaks how many leading digits
		// of a guess were right, which turns 10^6 guesses into about 60.
		if subtle.ConstantTimeCompare([]byte(totpAt(secret, step)), []byte(code)) == 1 {
			if lastUsedStep != nil && step <= *lastUsedStep {
				return step, ErrTOTPReplayed
			}
			return step, nil
		}
	}
	return 0, ErrTOTPBadCode
}

// OTPAuthURL builds the otpauth:// URI an authenticator app scans.
func OTPAuthURL(issuer, account, secretB32 string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secretB32)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(int(totpPeriod.Seconds())))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// --- secret encryption at rest -------------------------------------------
//
// The secret is a password equivalent: anyone holding it can mint valid codes
// forever. A database dump — a backup on a laptop, a restored copy in staging —
// must not hand over everyone's second factor, so the column stores AES-GCM
// ciphertext keyed by TOTP_ENCRYPTION_KEY, which lives only in the environment.

// TOTPCipher encrypts and decrypts stored secrets.
type TOTPCipher struct{ aead cipher.AEAD }

// NewTOTPCipher derives the key from the configured passphrase.
func NewTOTPCipher(key string) (*TOTPCipher, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoTOTPKey
	}
	// SHA-256 of the passphrase gives a 32-byte AES key from an arbitrary
	// string, so operators are not forced to produce exact key material by hand.
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("security: totp cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("security: totp gcm: %w", err)
	}
	return &TOTPCipher{aead: aead}, nil
}

// Encrypt returns base64 ciphertext with the nonce prefixed.
func (c *TOTPCipher) Encrypt(plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("security: totp nonce: %w", err)
	}
	out := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt reverses Encrypt.
func (c *TOTPCipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("security: totp decode: %w", err)
	}
	n := c.aead.NonceSize()
	if len(raw) < n {
		return "", errors.New("security: totp ciphertext is truncated")
	}
	plain, err := c.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		// GCM authenticates: this means a wrong key or a tampered row, and both
		// deserve to fail closed rather than yield garbage.
		return "", fmt.Errorf("security: totp decrypt: %w", err)
	}
	return string(plain), nil
}

// --- recovery codes -------------------------------------------------------

// NewRecoveryCodes returns n single-use codes and their hashes.
//
// The plaintext is shown ONCE at enrolment and never again; only the hashes are
// stored, for the same reason passwords are hashed. A lost phone is the common
// case and locking an admin out of their own system is a worse outcome than the
// marginal risk these carry.
func NewRecoveryCodes(n int) (plain []string, hashed []string, err error) {
	for i := 0; i < n; i++ {
		code, err := id.Token(10) // Crockford base32, no ambiguous characters
		if err != nil {
			return nil, nil, err
		}
		pretty := code[:5] + "-" + code[5:]
		plain = append(plain, pretty)
		hashed = append(hashed, HashToken(pretty))
	}
	return plain, hashed, nil
}

// MatchRecoveryCode finds a code in the stored hashes, returning its index.
func MatchRecoveryCode(code string, hashes []string) (int, bool) {
	want := HashToken(strings.ToUpper(strings.TrimSpace(code)))
	found := -1
	// Every entry is compared even after a match, so the response time does not
	// reveal how far down the list a code sat.
	for i, h := range hashes {
		if subtle.ConstantTimeCompare([]byte(h), []byte(want)) == 1 {
			found = i
		}
	}
	return found, found >= 0
}
