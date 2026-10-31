package intermodal_old

// Move plans (rdt: value_objects.py MovePlan / LiveLoad / LiveUnload) describe a
// kind of drayage move and know how to expand themselves into an ordered task
// spec. They are pure value objects — no aggregate state — so the tasks for a
// given plan is deterministic and unit-testable on its own. DispatchFactory
// (ported later with the aggregate) turns a plan's spec into a queued Dispatch.

// MovePlan is a planned move that can produce the ordered tasks a Dispatch
// will execute (rdt: MovePlan protocol).
type MovePlan interface {
	PlanType() PlanType
	ToTasksSpec() []TaskSpec
}

// facilityAt returns a pointer to a copy of f, for the optional Facility field.
func facilityAt(f FacilityID) *FacilityID { return &f }

// containerDetails returns task details carrying a copy of c.
func containerDetails(c ContainerNumber) *TaskDetails {
	return &TaskDetails{Container: &c}
}

// LiveLoad: pick up an empty container, deliver it to a dock door to be loaded,
// wait, take the now-loaded container, and drop it at the destination ramp
// (rdt: LiveLoad). The premounted variant skips the separate chassis
// acquire/return because the container arrives already mounted on a chassis.
type LiveLoad struct {
	ContainerNumber      ContainerNumber
	ContainerPickupRamp  FacilityID
	ContainerDropoffRamp FacilityID
	DockDoor             FacilityID
	Premounted           bool
}

// PlanType identifies this as a live-load move.
func (p LiveLoad) PlanType() PlanType { return PlanLiveLoad }

// TasksSpec expands the plan into its ordered tasks: set-up, then the core
// deliver/hold/take at the dock, then wrap-up. Order and actions mirror rdt
// exactly so the two apps plan identical tasks.
func (p LiveLoad) ToTasksSpec() []TaskSpec {
	setUp := []TaskSpec{
		{Action: ActionAcquireChassis},
		{
			Action:   ActionPickUpEmptyContainer,
			Facility: facilityAt(p.ContainerPickupRamp),
			Details:  containerDetails(p.ContainerNumber),
		},
	}
	core := []TaskSpec{
		{Action: ActionDeliverEmptyContainer, Facility: facilityAt(p.DockDoor)},
		{Action: ActionHoldForLoading, Facility: facilityAt(p.DockDoor)},
		{Action: ActionTakeLoadedContainer, Facility: facilityAt(p.DockDoor)},
	}
	wrapUp := []TaskSpec{
		{Action: ActionDropLoadedContainer, Facility: facilityAt(p.ContainerDropoffRamp)},
		{Action: ActionReturnChassis},
	}
	if p.Premounted {
		setUp = []TaskSpec{
			{
				Action:   ActionPickUpEmptyPremountedContainer,
				Facility: facilityAt(p.ContainerPickupRamp),
				Details:  containerDetails(p.ContainerNumber),
			},
		}
		wrapUp = []TaskSpec{
			{Action: ActionDropLoadedPremountedContainer, Facility: facilityAt(p.ContainerDropoffRamp)},
		}
	}
	return concat(setUp, core, wrapUp)
}

// LiveUnload: pick up a loaded container, deliver it to a dock door to be
// emptied, wait, take the now-empty container, and drop it at the destination
// ramp (rdt: LiveUnload). The premounted variant skips the separate chassis
// acquire/return.
type LiveUnload struct {
	ContainerNumber      ContainerNumber
	ContainerPickupRamp  FacilityID
	ContainerDropoffRamp FacilityID
	DockDoor             FacilityID
	Premounted           bool
}

// PlanType identifies this as a live-unload move.
func (p LiveUnload) PlanType() PlanType { return PlanLiveUnload }

// ToTasksSpec expands the plan into its ordered tasks, mirroring rdt.
func (p LiveUnload) ToTasksSpec() []TaskSpec {
	setUp := []TaskSpec{
		{Action: ActionAcquireChassis},
		{
			Action:   ActionPickUpLoadedContainer,
			Facility: facilityAt(p.ContainerPickupRamp),
			Details:  containerDetails(p.ContainerNumber),
		},
	}
	core := []TaskSpec{
		{Action: ActionDeliverLoadedContainer, Facility: facilityAt(p.DockDoor)},
		{Action: ActionHoldForEmptying, Facility: facilityAt(p.DockDoor)},
		{Action: ActionTakeEmptyContainer, Facility: facilityAt(p.DockDoor)},
	}
	wrapUp := []TaskSpec{
		{Action: ActionDropEmptyContainer, Facility: facilityAt(p.ContainerDropoffRamp)},
		{Action: ActionReturnChassis},
	}
	if p.Premounted {
		setUp = []TaskSpec{
			{
				Action:   ActionPickUpLoadedPremountedContainer,
				Facility: facilityAt(p.ContainerPickupRamp),
				Details:  containerDetails(p.ContainerNumber),
			},
		}
		wrapUp = []TaskSpec{
			{Action: ActionDropEmptyPremountedContainer, Facility: facilityAt(p.ContainerDropoffRamp)},
		}
	}
	return concat(setUp, core, wrapUp)
}

// concat flattens task segments into a single ordered slice.
func concat(segments ...[]TaskSpec) []TaskSpec {
	var n int
	for _, s := range segments {
		n += len(s)
	}
	out := make([]TaskSpec, 0, n)
	for _, s := range segments {
		out = append(out, s...)
	}
	return out
}
