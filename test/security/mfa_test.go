package security_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	sec "github.com/stevenwilliam/healthy_catering/internal/platform/security"
)

// The challenge token handed out after a correct password must not be a
// session. If this fails, 2FA is decorative: an attacker with the password
// holds a working token and never needs the phone.
func TestMFAChallengeTokenIsNotASession(t *testing.T) {
	signer := sec.NewTokenSigner("a-test-signing-key-of-adequate-length",
		"", "evermore-test", 15*time.Minute, time.Now)
	userID := uuid.New()

	challenge, err := signer.IssueMFAChallenge(userID, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	// It parses — it is a real, correctly signed token — but it is MARKED, and
	// that mark is what RequireAuth refuses.
	claims, err := signer.Parse(challenge)
	if err != nil {
		t.Fatalf("the challenge should be a valid token: %v", err)
	}
	if claims.Purpose != sec.PurposeMFA {
		t.Fatalf("the challenge is not marked as such: purpose=%q", claims.Purpose)
	}
	if claims.Role != "" {
		t.Fatalf("the challenge carries a role (%q) — it must grant nothing", claims.Role)
	}

	// And the reverse: a full access token must not complete somebody's
	// pending challenge.
	access, _, err := signer.Issue(sec.SubjectStaff, userID, sec.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.ParseMFAChallenge(access); err == nil {
		t.Fatal("an access token was accepted as an MFA challenge")
	}
}

func TestMFAChallengeExpires(t *testing.T) {
	base := time.Now()
	clock := base
	signer := sec.NewTokenSigner("a-test-signing-key-of-adequate-length",
		"", "evermore-test", 15*time.Minute, func() time.Time { return clock })

	challenge, err := signer.IssueMFAChallenge(uuid.New(), 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.ParseMFAChallenge(challenge); err != nil {
		t.Fatalf("should be valid now: %v", err)
	}

	clock = base.Add(6 * time.Minute)
	if _, err := signer.ParseMFAChallenge(challenge); err == nil {
		t.Fatal("an expired challenge was accepted")
	}
}

// A TOTP code must be spendable exactly once even when two requests race —
// the guard is in SQL, not in Go, and this proves it against a real database.
func TestTOTPCodeCannotBeSpentTwiceConcurrently(t *testing.T) {
	ctx := context.Background()
	userID := seedTOTPUser(t)

	const step = int64(58000000)
	const racers = 8

	var wg sync.WaitGroup
	won := make(chan bool, racers)
	start := make(chan struct{})

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := db.ExecContext(ctx, `
				UPDATE user_totp SET last_used_step = $1
				 WHERE user_id = $2
				   AND (last_used_step IS NULL OR last_used_step < $1)`, step, userID)
			if err != nil {
				return
			}
			n, _ := res.RowsAffected()
			won <- n == 1
		}()
	}
	close(start)
	wg.Wait()
	close(won)

	accepted := 0
	for ok := range won {
		if ok {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("the same code was accepted %d times — replay protection does not hold", accepted)
	}
}

// A recovery code is single-use, and the removal has to be atomic for the same
// reason: two people, or one person clicking twice, must not both get in.
func TestRecoveryCodeIsConsumedExactlyOnce(t *testing.T) {
	ctx := context.Background()
	userID := seedTOTPUser(t)

	plain, hashes, err := sec.NewRecoveryCodes(4)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(hashes)
	if _, err := db.ExecContext(ctx,
		`UPDATE user_totp SET recovery_codes = $1::jsonb, confirmed_at = now() WHERE user_id = $2`,
		string(payload), userID); err != nil {
		t.Fatal(err)
	}

	idx, ok := sec.MatchRecoveryCode(plain[2], hashes)
	if !ok {
		t.Fatal("the code did not match its own hash")
	}
	target := hashes[idx]

	const racers = 6
	var wg sync.WaitGroup
	won := make(chan bool, racers)
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := db.ExecContext(ctx, `
				UPDATE user_totp
				   SET recovery_codes = (
				         SELECT COALESCE(jsonb_agg(c), '[]'::jsonb)
				           FROM jsonb_array_elements(recovery_codes) AS c
				          WHERE c <> to_jsonb($1::text))
				 WHERE user_id = $2
				   AND recovery_codes @> to_jsonb($1::text)`, target, userID)
			if err != nil {
				return
			}
			n, _ := res.RowsAffected()
			won <- n == 1
		}()
	}
	close(start)
	wg.Wait()
	close(won)

	accepted := 0
	for ok := range won {
		if ok {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("one recovery code was spent %d times", accepted)
	}

	var left int
	if err := db.QueryRowContext(ctx,
		`SELECT jsonb_array_length(recovery_codes) FROM user_totp WHERE user_id = $1`,
		userID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 3 {
		t.Fatalf("want 3 codes left, got %d", left)
	}
}

// The secret must never be readable from a database dump.
func TestStoredTOTPSecretIsNotPlaintext(t *testing.T) {
	ctx := context.Background()
	userID := seedTOTPUser(t)

	cipher, err := sec.NewTOTPCipher("test-key-from-the-environment")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := sec.NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := cipher.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE user_totp SET secret_cipher = $1 WHERE user_id = $2`, ct, userID); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := db.QueryRowContext(ctx,
		`SELECT secret_cipher FROM user_totp WHERE user_id = $1`, userID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == secret {
		t.Fatal("the secret is stored in the clear")
	}
	back, err := cipher.Decrypt(stored)
	if err != nil || back != secret {
		t.Fatalf("the stored secret does not round-trip: %v", err)
	}
}

// seedTOTPUser creates a throwaway staff account with a TOTP row.
func seedTOTPUser(t *testing.T) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	userID := uuid.New()
	email := "totp-" + userID.String()[:8] + "@evermore.test"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO app_user (id, email, password_hash, full_name, is_active, email_verified_at)
		VALUES ($1, $2, 'x', 'TOTP Test', true, now())`, userID, email); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_totp (user_id, secret_cipher) VALUES ($1, 'placeholder')`,
		userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM app_user WHERE id = $1`, userID)
	})
	return userID
}
