package intermodal

import (
	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// This file ports the Dispatch aggregate (rdt: root.py): the aggregate root,
// its child entities (Task, Assignment, the two custody-episode types),
// and the factory that queues a dispatch from a move plan.
//
// Optionality note: method parameters use pointers for optional ids (the
// convention), while the comparable value objects Assignee/Equipment use
// zero-value sentinels (see values.go). Child-entity id/attribution fields that
// the tests compare by value are stored as values with a zero sentinel.

// --- action groups (rdt: the *_ACTION sets in root.py) -------------------- //

var (
	pickupContainerActions  = set(ActionPickUpEmptyContainer, ActionPickUpLoadedContainer)
	pickupPremountedActions = set(ActionPickUpEmptyPremountedContainer, ActionPickUpLoadedPremountedContainer)
	deliverContainerActions = set(ActionDeliverEmptyContainer, ActionDeliverLoadedContainer)
	takeContainerActions    = set(ActionTakeEmptyContainer, ActionTakeLoadedContainer)
	dropContainerActions    = set(ActionDropEmptyContainer, ActionDropLoadedContainer)
	dropPremountedActions   = set(ActionDropEmptyPremountedContainer, ActionDropLoadedPremountedContainer)
	acquireChassisActions   = set(ActionAcquireChassis)
	returnChassisActions    = set(ActionReturnChassis)
	holdActions             = set(ActionHoldForEmptying, ActionHoldForLoading)
)

func set(actions ...Action) map[Action]bool {
	m := make(map[Action]bool, len(actions))
	for _, a := range actions {
		m[a] = true
	}
	return m
}

// --- child entities ------------------------------------------------------- //

// ChassisCustodyEpisode tracks one stretch of chassis custody (rdt:
// ChassisCustodyEpisode). A chassis starts un-stationed; when opened and
// un-stationed it is the live chassis.
type ChassisCustodyEpisode struct {
	id            ChassisCustodyEpisodeID
	chassisNumber ChassisNumber // zero when unknown
	status        CustodyEpisodeStatus
	sourcedFrom   FacilityID
	stationedAt   *FacilityID
}

// ID returns the episode id.
func (e *ChassisCustodyEpisode) ID() ChassisCustodyEpisodeID { return e.id }

// ChassisNumber returns the chassis marking (zero if unknown).
func (e *ChassisCustodyEpisode) ChassisNumber() ChassisNumber { return e.chassisNumber }

// Status returns the episode status.
func (e *ChassisCustodyEpisode) Status() CustodyEpisodeStatus { return e.status }

// IsClosed reports whether custody has ended.
func (e *ChassisCustodyEpisode) IsClosed() bool { return e.status == CustodyClosed }

func (e *ChassisCustodyEpisode) open()  { e.stationedAt = nil; e.status = CustodyOpened }
func (e *ChassisCustodyEpisode) close() { e.status = CustodyClosed }

// ContainerCustodyEpisode tracks one stretch of container custody (rdt:
// ContainerCustodyEpisode). A container starts stationed at its source; when
// opened (and thereby un-stationed) it is the live container.
type ContainerCustodyEpisode struct {
	id              ContainerCustodyEpisodeID
	containerNumber ContainerNumber
	status          CustodyEpisodeStatus
	sourcedFrom     FacilityID
	stationedAt     *FacilityID
}

// ID returns the episode id.
func (e *ContainerCustodyEpisode) ID() ContainerCustodyEpisodeID { return e.id }

// ContainerNumber returns the container marking.
func (e *ContainerCustodyEpisode) ContainerNumber() ContainerNumber { return e.containerNumber }

// Status returns the episode status.
func (e *ContainerCustodyEpisode) Status() CustodyEpisodeStatus { return e.status }

// IsClosed reports whether custody has ended.
func (e *ContainerCustodyEpisode) IsClosed() bool { return e.status == CustodyClosed }

func (e *ContainerCustodyEpisode) open()  { e.stationedAt = nil; e.status = CustodyOpened }
func (e *ContainerCustodyEpisode) close() { e.status = CustodyClosed }

// Assignment is a driver+truck pairing on a dispatch (rdt: Assignment). Driver
// and truck use zero sentinels for "unset" while the pair is built up.
type Assignment struct {
	id       AssignmentID
	driverID DriverID
	truckID  TruckID
	status   AssignmentStatus
}

func (a *Assignment) ID() AssignmentID { return a.id }

func (a *Assignment) Status() AssignmentStatus { return a.status }

func (a *Assignment) IsActive() bool { return a.status == AssignmentActive }

func (a *Assignment) activate() { a.status = AssignmentActive }

func (a *Assignment) conclude() { a.status = AssignmentConcluded }

func (a *Assignment) assignee() Assignee { return Assignee{Driver: a.driverID, Truck: a.truckID} }

// taskDetails is the per-task data an task carries
type taskDetails struct {
	container ContainerNumber // zero when the action carries no container
}

type Task struct {
	id               TaskID
	no               int
	action           Action
	facilityID       *FacilityID
	status           TaskStatus
	details          taskDetails
	attributedDriver DriverID // zero until executed
	attributedTruck  TruckID  // zero until executed
}

func (t *Task) ID() TaskID { return t.id }

func (t *Task) No() int { return t.no }

func (t *Task) Action() Action { return t.action }

func (t *Task) FacilityID() *FacilityID { return t.facilityID }

func (t *Task) Status() TaskStatus { return t.status }

func (t *Task) AttributedDriver() DriverID { return t.attributedDriver }

func (t *Task) AttributedTruck() TruckID { return t.attributedTruck }

func (t *Task) IsCompleted() bool { return t.status == TaskCompleted }

func (t *Task) execute(byDriver DriverID, withTruck TruckID) {
	t.attributedDriver = byDriver
	t.attributedTruck = withTruck
	t.status = TaskCompleted
}

// --- aggregate root ------------------------------------------------------- //

// Dispatch is the intermodal drayage move aggregate (rdt: Dispatch). It is a
// sequential state machine
type Dispatch struct {
	domain.Root

	id                  DispatchID
	sourceQuoteID       SourceQuoteID // zero when there is no source quote
	dispatcherID        *DispatcherID
	planType            PlanType
	tasks               []*Task
	chassisEpisodes     []*ChassisCustodyEpisode
	containerEpisodes   []*ContainerCustodyEpisode
	assignments         []*Assignment
	currentAssignmentID *AssignmentID
	status              Status
	// cursor counts executed tasks (0 = none executed yet); the next
	// task to run is tasks[cursor]. Replaces rdt's nullable _item_cursor.
	cursor int
	// version is optimistic-concurrency bookkeeping for the data mapper (rdt:
	// the version_id_col on the imperative mapping). It is not domain state:
	// Queue starts it at 0, Reconstitute carries the persisted value, and the
	// repository bumps it on each successful Save.
	version int
}

// Version returns the optimistic-lock version last loaded or queued.
func (d *Dispatch) Version() int { return d.version }

// MarkPersisted records the version the repository wrote, after a successful
// Add/Save, so the in-memory aggregate matches the row for any further saves.
func (d *Dispatch) MarkPersisted(version int) { d.version = version }

// ID returns the dispatch id.
func (d *Dispatch) ID() DispatchID { return d.id }

// Status returns the lifecycle status.
func (d *Dispatch) Status() Status { return d.status }

// PlanType returns the plan the dispatch was built from.
func (d *Dispatch) PlanType() PlanType { return d.planType }

// SourceQuoteID returns the originating quote id and whether one is set.
func (d *Dispatch) SourceQuoteID() (SourceQuoteID, bool) {
	return d.sourceQuoteID, !d.sourceQuoteID.IsZero()
}

// Dispatcher returns the dispatcher the dispatch is delegated to, or nil.
func (d *Dispatch) Dispatcher() *DispatcherID { return d.dispatcherID }

// LiveEquipment returns the chassis/container currently in hand — the open,
// un-stationed episodes — and whether any is held (rdt: live_equipment).
func (d *Dispatch) LiveEquipment() (Equipment, bool) {
	var eq Equipment
	for _, e := range d.chassisEpisodes {
		if e.status == CustodyOpened && e.stationedAt == nil {
			eq.Chassis = e.chassisNumber
		}
	}
	for _, e := range d.containerEpisodes {
		if e.status == CustodyOpened && e.stationedAt == nil {
			eq.Container = e.containerNumber
		}
	}
	if eq.Container.IsZero() && eq.Chassis.IsZero() {
		return Equipment{}, false
	}
	return eq, true
}

// CurrentAssignee returns the assignee that would carry out the dispatch, if
// there is a usable one (rdt: current_assignee). A concluded assignment, or one
// with neither driver nor truck, yields no assignee.
func (d *Dispatch) CurrentAssignee() (Assignee, bool) {
	a := d.currentAssignment()
	if a == nil || a.status == AssignmentConcluded {
		return Assignee{}, false
	}
	assignee := a.assignee()
	if assignee.IsEmpty() {
		return Assignee{}, false
	}
	return assignee, true
}

// LiveAssignee returns the assignee only while the dispatch is actively running
// (the current assignment is active) (rdt: live_assignee). Unlike rdt this
// guards a nil current assignment rather than dereferencing it (see CONVENTIONS
// "fix known quirks").
func (d *Dispatch) LiveAssignee() (Assignee, bool) {
	a := d.currentAssignment()
	if a == nil || a.status != AssignmentActive {
		return Assignee{}, false
	}
	return a.assignee(), true
}

// Assignees returns the set of assignees across all assignments (rdt:
// assignees). Order is not significant.
func (d *Dispatch) Assignees() []Assignee {
	seen := make(map[Assignee]struct{}, len(d.assignments))
	out := make([]Assignee, 0, len(d.assignments))
	for _, a := range d.assignments {
		as := a.assignee()
		if _, ok := seen[as]; ok {
			continue
		}
		seen[as] = struct{}{}
		out = append(out, as)
	}
	return out
}

// ContainerEpisode finds the custody episode for a container marking, or nil.
func (d *Dispatch) ContainerEpisode(n ContainerNumber) *ContainerCustodyEpisode {
	for _, e := range d.containerEpisodes {
		if e.containerNumber == n {
			return e
		}
	}
	return nil
}

// ChassisEpisode finds the custody episode for a chassis marking, or nil.
func (d *Dispatch) ChassisEpisode(n ChassisNumber) *ChassisCustodyEpisode {
	for _, e := range d.chassisEpisodes {
		if e.chassisNumber == n {
			return e
		}
	}
	return nil
}

// Assignment finds an assignment by id, or nil.
func (d *Dispatch) Assignment(id AssignmentID) *Assignment {
	for _, a := range d.assignments {
		if a.id == id {
			return a
		}
	}
	return nil
}

// AssignmentsWithStatus returns the assignments in the given status.
func (d *Dispatch) AssignmentsWithStatus(s AssignmentStatus) []*Assignment {
	var out []*Assignment
	for _, a := range d.assignments {
		if a.status == s {
			out = append(out, a)
		}
	}
	return out
}

func (d *Dispatch) Task(id TaskID) *Task {
	for _, t := range d.tasks {
		if t.id == id {
			return t
		}
	}
	return nil
}

func (d *Dispatch) currentAssignment() *Assignment {
	if d.currentAssignmentID == nil {
		return nil
	}
	return d.Assignment(*d.currentAssignmentID)
}

// --- lifecycle commands --------------------------------------------------- //

// Delegate assigns the dispatch to a dispatcher (rdt: delegate).
func (d *Dispatch) Delegate(dispatcherID DispatcherID) {
	d.dispatcherID = &dispatcherID
	d.RecordEvents(DispatchDelegated{
		DispatchID:     d.id,
		ToDispatcherID: dispatcherID,
	})
}

// Assign proposes or updates the driver+truck pair (rdt: assign). Either id may
// be nil while the pair is being built up. Returns the current assignment id.
func (d *Dispatch) Assign(driverID *DriverID, truckID *TruckID) (AssignmentID, error) {
	switch d.status {
	case StatusInProgress:
		return AssignmentID{}, ErrDispatchAssignedWhileInProgress
	case StatusCompleted:
		return AssignmentID{}, ErrDispatchAssignedAfterCompletion
	}

	current := d.currentAssignment()
	switch {
	case current == nil && driverID == nil && truckID == nil:
		return AssignmentID{}, ErrDispatchAssigneeEmpty
	case current == nil:
		a := &Assignment{
			id:       NewAssignmentID(),
			driverID: derefDriver(driverID),
			truckID:  derefTruck(truckID),
			status:   AssignmentPending,
		}
		d.assignments = append(d.assignments, a)
		id := a.id
		d.currentAssignmentID = &id
		current = a
	case current.driverID == derefDriver(driverID) && current.truckID == derefTruck(truckID):
		return AssignmentID{}, ErrDispatchAlreadyAssignedAssignee
	default:
		current.driverID = derefDriver(driverID)
		current.truckID = derefTruck(truckID)
	}

	if d.status == StatusReady {
		d.status = StatusQueued
		d.RecordEvents(DispatchRequeued{DispatchID: d.id})
	}
	if driverID != nil && truckID != nil {
		d.RecordEvents(AssignmentProposed{
			DispatchID:   d.id,
			AssignmentID: *d.currentAssignmentID,
			DriverID:     *driverID,
			TruckID:      *truckID,
		})
	}
	return current.id, nil
}

// Approve marks a delegated, fully-assigned dispatch ready to start (rdt:
// approve).
func (d *Dispatch) Approve() error {
	if d.dispatcherID == nil {
		return ErrDispatchNotDelegated
	}
	current := d.currentAssignment()
	if d.currentAssignmentID == nil || current == nil {
		return ErrDispatchNotAssigned
	}
	if current.driverID.IsZero() {
		return ErrDispatchDriverMissing
	}
	if current.truckID.IsZero() {
		return ErrDispatchTruckMissing
	}
	d.status = StatusReady
	assignee, _ := d.CurrentAssignee()
	d.RecordEvents(DispatchApproved{
		DispatchID: d.id,
		DriverID:   assignee.Driver,
		TruckID:    assignee.Truck,
	})
	return nil
}

// Start activates the assignment and begins task execution.
func (d *Dispatch) Start() error {
	if d.status != StatusReady {
		return ErrDispatchNotReady
	}
	d.currentAssignment().activate()
	d.status = StatusInProgress
	d.RecordEvents(DispatchStarted{DispatchID: d.id})
	return nil
}

// Verify marks a completed dispatch verified (rdt: verify).
func (d *Dispatch) Verify() error {
	if d.status != StatusCompleted {
		return ErrDispatchNotCompleted
	}
	d.status = StatusVerified
	d.RecordEvents(DispatchVerified{
		DispatchID:    d.id,
		SourceQuoteID: d.sourceQuoteID,
	})
	return nil
}

// --- Task execution commands -------------------------------------------- //

// Hold executes the current hold step (rdt: hold).
func (d *Dispatch) Hold() (TaskID, error) {
	task, err := d.executeTask(holdActions)
	if err != nil {
		return TaskID{}, err
	}
	if task.action == ActionHoldForLoading {
		d.RecordEvents(DispatchHeldForLoading{
			DispatchID: d.id, FacilityID: deref(task.facilityID),
		})
	} else {
		d.RecordEvents(DispatchHeldForEmptying{
			DispatchID: d.id, FacilityID: deref(task.facilityID),
		})
	}
	return task.id, nil
}

// AcquireChassis executes the acquire-chassis step, opening a chassis custody
// episode (rdt: acquire_chassis).
func (d *Dispatch) AcquireChassis(facilityID FacilityID, chassisNumber ChassisNumber) (TaskID, error) {
	task, err := d.executeTask(acquireChassisActions)
	if err != nil {
		return TaskID{}, err
	}
	d.chassisEpisodes = append(d.chassisEpisodes, &ChassisCustodyEpisode{
		id:            NewChassisCustodyEpisodeID(),
		sourcedFrom:   facilityID,
		chassisNumber: chassisNumber,
		status:        CustodyOpened,
	})
	d.RecordEvents(ChassisAcquired{
		DispatchID: d.id, FacilityID: facilityID, ChassisNumber: chassisNumber,
	})
	return task.id, nil
}

// ReturnChassis executes the return-chassis step, closing custody and the
// dispatch (rdt: return_chassis).
func (d *Dispatch) ReturnChassis(facilityID FacilityID) (TaskID, error) {
	task, err := d.executeTask(returnChassisActions)
	if err != nil {
		return TaskID{}, err
	}
	eq, _ := d.LiveEquipment()
	chassisNumber := eq.Chassis
	if episode := d.ChassisEpisode(chassisNumber); episode != nil {
		episode.close()
	}
	d.currentAssignment().conclude()
	d.RecordEvents(
		ChassisReturned{DispatchID: d.id, FacilityID: facilityID, ChassisNumber: chassisNumber},
		DispatchCompleted{DispatchID: d.id, SourceQuoteID: d.sourceQuoteID},
	)
	return task.id, nil
}

// PickUpContainer executes the pick-up-container step, opening container
// custody (rdt: pick_up_container).
func (d *Dispatch) PickUpContainer() (TaskID, error) {
	task, err := d.executeTask(pickupContainerActions)
	if err != nil {
		return TaskID{}, err
	}
	if episode := d.ContainerEpisode(task.details.container); episode != nil {
		episode.open()
	}
	eq, _ := d.LiveEquipment()
	if task.action == ActionPickUpLoadedContainer {
		d.RecordEvents(LoadedContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	}
	return task.id, nil
}

// PickUpPremountedContainer executes the pick-up-premounted step, opening
// container custody and a chassis episode together (rdt:
// pick_up_premounted_container).
func (d *Dispatch) PickUpPremountedContainer(chassisNumber ChassisNumber) (TaskID, error) {
	task, err := d.executeTask(pickupPremountedActions)
	if err != nil {
		return TaskID{}, err
	}
	containerNumber := task.details.container
	if episode := d.ContainerEpisode(containerNumber); episode != nil {
		episode.open()
	}
	d.chassisEpisodes = append(d.chassisEpisodes, &ChassisCustodyEpisode{
		id:            NewChassisCustodyEpisodeID(),
		sourcedFrom:   deref(task.facilityID),
		chassisNumber: chassisNumber,
		status:        CustodyOpened,
	})
	var pickup domain.DomainEvent
	if task.action == ActionPickUpLoadedPremountedContainer {
		pickup = LoadedContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		}
	} else {
		pickup = EmptyContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		}
	}
	d.RecordEvents(
		ChassisAcquired{DispatchID: d.id, FacilityID: deref(task.facilityID), ChassisNumber: chassisNumber},
		pickup,
	)
	return task.id, nil
}

