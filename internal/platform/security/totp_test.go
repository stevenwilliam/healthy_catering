package security

import (
	"testing"
	"time"
)

// The RFC 6238 Appendix B secret, "12345678901234567890" in base32.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// TestTOTPAgainstRFC6238 pins the implementation to the published vectors.
//
// This is the test that matters: a home-grown TOTP that is self-consistent but
// disagrees with the RFC produces codes no phone accepts, and the failure looks
// like "the user typed it wrong" from every angle.
func TestTOTPAgainstRFC6238(t *testing.T) {
	// The RFC prints 8-digit codes; these are the trailing 6, which is what a
	// 6-digit authenticator shows for the same instant.
	vectors := []struct {
		unix int64
		code string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, v := range vectors {
		step := v.unix / 30
		secret, err := base32NoPad.DecodeString(rfcSecret)
		if err != nil {
			t.Fatal(err)
		}
		if got := totpAt(secret, step); got != v.code {
			t.Errorf("t=%d: got %s, want %s", v.unix, got, v.code)
		}
	}
}

func TestTOTPAcceptsClockSkewButNotMore(t *testing.T) {
	now := time.Unix(1234567890, 0)
	secret, _ := base32NoPad.DecodeString(rfcSecret)
	current := now.Unix() / 30

	for _, delta := range []int64{-1, 0, 1} {
		code := totpAt(secret, current+delta)
		if _, err := VerifyTOTP(rfcSecret, code, now, nil); err != nil {
			t.Errorf("delta %d should be accepted: %v", delta, err)
		}
	}
	for _, delta := range []int64{-2, 2, 10} {
		code := totpAt(secret, current+delta)
		if _, err := VerifyTOTP(rfcSecret, code, now, nil); err == nil {
			t.Errorf("delta %d should be refused", delta)
		}
	}
}

// TestTOTPCodeCannotBeReplayed is the property the last_used_step column exists
// for. Without it a code stays valid for its whole window.
func TestTOTPCodeCannotBeReplayed(t *testing.T) {
	now := time.Unix(1234567890, 0)
	secret, _ := base32NoPad.DecodeString(rfcSecret)
	code := totpAt(secret, now.Unix()/30)

	step, err := VerifyTOTP(rfcSecret, code, now, nil)
	if err != nil {
		t.Fatalf("first use must succeed: %v", err)
	}

	if _, err := VerifyTOTP(rfcSecret, code, now, &step); err != ErrTOTPReplayed {
		t.Fatalf("second use must be refused, got %v", err)
	}

	// An OLDER code inside the skew window is refused too, not just the exact
	// one already seen — otherwise replay is defeated by waiting 30 seconds.
	older := totpAt(secret, now.Unix()/30-1)
	if _, err := VerifyTOTP(rfcSecret, older, now, &step); err != ErrTOTPReplayed {
		t.Fatalf("an older step must be refused, got %v", err)
	}
}

func TestTOTPToleratesHumanTyping(t *testing.T) {
	now := time.Unix(1234567890, 0)
	secret, _ := base32NoPad.DecodeString(rfcSecret)
	code := totpAt(secret, now.Unix()/30)

	spaced := code[:3] + " " + code[3:]
	if _, err := VerifyTOTP(rfcSecret, spaced, now, nil); err != nil {
		t.Errorf("a space in the middle must still work: %v", err)
	}
	if _, err := VerifyTOTP(rfcSecret, "12345", now, nil); err == nil {
		t.Error("a short code must be refused")
	}
}

func TestTOTPSecretIsEncryptedAtRest(t *testing.T) {
	c, err := NewTOTPCipher("a-passphrase-from-the-environment")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}

	ct, err := c.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if ct == secret {
		t.Fatal("the stored value is the secret in the clear")
	}

	back, err := c.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if back != secret {
		t.Fatalf("round trip changed the secret: %q vs %q", back, secret)
	}

	// Encrypting twice must not produce the same ciphertext, or a dump reveals
	// which users share... nothing useful, but it also means the nonce is fixed.
	again, _ := c.Encrypt(secret)
	if again == ct {
		t.Fatal("the nonce is not random — ciphertext repeats")
	}

	// The wrong key must FAIL, not return garbage.
	other, _ := NewTOTPCipher("a-different-passphrase")
	if _, err := other.Decrypt(ct); err == nil {
		t.Fatal("the wrong key decrypted the secret")
	}
}

func TestNoTOTPKeyIsAnError(t *testing.T) {
	if _, err := NewTOTPCipher("  "); err != ErrNoTOTPKey {
		t.Fatalf("an empty key must be refused, got %v", err)
	}
}

func TestRecoveryCodesAreStoredHashed(t *testing.T) {
	plain, hashed, err := NewRecoveryCodes(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 8 || len(hashed) != 8 {
		t.Fatalf("want 8 codes, got %d/%d", len(plain), len(hashed))
	}
	for i, p := range plain {
		if p == hashed[i] {
			t.Fatal("the code is stored in the clear")
		}
	}

	seen := map[string]bool{}
	for _, p := range plain {
		if seen[p] {
			t.Fatalf("duplicate recovery code %q", p)
		}
		seen[p] = true
	}

	idx, ok := MatchRecoveryCode(plain[3], hashed)
	if !ok || idx != 3 {
		t.Fatalf("code 3 should match at index 3, got %d/%v", idx, ok)
	}
	if _, ok := MatchRecoveryCode("ZZZZZ-ZZZZZ", hashed); ok {
		t.Fatal("an unknown code matched")
	}
	// Case and spacing are forgiven, since these get written down on paper.
	if _, ok := MatchRecoveryCode("  "+lower(plain[0])+"  ", hashed); !ok {
		t.Fatal("a lower-cased code should still match")
	}
}

func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + 32
		}
	}
	return string(out)
}
