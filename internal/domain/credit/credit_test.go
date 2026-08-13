package credit

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The matrix from docs/01 §5.2.

var (
	pkgID  = uuid.MustParse("00000000-0000-7000-8000-0000000000c1")
	custID = uuid.MustParse("00000000-0000-7000-8000-0000000000c2")
	delivA = uuid.MustParse("00000000-0000-7000-8000-0000000000d1")
	staff  = uuid.MustParse("00000000-0000-7000-8000-0000000000e1")
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func activePkg() Package {
	exp := day("2026-09-30")
	act := day("2026-09-01")
	return Package{ID: pkgID, CustomerID: custID, Status: Active, Credits: 20,
		ActivatedAt: &act, ExpiresAt: &exp}
}

func purchase(n int) []Entry {
	return []Entry{{PackageID: pkgID, CustomerID: custID, Type: Purchase, Qty: n}}
}

func TestBalance(t *testing.T) {
	entries := append(purchase(20),
		Entry{Type: Redeem, Qty: -1}, Entry{Type: Redeem, Qty: -1}, Entry{Type: Refund, Qty: 1})
	if got := Balance(entries); got != 19 {
		t.Errorf("balance = %d, want 19", got)
	}
	if got := Balance(nil); got != 0 {
		t.Errorf("empty balance = %d, want 0", got)
	}
}

// One credit buys one meal whatever it contains (D-32). Nothing counts foods.
func TestRedeemOneSpendsExactlyOneCreditPerMeal(t *testing.T) {
	got, err := RedeemOne(activePkg(), purchase(20), delivA, day("2026-09-10"), day("2026-09-05"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Qty != -1 {
		t.Errorf("qty = %d, want exactly -1 per meal", got.Qty)
	}
	if got.Type != Redeem {
		t.Errorf("type = %s, want REDEEM", got.Type)
	}
	// The idempotency key: a retry cannot spend a second credit.
	if got.ReferenceType != "delivery" || got.ReferenceID != delivA {
		t.Errorf("reference = %s/%s, want delivery/%s", got.ReferenceType, got.ReferenceID, delivA)
	}
}

func TestRedeemRefusesAtZero(t *testing.T) {
	spent := append(purchase(1), Entry{Type: Redeem, Qty: -1})
	_, err := RedeemOne(activePkg(), spent, delivA, day("2026-09-10"), day("2026-09-05"))
	if !errors.Is(err, ErrInsufficient) {
		t.Fatalf("err = %v, want ErrInsufficient", err)
	}
}

// Each non-Active status gets its OWN error, because each needs a different
// message. A customer who just spent their last credit must not be told we are
// still waiting for their bank transfer.
func TestRedeemRefusesInactivePackageWithTheRightReason(t *testing.T) {
	tests := []struct {
		status Status
		want   error
	}{
		{Pending, ErrNotActive},
		{Cancelled, ErrNotActive},
		{Exhausted, ErrInsufficient},
		{Expired, ErrExpired},
	}
	for _, tc := range tests {
		p := activePkg()
		p.Status = tc.status
		_, err := RedeemOne(p, purchase(20), delivA, day("2026-09-10"), day("2026-09-05"))
		if !errors.Is(err, tc.want) {
			t.Errorf("status %s: err = %v, want %v", tc.status, err, tc.want)
		}
	}
}

func TestRedeemRefusesAfterExpiry(t *testing.T) {
	_, err := RedeemOne(activePkg(), purchase(20), delivA, day("2026-09-10"), day("2026-10-01"))
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("err = %v, want ErrExpired", err)
	}
}

// D-27: a credit redeemed on the last valid day must not schedule a delivery a
// month later, or the package never really expires.
func TestRedeemRefusesDeliveryAfterExpiry(t *testing.T) {
	_, err := RedeemOne(activePkg(), purchase(20), delivA, day("2026-10-15"), day("2026-09-30"))
	if !errors.Is(err, ErrAfterExpiry) {
		t.Fatalf("err = %v, want ErrAfterExpiry", err)
	}
	// The last valid day itself is allowed.
	if _, err := RedeemOne(activePkg(), purchase(20), delivA, day("2026-09-30"), day("2026-09-30")); err != nil {
		t.Errorf("delivery on the expiry date must be allowed: %v", err)
	}
}

// PROMPT §8.3: a skip before the cut-off returns the credit; after it does not.
func TestSkipReturnsCreditOnlyBeforeCutoff(t *testing.T) {
	e, ok := ReturnForSkip(activePkg(), delivA, true, day("2026-09-05"))
	if !ok || e.Qty != 1 || e.Type != Refund {
		t.Errorf("before cut-off: got %+v ok=%v, want a +1 REFUND", e, ok)
	}
	if _, ok := ReturnForSkip(activePkg(), delivA, false, day("2026-09-05")); ok {
		t.Error("after cut-off the credit must NOT be returned — the food is committed")
	}
}

func TestExpireRemainderForfeitsBalance(t *testing.T) {
	entries := append(purchase(20), Entry{Type: Redeem, Qty: -3})
	e, ok := ExpireRemainder(activePkg(), entries, day("2026-10-01"))
	if !ok {
		t.Fatal("expected an EXPIRE entry")
	}
	if e.Qty != -17 {
		t.Errorf("qty = %d, want -17 to zero the balance", e.Qty)
	}
	if got := Balance(append(entries, e)); got != 0 {
		t.Errorf("post-expiry balance = %d, want 0", got)
	}
	// Nothing to expire when already empty.
	spent := append(purchase(1), Entry{Type: Redeem, Qty: -1})
	if _, ok := ExpireRemainder(activePkg(), spent, day("2026-10-01")); ok {
		t.Error("an empty package needs no EXPIRE entry")
	}
}

// D-28: an extension compensates, it never deletes. The ledger is append-only
// and the balance history has to reconcile.
func TestReverseExpiryCompensatesRatherThanDeletes(t *testing.T) {
	entries := append(purchase(20), Entry{Type: Redeem, Qty: -3}, Entry{Type: Expire, Qty: -17})
	if got := Balance(entries); got != 0 {
		t.Fatalf("precondition: balance = %d, want 0", got)
	}

	adj, err := ReverseExpiry(activePkg(), entries, "customer was hospitalised", staff, day("2026-10-02"))
	if err != nil {
		t.Fatal(err)
	}
	if adj.Type != Adjustment || adj.Qty != 17 {
		t.Errorf("got %s %+d, want ADJUSTMENT +17", adj.Type, adj.Qty)
	}
	if got := Balance(append(entries, adj)); got != 17 {
		t.Errorf("restored balance = %d, want 17", got)
	}
	// The EXPIRE entry is still there — history was not rewritten.
	found := false
	for _, e := range entries {
		if e.Type == Expire {
			found = true
		}
	}
	if !found {
		t.Error("the EXPIRE entry must survive the reversal")
	}
}

func TestReverseExpiryRequiresReason(t *testing.T) {
	entries := append(purchase(20), Entry{Type: Expire, Qty: -20})
	if _, err := ReverseExpiry(activePkg(), entries, "", staff, day("2026-10-02")); !errors.Is(err, ErrReasonRequired) {
		t.Errorf("err = %v, want ErrReasonRequired", err)
	}
}

// No refunds (D-31) means compensation is in credits, and it always has a
// reason attached.
func TestGoodwillRequiresReason(t *testing.T) {
	if _, err := Goodwill(activePkg(), 2, "", staff, day("2026-09-05")); !errors.Is(err, ErrReasonRequired) {
		t.Errorf("err = %v, want ErrReasonRequired", err)
	}
	e, err := Goodwill(activePkg(), 2, "kitchen could not fulfil", staff, day("2026-09-05"))
	if err != nil || e.Qty != 2 || e.Type != Adjustment {
		t.Errorf("got %+v, %v", e, err)
	}
}

func TestEntryValidateMirrorsTheCheckConstraints(t *testing.T) {
	tests := []struct {
		name    string
		e       Entry
		wantErr bool
	}{
		{"purchase positive", Entry{Type: Purchase, Qty: 20}, false},
		{"purchase negative", Entry{Type: Purchase, Qty: -20}, true},
		{"redeem negative", Entry{Type: Redeem, Qty: -1}, false},
		{"redeem positive", Entry{Type: Redeem, Qty: 1}, true},
		{"expire positive", Entry{Type: Expire, Qty: 5}, true},
		{"refund positive", Entry{Type: Refund, Qty: 1}, false},
		{"adjustment needs a note", Entry{Type: Adjustment, Qty: 1}, true},
		{"adjustment with a note", Entry{Type: Adjustment, Qty: 1, Note: "why"}, false},
		{"zero movement", Entry{Type: Purchase, Qty: 0}, true},
		{"unknown type", Entry{Type: "NOPE", Qty: 1}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.e.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNextStatus(t *testing.T) {
	tests := []struct {
		name    string
		pkg     Package
		entries []Entry
		now     string
		want    Status
	}{
		{"active with credits", activePkg(), purchase(20), "2026-09-05", Active},
		{"exhausted at zero", activePkg(), append(purchase(1), Entry{Type: Redeem, Qty: -1}), "2026-09-05", Exhausted},
		{"expired by date", activePkg(), purchase(20), "2026-10-01", Expired},
		{"expiry beats exhaustion", activePkg(), append(purchase(1), Entry{Type: Redeem, Qty: -1}), "2026-10-01", Expired},
		{"pending stays pending", Package{Status: Pending}, nil, "2026-09-05", Pending},
		{"cancelled stays cancelled", Package{Status: Cancelled}, nil, "2026-09-05", Cancelled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextStatus(tc.pkg, tc.entries, day(tc.now)); got != tc.want {
				t.Errorf("status = %s, want %s", got, tc.want)
			}
		})
	}
}

// D-14: the active period starts on payment verification, not purchase.
func TestActivateStartsOnVerification(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	verified := time.Date(2026, 9, 1, 14, 30, 0, 0, jkt)
	activated, expires := Activate(activePkg(), 30, verified, jkt)
	if !activated.Equal(verified) {
		t.Errorf("activated = %v, want the verification instant", activated)
	}
	if got := expires.Format("2006-01-02"); got != "2026-10-01" {
		t.Errorf("expires = %s, want 2026-10-01 (30 days from verification)", got)
	}
}
