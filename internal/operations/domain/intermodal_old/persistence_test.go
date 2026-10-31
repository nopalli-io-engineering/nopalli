package intermodal_old

import (
	"reflect"
	"testing"
)

// Snapshot -> Reconstitute -> Snapshot must be a faithful round-trip, and a
// reconstituted aggregate must carry no pending domain events.

func TestSnapshotRoundTrip_Queued(t *testing.T) {
	d := newDispatch()
	assertRoundTrips(t, d)
}

func TestSnapshotRoundTrip_InProgress(t *testing.T) {
	d := newDispatch()
	startDispatch(t, d)
	_, err := d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
	_, err = d.PickUpContainer()
	mustNoErr(t, err)
	assertRoundTrips(t, d)
}

func TestSnapshotRoundTrip_Completed(t *testing.T) {
	d := newDispatch()
	completeDispatch(t, d)
	assertRoundTrips(t, d)
}

func TestReconstituteRaisesNoEvents(t *testing.T) {
	d := newDispatch()
	completeDispatch(t, d)
	got := Reconstitute(d.Snapshot())
	if n := len(got.Events()); n != 0 {
		t.Fatalf("reconstituted aggregate has %d pending events, want 0", n)
	}
}

func assertRoundTrips(t *testing.T, d *Dispatch) {
	t.Helper()
	snap := d.Snapshot()
	again := Reconstitute(snap).Snapshot()
	if !reflect.DeepEqual(snap, again) {
		t.Fatalf("snapshot round-trip mismatch:\n first: %+v\nsecond: %+v", snap, again)
	}
}
