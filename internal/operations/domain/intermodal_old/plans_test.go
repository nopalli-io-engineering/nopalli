package intermodal_old

import (
	"slices"
	"testing"
)

func mustContainer(t *testing.T, raw string) ContainerNumber {
	t.Helper()
	c, err := NewContainerNumber(raw)
	if err != nil {
		t.Fatalf("container number %q: %v", raw, err)
	}
	return c
}

func taskActions(specs []TaskSpec) []Action {
	out := make([]Action, len(specs))
	for i, s := range specs {
		out[i] = s.Action
	}
	return out
}

func TestLiveLoad_TaskOrder(t *testing.T) {
	plan := LiveLoad{
		ContainerNumber:      mustContainer(t, "MSKU1"),
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	got := taskActions(plan.ToTasksSpec())
	want := []Action{
		ActionAcquireChassis,
		ActionPickUpEmptyContainer,
		ActionDeliverEmptyContainer,
		ActionHoldForLoading,
		ActionTakeLoadedContainer,
		ActionDropLoadedContainer,
		ActionReturnChassis,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("live-load tasks\n got: %v\nwant: %v", got, want)
	}
}

func TestLiveLoad_PremountedTaskOrder(t *testing.T) {
	plan := LiveLoad{
		ContainerNumber:      mustContainer(t, "MSKU1"),
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
		Premounted:           true,
	}
	got := taskActions(plan.ToTasksSpec())
	want := []Action{
		ActionPickUpEmptyPremountedContainer,
		ActionDeliverEmptyContainer,
		ActionHoldForLoading,
		ActionTakeLoadedContainer,
		ActionDropLoadedPremountedContainer,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("premounted live-load tasks\n got: %v\nwant: %v", got, want)
	}
}

func TestLiveUnload_TaskOrder(t *testing.T) {
	plan := LiveUnload{
		ContainerNumber:      mustContainer(t, "MSKU2"),
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	got := taskActions(plan.ToTasksSpec())
	want := []Action{
		ActionAcquireChassis,
		ActionPickUpLoadedContainer,
		ActionDeliverLoadedContainer,
		ActionHoldForEmptying,
		ActionTakeEmptyContainer,
		ActionDropEmptyContainer,
		ActionReturnChassis,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("live-unload tasks\n got: %v\nwant: %v", got, want)
	}
}

func TestLiveUnload_PremountedTaskOrder(t *testing.T) {
	plan := LiveUnload{
		ContainerNumber:      mustContainer(t, "MSKU2"),
		ContainerPickupRamp:  NewFacilityID(),
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
		Premounted:           true,
	}
	got := taskActions(plan.ToTasksSpec())
	want := []Action{
		ActionPickUpLoadedPremountedContainer,
		ActionDeliverLoadedContainer,
		ActionHoldForEmptying,
		ActionTakeEmptyContainer,
		ActionDropEmptyPremountedContainer,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("premounted live-unload tasks\n got: %v\nwant: %v", got, want)
	}
}

// The pickup task must carry the container marking and its ramp; chassis
// acquire/return tasks carry no facility (rdt: those specs omit facility_id).
func TestLiveLoad_PickupCarriesContainerAndFacility(t *testing.T) {
	pickupRamp := NewFacilityID()
	container := mustContainer(t, "MSKU9")
	plan := LiveLoad{
		ContainerNumber:      container,
		ContainerPickupRamp:  pickupRamp,
		ContainerDropoffRamp: NewFacilityID(),
		DockDoor:             NewFacilityID(),
	}
	specs := plan.ToTasksSpec()

	acquire := specs[0]
	if acquire.Action != ActionAcquireChassis {
		t.Fatalf("expected first task to be acquire chassis, got %v", acquire.Action)
	}
	if acquire.Facility != nil {
		t.Fatal("acquire chassis must have no facility")
	}

	pickup := specs[1]
	if pickup.Facility == nil || *pickup.Facility != pickupRamp {
		t.Fatalf("pickup facility = %v, want %v", pickup.Facility, pickupRamp)
	}
	if pickup.Details == nil || pickup.Details.Container == nil || *pickup.Details.Container != container {
		t.Fatalf("pickup must carry container %v, got %+v", container, pickup.Details)
	}
}
