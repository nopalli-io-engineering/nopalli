package intermodal

// Premounted Live Load / Live Unload — one test per transition (rdt:
// tests/test_premounted_live_load.py, test_premounted_live_unload.py). The
// container arrives already mounted, so there is no separate acquire/return
// chassis step: pick-up and drop are the premounted variants that open/close
// the chassis episode alongside the container.

import "testing"

var (
	pmContainer = mkContainer("TCLU0000001")
	pmFull      = Equipment{Chassis: chassis, Container: pmContainer}
)

// premountedFixture works for either plan: it keeps the facilities and container
// directly so load/unload tests share the advance chain.
type premountedFixture struct {
	d                                 *Dispatch
	pickupRamp, dropoffRamp, dockDoor FacilityID
}

func newPremountedLoad(t *testing.T) premountedFixture {
	t.Helper()
	move := LiveLoad{
		ContainerNumber:      pmContainer,
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
		Premounted:           true,
	}
	return premountedFixture{Queue(&sourceQuote, move), move.ContainerPickupRamp, move.ContainerDropoffRamp, move.DockDoor}
}

func newPremountedUnload(t *testing.T) premountedFixture {
	t.Helper()
	move := LiveUnload{
		ContainerNumber:      pmContainer,
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
		Premounted:           true,
	}
	return premountedFixture{Queue(&sourceQuote, move), move.ContainerPickupRamp, move.ContainerDropoffRamp, move.DockDoor}
}

func (f premountedFixture) toPickUp(t *testing.T) {
	t.Helper()
	f.d.Delegate(dispatcher)
	_, err := f.d.Assign(&driver, &truck)
	mustNoErr(t, err)
	mustNoErr(t, f.d.Approve())
	mustNoErr(t, f.d.Start())
}
func (f premountedFixture) toDeliver(t *testing.T) {
	t.Helper()
	f.toPickUp(t)
	_, err := f.d.PickUpPremountedContainer(chassis)
	mustNoErr(t, err)
}
func (f premountedFixture) toHold(t *testing.T) {
	t.Helper()
	f.toDeliver(t)
	_, err := f.d.DeliverContainer()
	mustNoErr(t, err)
}
func (f premountedFixture) toTake(t *testing.T) {
	t.Helper()
	f.toHold(t)
	_, err := f.d.Hold()
	mustNoErr(t, err)
}
func (f premountedFixture) toDrop(t *testing.T) {
	t.Helper()
	f.toTake(t)
	_, err := f.d.TakeContainer()
	mustNoErr(t, err)
}

// --- premounted live load ------------------------------------------------- //

func TestPremountedLoad_PickUp(t *testing.T) {
	f := newPremountedLoad(t)
	f.toPickUp(t)
	track := trackFrom(f.d)
	id, err := f.d.PickUpPremountedContainer(chassis)
	mustNoErr(t, err)

	requireEquipment(t, f.d, pmFull)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisAcquired", "EmptyContainerPickedUp")
	acq := oneEvent[ChassisAcquired](t, ev)
	if acq.FacilityID != f.pickupRamp || acq.ChassisNumber != chassis {
		t.Fatal("ChassisAcquired payload mismatch")
	}
	picked := oneEvent[EmptyContainerPickedUp](t, ev)
	if picked.FacilityID != f.pickupRamp || picked.ContainerNumber != pmContainer {
		t.Fatal("EmptyContainerPickedUp payload mismatch")
	}
}

func TestPremountedLoad_DropClosesAllCustody(t *testing.T) {
	f := newPremountedLoad(t)
	f.toDrop(t)
	track := trackFrom(f.d)
	id, err := f.d.DropPremountedContainer()
	mustNoErr(t, err)

	requireNoEquipment(t, f.d)
	requireNoLiveAssignee(t, f.d)
	requireNoAssignee(t, f.d)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisReturned", "LoadedContainerDropped", "DispatchCompleted")
	ret := oneEvent[ChassisReturned](t, ev)
	if ret.FacilityID != f.dropoffRamp || ret.ChassisNumber != chassis {
		t.Fatal("ChassisReturned payload mismatch")
	}
	ld := oneEvent[LoadedContainerDropped](t, ev)
	if ld.FacilityID != f.dropoffRamp || ld.ContainerNumber != pmContainer {
		t.Fatal("LoadedContainerDropped payload mismatch")
	}
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
}

// --- premounted live unload ----------------------------------------------- //

func TestPremountedUnload_PickUp(t *testing.T) {
	f := newPremountedUnload(t)
	f.toPickUp(t)
	track := trackFrom(f.d)
	id, err := f.d.PickUpPremountedContainer(chassis)
	mustNoErr(t, err)

	requireEquipment(t, f.d, pmFull)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisAcquired", "LoadedContainerPickedUp")
	if oneEvent[ChassisAcquired](t, ev).ChassisNumber != chassis {
		t.Fatal("ChassisAcquired payload mismatch")
	}
	picked := oneEvent[LoadedContainerPickedUp](t, ev)
	if picked.FacilityID != f.pickupRamp || picked.ContainerNumber != pmContainer {
		t.Fatal("LoadedContainerPickedUp payload mismatch")
	}
}

func TestPremountedUnload_DropClosesAllCustody(t *testing.T) {
	f := newPremountedUnload(t)
	f.toDrop(t)
	track := trackFrom(f.d)
	id, err := f.d.DropPremountedContainer()
	mustNoErr(t, err)

	requireNoEquipment(t, f.d)
	assertTaskCompletedBy(t, f.d, id, assignee())
	ev := track.since()
	assertTypeNames(t, ev, "ChassisReturned", "EmptyContainerDropped", "DispatchCompleted")
	ret := oneEvent[ChassisReturned](t, ev)
	if ret.FacilityID != f.dropoffRamp || ret.ChassisNumber != chassis {
		t.Fatal("ChassisReturned payload mismatch")
	}
	ld := oneEvent[EmptyContainerDropped](t, ev)
	if ld.FacilityID != f.dropoffRamp || ld.ContainerNumber != pmContainer {
		t.Fatal("EmptyContainerDropped payload mismatch")
	}
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
}

// Thin end-to-end smoke tests (guard against state leaking between steps).

func TestPremountedLoad_FullLifecycle(t *testing.T) {
	f := newPremountedLoad(t)
	f.toDrop(t)
	_, err := f.d.DropPremountedContainer()
	mustNoErr(t, err)
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
	names := eventTypeNames(f.d.Events())
	if names[len(names)-1] != "DispatchCompleted" {
		t.Fatalf("last event = %q, want DispatchCompleted", names[len(names)-1])
	}
}

func TestPremountedUnload_FullLifecycle(t *testing.T) {
	f := newPremountedUnload(t)
	f.toDrop(t)
	_, err := f.d.DropPremountedContainer()
	mustNoErr(t, err)
	if f.d.Status() != StatusCompleted {
		t.Fatalf("status = %v, want completed", f.d.Status())
	}
}