// DeliverContainer executes the deliver-container step (rdt: deliver_container).
func (d *Dispatch) DeliverContainer() (TaskID, error) {
	task, err := d.executeTask(deliverContainerActions)
	if err != nil {
		return TaskID{}, err
	}
	eq, _ := d.LiveEquipment()
	if task.action == ActionDeliverLoadedContainer {
		d.RecordEvents(LoadedContainerDelivered{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerDelivered{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	}
	return task.id, nil
}

// TakeContainer executes the take-container step, releasing the hold (rdt:
// take_container).
func (d *Dispatch) TakeContainer() (TaskID, error) {
	task, err := d.executeTask(takeContainerActions)
	if err != nil {
		return TaskID{}, err
	}
	eq, _ := d.LiveEquipment()
	if task.action == ActionTakeLoadedContainer {
		d.RecordEvents(LoadedContainerTaken{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerTaken{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: eq.Container,
		})
	}
	d.RecordEvents(DispatchReleased{DispatchID: d.id, FacilityID: deref(task.facilityID)})
	return task.id, nil
}

// DropContainer executes the drop-container step, closing container custody
// (rdt: drop_container).
func (d *Dispatch) DropContainer() (TaskID, error) {
	task, err := d.executeTask(dropContainerActions)
	if err != nil {
		return TaskID{}, err
	}
	eq, _ := d.LiveEquipment()
	containerNumber := eq.Container
	if task.action == ActionDropLoadedContainer {
		d.RecordEvents(LoadedContainerDropped{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		})
	} else {
		d.RecordEvents(EmptyContainerDropped{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		})
	}
	if episode := d.ContainerEpisode(containerNumber); episode != nil {
		episode.close()
	}
	return task.id, nil
}

// DropPremountedContainer executes the drop-premounted step, closing both
// custody episodes and concluding the assignment (rdt:
// drop_premounted_container).
func (d *Dispatch) DropPremountedContainer() (TaskID, error) {
	task, err := d.executeTask(dropPremountedActions)
	if err != nil {
		return TaskID{}, err
	}
	d.currentAssignment().conclude()
	eq, _ := d.LiveEquipment()
	containerNumber, chassisNumber := eq.Container, eq.Chassis
	if episode := d.ContainerEpisode(containerNumber); episode != nil {
		episode.close()
	}
	if episode := d.ChassisEpisode(chassisNumber); episode != nil {
		episode.close()
	}
	var deliver domain.DomainEvent
	if task.action == ActionDropEmptyPremountedContainer {
		deliver = EmptyContainerDropped{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		}
	} else {
		deliver = LoadedContainerDropped{
			DispatchID: d.id, FacilityID: deref(task.facilityID), ContainerNumber: containerNumber,
		}
	}
	d.RecordEvents(
		ChassisReturned{DispatchID: d.id, FacilityID: deref(task.facilityID), ChassisNumber: chassisNumber},
		deliver,
	)
	if d.tasksCompleted() {
		d.RecordEvents(DispatchCompleted{DispatchID: d.id, SourceQuoteID: d.sourceQuoteID})
	}
	return task.id, nil
}

// --- Task cursor -------------------------------------------------------- //

func (d *Dispatch) executeTask(allowed map[Action]bool) (*Task, error) {
	if err := d.checkNextAction(allowed); err != nil {
		return nil, err
	}
	d.cursor++
	if d.tasksCompleted() {
		d.status = StatusCompleted
	}
	task := d.tasks[d.cursor-1]
	assignee, _ := d.LiveAssignee()
	task.execute(assignee.Driver, assignee.Truck)
	return task, nil
}

func (d *Dispatch) nextTask() *Task {
	if d.tasksCompleted() {
		return nil
	}
	return d.tasks[d.cursor]
}

func (d *Dispatch) tasksCompleted() bool {
	return d.cursor == len(d.tasks)
}

func (d *Dispatch) checkNextAction(allowed map[Action]bool) error {
	if d.status != StatusInProgress {
		return ErrDispatchNotInProgress
	}
	next := d.nextTask()
	if next == nil || !allowed[next.action] {
		var expected Action
		if next != nil {
			expected = next.action
		}
		return newOutOfSequence(expected)
	}
	return nil
}

// --- factory -------------------------------------------------------------- //

// Queue builds and queues a dispatch from a move plan (rdt: DispatchFactory.
// queue). sourceQuoteID is optional.
func Queue(sourceQuoteID *SourceQuoteID, move MovePlan) *Dispatch {
	return QueueFromSpecs(sourceQuoteID, move.ToTasksSpec(), move.PlanType())
}

// QueueFromSpecs builds and queues a dispatch directly from an task spec
func QueueFromSpecs(sourceQuoteID *SourceQuoteID, specs []TaskSpec, planType PlanType) *Dispatch {
	tasks := make([]*Task, 0, len(specs))
	var containerEpisodes []*ContainerCustodyEpisode
	for i, s := range specs {
		var details taskDetails
		if s.Details != nil && s.Details.Container != nil {
			details.container = *s.Details.Container
		}
		tasks = append(tasks, &Task{
			id:         NewTaskID(),
			no:         i + 1,
			facilityID: s.Facility,
			action:     s.Action,
			status:     TaskPending,
			details:    details,
		})
		if s.Details != nil && s.Details.Container != nil {
			sourced := deref(s.Facility)
			stationed := sourced
			containerEpisodes = append(containerEpisodes, &ContainerCustodyEpisode{
				id:              NewContainerCustodyEpisodeID(),
				sourcedFrom:     sourced,
				containerNumber: *s.Details.Container,
				status:          CustodyPending,
				stationedAt:     &stationed,
			})
		}
	}

	d := &Dispatch{
		id:                NewDispatchID(),
		sourceQuoteID:     derefSourceQuote(sourceQuoteID),
		planType:          planType,
		tasks:             tasks,
		containerEpisodes: containerEpisodes,
		status:            StatusQueued,
	}
	d.RecordEvents(DispatchQueued{
		DispatchID:    d.id,
		SourceQuoteID: d.sourceQuoteID,
	})
	return d
}

// --- small helpers -------------------------------------------------------- //

func deref(p *FacilityID) FacilityID {
	if p == nil {
		return FacilityID{}
	}
	return *p
}

func derefDriver(p *DriverID) DriverID {
	if p == nil {
		return DriverID{}
	}
	return *p
}

func derefTruck(p *TruckID) TruckID {
	if p == nil {
		return TruckID{}
	}
	return *p
}

func derefSourceQuote(p *SourceQuoteID) SourceQuoteID {
	if p == nil {
		return SourceQuoteID{}
	}
	return *p
}
