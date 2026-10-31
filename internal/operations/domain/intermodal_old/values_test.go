package intermodal_old

import (
	"errors"
	"testing"
)

func TestNewContainerNumber_TrimsAndRequiresNonEmpty(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		if _, err := NewContainerNumber(raw); err == nil {
			t.Fatalf("expected rejection of %q", raw)
		} else if !errors.Is(err, ErrInvalidValue) {
			t.Fatalf("expected ErrInvalidValue for %q, got %v", raw, err)
		}
	}

	c, err := NewContainerNumber("  MSKU1234567 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.String() != "MSKU1234567" {
		t.Fatalf("expected trimmed value, got %q", c.String())
	}
	if c.IsZero() {
		t.Fatal("a constructed container number must not be zero")
	}
}

func TestNewChassisNumber_RequiresNonEmpty(t *testing.T) {
	if _, err := NewChassisNumber("   "); err == nil {
		t.Fatal("expected rejection of whitespace-only chassis number")
	}
	if _, err := NewChassisNumber("CHAS-001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainerNumber_Comparable(t *testing.T) {
	a, _ := NewContainerNumber("ABC1")
	b, _ := NewContainerNumber("ABC1")
	c, _ := NewContainerNumber("ABC2")
	if a != b {
		t.Fatal("equal markings must compare equal")
	}
	if a == c {
		t.Fatal("different markings must not compare equal")
	}
}

func TestEnumValidity(t *testing.T) {
	valid := []interface{ Valid() bool }{
		StatusInProgress, PlanLiveLoad, ActionAcquireChassis,
		AssignmentActive, CustodyOpened, ChassisCloseReturned,
		ContainerCloseStreetTurned, TaskCompleted,
	}
	for _, v := range valid {
		if !v.Valid() {
			t.Fatalf("%v should be valid", v)
		}
	}

	invalid := []interface{ Valid() bool }{
		Status("bogus"), PlanType("bogus"), Action("bogus"),
		AssignmentStatus("bogus"), CustodyEpisodeStatus("bogus"),
		ChassisClose("bogus"), ContainerClose("bogus"), TaskStatus("bogus"),
	}
	for _, v := range invalid {
		if v.Valid() {
			t.Fatalf("%v should be invalid", v)
		}
	}
}
