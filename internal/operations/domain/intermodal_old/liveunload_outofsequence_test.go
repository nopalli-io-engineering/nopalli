package intermodal_old

// Live Unload — out-of-sequence errors (rdt: tests/test_live_unload_out_of_sequence.py).

import "testing"

func luAtPickUp(t *testing.T) *Dispatch  { f := newLiveUnload(t); f.toPickUp(t); return f.d }
func luAtDeliver(t *testing.T) *Dispatch { f := newLiveUnload(t); f.toDeliver(t); return f.d }
func luAtHold(t *testing.T) *Dispatch    { f := newLiveUnload(t); f.toHold(t); return f.d }
func luAtDrop(t *testing.T) *Dispatch    { f := newLiveUnload(t); f.toDrop(t); return f.d }

func TestLUOOS_DeliverWhenPickupExpected(t *testing.T) {
	_, err := luAtPickUp(t).DeliverContainer()
	requireOutOfSequence(t, err, ActionPickUpLoadedContainer)
}

func TestLUOOS_PickupWhenDeliverExpected(t *testing.T) {
	_, err := luAtDeliver(t).PickUpContainer()
	requireOutOfSequence(t, err, ActionDeliverLoadedContainer)
}

func TestLUOOS_HoldBeforeItsTurn(t *testing.T) {
	_, err := luAtPickUp(t).Hold()
	requireOutOfSequence(t, err, ActionPickUpLoadedContainer)
}

func TestLUOOS_TakeBeforeItsTurn(t *testing.T) {
	_, err := luAtHold(t).TakeContainer()
	requireOutOfSequence(t, err, ActionHoldForEmptying)
}

func TestLUOOS_DropWhenHoldExpected(t *testing.T) {
	_, err := luAtHold(t).DropContainer()
	requireOutOfSequence(t, err, ActionHoldForEmptying)
}

func TestLUOOS_AcquireChassisDuringCustomerPhase(t *testing.T) {
	_, err := luAtPickUp(t).AcquireChassis(chassisPickupYard, chassis)
	requireOutOfSequence(t, err, ActionPickUpLoadedContainer)
}

func TestLUOOS_ReturnChassisTooEarly(t *testing.T) {
	_, err := luAtPickUp(t).ReturnChassis(chassisDropoffYard)
	requireOutOfSequence(t, err, ActionPickUpLoadedContainer)
}

func TestLUOOS_TakeWhenDropExpected(t *testing.T) {
	_, err := luAtDrop(t).TakeContainer()
	requireOutOfSequence(t, err, ActionDropEmptyContainer)
}
