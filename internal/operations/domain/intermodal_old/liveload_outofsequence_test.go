package intermodal_old

// Live Load — out-of-sequence errors (rdt: tests/test_live_load_out_of_sequence.py).
// Every execution action must be the one the cursor is at; the wrong call
// raises OutOfSequenceError carrying the expected action.

import (
	"errors"
	"testing"
)

// requireOutOfSequence asserts err is an OutOfSequenceError expecting `want`.
func requireOutOfSequence(t *testing.T, err error, want Action) {
	t.Helper()
	var oos *OutOfSequenceError
	if !errors.As(err, &oos) {
		t.Fatalf("err = %v, want *OutOfSequenceError", err)
	}
	if oos.Expected != want {
		t.Fatalf("expected action = %v, want %v", oos.Expected, want)
	}
	wantMsg := "Expected action '" + string(want) + "' not taken"
	if oos.Error() != wantMsg {
		t.Fatalf("message = %q, want %q", oos.Error(), wantMsg)
	}
}

// arrange to the point before a given step (no assertions).
func oosAtPickUp(t *testing.T) *Dispatch {
	t.Helper()
	f := newLiveLoad(t)
	f.toAcquireChassis(t)
	_, err := f.d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
	return f.d
}

func oosAtDeliver(t *testing.T) *Dispatch {
	t.Helper()
	d := oosAtPickUp(t)
	_, err := d.PickUpContainer()
	mustNoErr(t, err)
	return d
}

func oosAtHold(t *testing.T) *Dispatch {
	t.Helper()
	d := oosAtDeliver(t)
	_, err := d.DeliverContainer()
	mustNoErr(t, err)
	return d
}

func oosAtDrop(t *testing.T) *Dispatch {
	t.Helper()
	d := oosAtHold(t)
	_, err := d.Hold()
	mustNoErr(t, err)
	_, err = d.TakeContainer()
	mustNoErr(t, err)
	return d
}

func TestOOS_DeliverWhenPickupExpected(t *testing.T) {
	d := oosAtPickUp(t)
	_, err := d.DeliverContainer()
	requireOutOfSequence(t, err, ActionPickUpEmptyContainer)
}

func TestOOS_PickupWhenDeliverExpected(t *testing.T) {
	d := oosAtDeliver(t)
	_, err := d.PickUpContainer()
	requireOutOfSequence(t, err, ActionDeliverEmptyContainer)
}

func TestOOS_HoldBeforeItsTurn(t *testing.T) {
	d := oosAtPickUp(t) // next is pick up, not hold
	_, err := d.Hold()
	requireOutOfSequence(t, err, ActionPickUpEmptyContainer)
}

func TestOOS_TakeBeforeItsTurn(t *testing.T) {
	d := oosAtHold(t) // next is hold, not take
	_, err := d.TakeContainer()
	requireOutOfSequence(t, err, ActionHoldForLoading)
}

func TestOOS_DropWhenHoldExpected(t *testing.T) {
	d := oosAtHold(t)
	_, err := d.DropContainer()
	requireOutOfSequence(t, err, ActionHoldForLoading)
}

func TestOOS_AcquireChassisDuringCustomerPhase(t *testing.T) {
	d := oosAtPickUp(t)
	_, err := d.AcquireChassis(chassisPickupYard, chassis)
	requireOutOfSequence(t, err, ActionPickUpEmptyContainer)
}

func TestOOS_ReturnChassisTooEarly(t *testing.T) {
	d := oosAtPickUp(t)
	_, err := d.ReturnChassis(chassisDropoffYard)
	requireOutOfSequence(t, err, ActionPickUpEmptyContainer)
}

func TestOOS_TakeWhenDropExpected(t *testing.T) {
	d := oosAtDrop(t)
	_, err := d.TakeContainer()
	requireOutOfSequence(t, err, ActionDropLoadedContainer)
}
