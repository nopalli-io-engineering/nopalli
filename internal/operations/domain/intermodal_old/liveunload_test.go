package intermodal_old

// Live Unload — one test per transition (rdt: tests/test_live_unload.py). Same
// method sequence as live-load, but the plan picks up a loaded container and
// returns an empty one, so the events are the Loaded/Empty mirror image.

import "testing"

var (
	luContainer = mkContainer("MSCU7654321")
	luFull      = Equipment{Chassis: chassis, Container: luContainer}
	luPartial   = Equipment{Chassis: chassis}
)

type liveUnloadFixture struct {
	d    *Dispatch
	move LiveUnload
}

func newLiveUnload(t *testing.T) liveUnloadFixture {
	t.Helper()
	move := LiveUnload{
		ContainerNumber:      luContainer,
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	return liveUnloadFixture{d: Queue(&sourceQuote, move), move: move}
}

func (f liveUnloadFixture) toStart(t *testing.T) {
	t.Helper()
	f.d.Delegate(dispatcher)
	_, err := f.d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, f.d.Approve())
}
func (f liveUnloadFixture) toAcquireChassis(t *testing.T) {
	t.Helper()
	f.toStart(t)
	mustNoErr(t, f.d.Start())
}
func (f liveUnloadFixture) toPickUp(t *testing.T) {
	t.Helper()
	f.toAcquireChassis(t)
	_, err := f.d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)
}
func (f liveUnloadFixture) toDeliver(t *testing.T) {
	t.Helper()
	f.toPickUp(t)
	_, err := f.d.PickUpContainer()
	mustNoErr(t, err)
}
func (f liveUnloadFixture) toHold(t *testing.T) {
	t.Helper()
	f.toDeliver(t)
	_, err := f.d.DeliverContainer()
	mustNoErr(t, err)
}
func (f liveUnloadFixture) toTake(t *testing.T) {
	t.Helper()
	f.toHold(t)
	_, err := f.d.Hold()
	mustNoErr(t, err)
}
func (f liveUnloadFixture) toDrop(t *testing.T) {
	t.Helper()
	f.toTake(t)
	_, err := f.d.TakeContainer()
	mustNoErr(t, err)
}
func (f liveUnloadFixture) toReturnChassis(t *testing.T) {
	t.Helper()
	f.toDrop(t)
	_, err := f.d.DropContainer()
	mustNoErr(t, err)
}

func TestLiveUnload_Queue(t *testing.T) {
	f := newLiveUnload(t)
	if f.d.Status() != StatusQueued {
		t.Fatalf("status = %v, want queued", f.d.Status())
	}
	if oneEvent[DispatchQueued](t, f.d.Events()).SourceQuoteID != sourceQuote {
		t.Fatal("DispatchQueued payload mismatch")
	}
}

func TestLiveUnload_AcquireChassis(t *testing.T) {
	f := newLiveUnload(t)
	f.toAcquireChassis(t)
	track := trackFrom(f.d)
	id, err := f.d.AcquireChassis(chassisPickupYard, chassis)
	mustNoErr(t, err)

	requireEquipment(t, f.d, luPartial)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisAcquired")
	acq := oneEvent[ChassisAcquired](t, ev)
	if acq.FacilityID != chassisPickupYard || acq.ChassisNumber != chassis {
		t.Fatal("ChassisAcquired payload mismatch")
	}
}

func TestLiveUnload_PickUpLoadedContainer(t *testing.T) {
	f := newLiveUnload(t)
	f.toPickUp(t)
	track := trackFrom(f.d)
	id, err := f.d.PickUpContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, luFull)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "LoadedContainerPickedUp")
	picked := oneEvent[LoadedContainerPickedUp](t, ev)
	if picked.FacilityID != f.move.ContainerPickupRamp || picked.ContainerNumber != luContainer {
		t.Fatal("LoadedContainerPickedUp payload mismatch")
	}
}

func TestLiveUnload_DeliverLoadedContainer(t *testing.T) {
	f := newLiveUnload(t)
	f.toDeliver(t)
	track := trackFrom(f.d)
	id, err := f.d.DeliverContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, luFull)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "LoadedContainerDelivered")
	del := oneEvent[LoadedContainerDelivered](t, ev)
	if del.FacilityID != f.move.DockDoor || del.ContainerNumber != luContainer {
		t.Fatal("LoadedContainerDelivered payload mismatch")
	}
}

func TestLiveUnload_HoldForEmptying(t *testing.T) {
	f := newLiveUnload(t)
	f.toHold(t)
	track := trackFrom(f.d)
	id, err := f.d.Hold()
	mustNoErr(t, err)

	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "DispatchHeldForEmptying")
	if oneEvent[DispatchHeldForEmptying](t, ev).FacilityID != f.move.DockDoor {
		t.Fatal("DispatchHeldForEmptying payload mismatch")
	}
}

func TestLiveUnload_TakeReleasesHold(t *testing.T) {
	f := newLiveUnload(t)
	f.toTake(t)
	track := trackFrom(f.d)
	id, err := f.d.TakeContainer()
	mustNoErr(t, err)

	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "EmptyContainerTaken", "DispatchReleased")
	taken := oneEvent[EmptyContainerTaken](t, ev)
	if taken.FacilityID != f.move.DockDoor || taken.ContainerNumber != luContainer {
		t.Fatal("EmptyContainerTaken payload mismatch")
	}
	if oneEvent[DispatchReleased](t, ev).FacilityID != f.move.DockDoor {
		t.Fatal("DispatchReleased payload mismatch")
	}
}

func TestLiveUnload_DropEmptyClosesCustody(t *testing.T) {
	f := newLiveUnload(t)
	f.toDrop(t)
	track := trackFrom(f.d)
	id, err := f.d.DropContainer()
	mustNoErr(t, err)

	requireEquipment(t, f.d, luPartial) // chassis still open
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "EmptyContainerDropped")
	ld := oneEvent[EmptyContainerDropped](t, ev)
	if ld.FacilityID != f.move.ContainerDropoffRamp || ld.ContainerNumber != luContainer {
		t.Fatal("EmptyContainerDropped payload mismatch")
	}
	if f.d.Status() != StatusInProgress {
		t.Fatalf("status = %v, want in_progress (chassis still open)", f.d.Status())
	}
}

func TestLiveUnload_ReturnChassisCompletes(t *testing.T) {
	f := newLiveUnload(t)
	f.toReturnChassis(t)
	track := trackFrom(f.d)
	id, err := f.d.ReturnChassis(chassisDropoffYard)
	mustNoErr(t, err)

	requireNoEquipment(t, f.d)
	requireNoLiveAssignee(t, f.d)
	requireNoAssignee(t, f.d)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisReturned", "DispatchCompleted")
	ret := oneEvent[ChassisReturned](t, ev)
	if ret.FacilityID != chassisDropoffYard || ret.ChassisNumber != chassis {
		t.Fatal("ChassisReturned payload mismatch")
	}
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
}
