package intermodal_old

// Premounted Live Load / Live Unload — out-of-sequence errors (rdt:
// tests/test_premounted_live_load_out_of_sequence.py,
// test_premounted_live_unload_out_of_sequence.py).

import "testing"

// --- premounted live load ------------------------------------------------- //

func pmLoadAtPickUp(t *testing.T) *Dispatch  { f := newPremountedLoad(t); f.toPickUp(t); return f.d }
func pmLoadAtDeliver(t *testing.T) *Dispatch { f := newPremountedLoad(t); f.toDeliver(t); return f.d }
func pmLoadAtHold(t *testing.T) *Dispatch    { f := newPremountedLoad(t); f.toHold(t); return f.d }
func pmLoadAtDrop(t *testing.T) *Dispatch    { f := newPremountedLoad(t); f.toDrop(t); return f.d }

func TestPMLoadOOS_DeliverWhenPickupExpected(t *testing.T) {
	_, err := pmLoadAtPickUp(t).DeliverContainer()
	requireOutOfSequence(t, err, ActionPickUpEmptyPremountedContainer)
}

func TestPMLoadOOS_PickupWhenDeliverExpected(t *testing.T) {
	_, err := pmLoadAtDeliver(t).PickUpPremountedContainer(chassis)
	requireOutOfSequence(t, err, ActionDeliverEmptyContainer)
}

func TestPMLoadOOS_HoldBeforeItsTurn(t *testing.T) {
	_, err := pmLoadAtPickUp(t).Hold()
	requireOutOfSequence(t, err, ActionPickUpEmptyPremountedContainer)
}

func TestPMLoadOOS_TakeBeforeItsTurn(t *testing.T) {
	_, err := pmLoadAtHold(t).TakeContainer()
	requireOutOfSequence(t, err, ActionHoldForLoading)
}

func TestPMLoadOOS_DropWhenHoldExpected(t *testing.T) {
	_, err := pmLoadAtHold(t).DropPremountedContainer()
	requireOutOfSequence(t, err, ActionHoldForLoading)
}

func TestPMLoadOOS_TakeWhenDropExpected(t *testing.T) {
	_, err := pmLoadAtDrop(t).TakeContainer()
	requireOutOfSequence(t, err, ActionDropLoadedPremountedContainer)
}

// --- premounted live unload ----------------------------------------------- //

func pmUnloadAtPickUp(t *testing.T) *Dispatch { f := newPremountedUnload(t); f.toPickUp(t); return f.d }
func pmUnloadAtDeliver(t *testing.T) *Dispatch {
	f := newPremountedUnload(t)
	f.toDeliver(t)
	return f.d
}
func pmUnloadAtHold(t *testing.T) *Dispatch { f := newPremountedUnload(t); f.toHold(t); return f.d }
func pmUnloadAtDrop(t *testing.T) *Dispatch { f := newPremountedUnload(t); f.toDrop(t); return f.d }

func TestPMUnloadOOS_DeliverWhenPickupExpected(t *testing.T) {
	_, err := pmUnloadAtPickUp(t).DeliverContainer()
	requireOutOfSequence(t, err, ActionPickUpLoadedPremountedContainer)
}

func TestPMUnloadOOS_PickupWhenDeliverExpected(t *testing.T) {
	_, err := pmUnloadAtDeliver(t).PickUpPremountedContainer(chassis)
	requireOutOfSequence(t, err, ActionDeliverLoadedContainer)
}

func TestPMUnloadOOS_HoldBeforeItsTurn(t *testing.T) {
	_, err := pmUnloadAtPickUp(t).Hold()
	requireOutOfSequence(t, err, ActionPickUpLoadedPremountedContainer)
}

func TestPMUnloadOOS_TakeBeforeItsTurn(t *testing.T) {
	_, err := pmUnloadAtHold(t).TakeContainer()
	requireOutOfSequence(t, err, ActionHoldForEmptying)
}

func TestPMUnloadOOS_DropWhenHoldExpected(t *testing.T) {
	_, err := pmUnloadAtHold(t).DropPremountedContainer()
	requireOutOfSequence(t, err, ActionHoldForEmptying)
}

func TestPMUnloadOOS_TakeWhenDropExpected(t *testing.T) {
	_, err := pmUnloadAtDrop(t).TakeContainer()
	requireOutOfSequence(t, err, ActionDropEmptyPremountedContainer)
}
