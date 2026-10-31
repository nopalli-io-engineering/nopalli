package intermodal_old

import (
	"errors"
	"testing"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

func TestNewDispatchID_NonZeroAndUnique(t *testing.T) {
	a := NewDispatchID()
	b := NewDispatchID()
	if a.IsZero() {
		t.Fatal("a freshly minted id must not be the zero value")
	}
	if a == b {
		t.Fatal("two minted ids must differ")
	}
}

func TestParseDispatchID_RoundTrip(t *testing.T) {
	orig := NewDispatchID()
	got, err := ParseDispatchID(orig.String())
	if err != nil {
		t.Fatalf("parse round-trip: unexpected error: %v", err)
	}
	if got != orig {
		t.Fatalf("round-trip mismatch: got %v, want %v", got, orig)
	}
}

func TestParseDispatchID_Rejects(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"
	for _, raw := range []string{"", "not-a-uuid", zeroUUID} {
		_, err := ParseDispatchID(raw)
		if err == nil {
			t.Fatalf("expected error for %q", raw)
		}
		if !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("expected ErrInvalidID for %q, got %v", raw, err)
		}
	}
}

// Distinct id types are a compile-time guarantee: a line like
//
//	var _ DriverID = NewTruckID()
//
// does not compile. We assert here only that the embedded value plumbing works
// through the named type.
func TestTypedID_ExposesUUID(t *testing.T) {
	id := NewDriverID()
	if id.UUID() != id.UUID() {
		t.Fatal("UUID accessor must be stable")
	}
	if id.String() == "" {
		t.Fatal("a non-zero id must stringify")
	}
}
