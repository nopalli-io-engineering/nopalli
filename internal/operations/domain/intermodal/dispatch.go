package intermodal

import (
	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// This file ports the Dispatch aggregate (rdt: root.py): the aggregate root,
// its child entities (AgendaItem, Assignment, the two custody-episode types),
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

// ID returns the assignment id.
func (a *Assignment) ID() AssignmentID { return a.id }

// Status returns the assignment status.
func (a *Assignment) Status() AssignmentStatus { return a.status }

// IsActive reports whether this assignment is the active one.
func (a *Assignment) IsActive() bool { return a.status == AssignmentActive }

func (a *Assignment) activate() { a.status = AssignmentActive }
func (a *Assignment) conclude() { a.status = AssignmentConcluded }

// assignee returns this assignment as an Assignee value object.
func (a *Assignment) assignee() Assignee { return Assignee{Driver: a.driverID, Truck: a.truckID} }

// itemDetails is the per-item data an agenda item carries (rdt: AgendaItem's
// details dict, keyed "container_number").
type itemDetails struct {
	container ContainerNumber // zero when the action carries no container
}

// AgendaItem is one ordered step of a dispatch's agenda (rdt: AgendaItem).
type AgendaItem struct {
	id               AgendaItemID
	itemNo           int
	action           Action
	facilityID       *FacilityID
	status           AgendaItemStatus
	details          itemDetails
	attributedDriver DriverID // zero until executed
	attributedTruck  TruckID  // zero until executed
}

// ID returns the agenda item id.
func (i *AgendaItem) ID() AgendaItemID { return i.id }

// ItemNo returns the 1-based position of this item in the agenda.
func (i *AgendaItem) ItemNo() int { return i.itemNo }

// Action returns the action this item calls for.
func (i *AgendaItem) Action() Action { return i.action }

// FacilityID returns the facility the action happens at, if any.
func (i *AgendaItem) FacilityID() *FacilityID { return i.facilityID }

// Status returns the item status.
func (i *AgendaItem) Status() AgendaItemStatus { return i.status }

// AttributedDriver returns the driver who executed the item (zero if pending).
func (i *AgendaItem) AttributedDriver() DriverID { return i.attributedDriver }

// AttributedTruck returns the truck used to execute the item (zero if pending).
func (i *AgendaItem) AttributedTruck() TruckID { return i.attributedTruck }

// IsCompleted reports whether the item has been executed.
func (i *AgendaItem) IsCompleted() bool { return i.status == AgendaItemCompleted }

func (i *AgendaItem) execute(byDriver DriverID, withTruck TruckID) {
	i.attributedDriver = byDriver
	i.attributedTruck = withTruck
	i.status = AgendaItemCompleted
}

// --- aggregate root ------------------------------------------------------- //

// Dispatch is the intermodal drayage move aggregate (rdt: Dispatch). It is a
// sequential state machine over an agenda; see root.py for the lifecycle.
type Dispatch struct {
	domain.Root

	id                  DispatchID
	sourceQuoteID       SourceQuoteID // zero when there is no source quote
	dispatcherID        *DispatcherID
	planType            PlanType
	agendaItems         []*AgendaItem
	chassisEpisodes     []*ChassisCustodyEpisode
	containerEpisodes   []*ContainerCustodyEpisode
	assignments         []*Assignment
	currentAssignmentID *AssignmentID
	status              Status
	// cursor counts executed agenda items (0 = none executed yet); the next
	// item to run is agendaItems[cursor]. Replaces rdt's nullable _item_cursor.
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

// AgendaItem finds an agenda item by id, or nil.
func (d *Dispatch) AgendaItem(id AgendaItemID) *AgendaItem {
	for _, i := range d.agendaItems {
		if i.id == id {
			return i
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

// Start activates the assignment and begins agenda execution (rdt: start).
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

// --- agenda execution commands -------------------------------------------- //

// Hold executes the current hold step (rdt: hold).
func (d *Dispatch) Hold() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(holdActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	if item.action == ActionHoldForLoading {
		d.RecordEvents(DispatchHeldForLoading{
			DispatchID: d.id, FacilityID: deref(item.facilityID),
		})
	} else {
		d.RecordEvents(DispatchHeldForEmptying{
			DispatchID: d.id, FacilityID: deref(item.facilityID),
		})
	}
	return item.id, nil
}

// AcquireChassis executes the acquire-chassis step, opening a chassis custody
// episode (rdt: acquire_chassis).
func (d *Dispatch) AcquireChassis(facilityID FacilityID, chassisNumber ChassisNumber) (AgendaItemID, error) {
	item, err := d.executeAgendaItem(acquireChassisActions)
	if err != nil {
		return AgendaItemID{}, err
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
	return item.id, nil
}

// ReturnChassis executes the return-chassis step, closing custody and the
// dispatch (rdt: return_chassis).
func (d *Dispatch) ReturnChassis(facilityID FacilityID) (AgendaItemID, error) {
	item, err := d.executeAgendaItem(returnChassisActions)
	if err != nil {
		return AgendaItemID{}, err
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
	return item.id, nil
}

// PickUpContainer executes the pick-up-container step, opening container
// custody (rdt: pick_up_container).
func (d *Dispatch) PickUpContainer() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(pickupContainerActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	if episode := d.ContainerEpisode(item.details.container); episode != nil {
		episode.open()
	}
	eq, _ := d.LiveEquipment()
	if item.action == ActionPickUpLoadedContainer {
		d.RecordEvents(LoadedContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	}
	return item.id, nil
}

// PickUpPremountedContainer executes the pick-up-premounted step, opening
// container custody and a chassis episode together (rdt:
// pick_up_premounted_container).
func (d *Dispatch) PickUpPremountedContainer(chassisNumber ChassisNumber) (AgendaItemID, error) {
	item, err := d.executeAgendaItem(pickupPremountedActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	containerNumber := item.details.container
	if episode := d.ContainerEpisode(containerNumber); episode != nil {
		episode.open()
	}
	d.chassisEpisodes = append(d.chassisEpisodes, &ChassisCustodyEpisode{
		id:            NewChassisCustodyEpisodeID(),
		sourcedFrom:   deref(item.facilityID),
		chassisNumber: chassisNumber,
		status:        CustodyOpened,
	})
	var pickup domain.DomainEvent
	if item.action == ActionPickUpLoadedPremountedContainer {
		pickup = LoadedContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		}
	} else {
		pickup = EmptyContainerPickedUp{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		}
	}
	d.RecordEvents(
		ChassisAcquired{DispatchID: d.id, FacilityID: deref(item.facilityID), ChassisNumber: chassisNumber},
		pickup,
	)
	return item.id, nil
}

// DeliverContainer executes the deliver-container step (rdt: deliver_container).
func (d *Dispatch) DeliverContainer() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(deliverContainerActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	eq, _ := d.LiveEquipment()
	if item.action == ActionDeliverLoadedContainer {
		d.RecordEvents(LoadedContainerDelivered{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerDelivered{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	}
	return item.id, nil
}

// TakeContainer executes the take-container step, releasing the hold (rdt:
// take_container).
func (d *Dispatch) TakeContainer() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(takeContainerActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	eq, _ := d.LiveEquipment()
	if item.action == ActionTakeLoadedContainer {
		d.RecordEvents(LoadedContainerTaken{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	} else {
		d.RecordEvents(EmptyContainerTaken{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: eq.Container,
		})
	}
	d.RecordEvents(DispatchReleased{DispatchID: d.id, FacilityID: deref(item.facilityID)})
	return item.id, nil
}

// DropContainer executes the drop-container step, closing container custody
// (rdt: drop_container).
func (d *Dispatch) DropContainer() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(dropContainerActions)
	if err != nil {
		return AgendaItemID{}, err
	}
	eq, _ := d.LiveEquipment()
	containerNumber := eq.Container
	if item.action == ActionDropLoadedContainer {
		d.RecordEvents(LoadedContainerDropped{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		})
	} else {
		d.RecordEvents(EmptyContainerDropped{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		})
	}
	if episode := d.ContainerEpisode(containerNumber); episode != nil {
		episode.close()
	}
	return item.id, nil
}

// DropPremountedContainer executes the drop-premounted step, closing both
// custody episodes and concluding the assignment (rdt:
// drop_premounted_container).
func (d *Dispatch) DropPremountedContainer() (AgendaItemID, error) {
	item, err := d.executeAgendaItem(dropPremountedActions)
	if err != nil {
		return AgendaItemID{}, err
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
	if item.action == ActionDropEmptyPremountedContainer {
		deliver = EmptyContainerDropped{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		}
	} else {
		deliver = LoadedContainerDropped{
			DispatchID: d.id, FacilityID: deref(item.facilityID), ContainerNumber: containerNumber,
		}
	}
	d.RecordEvents(
		ChassisReturned{DispatchID: d.id, FacilityID: deref(item.facilityID), ChassisNumber: chassisNumber},
		deliver,
	)
	if d.agendaCompleted() {
		d.RecordEvents(DispatchCompleted{DispatchID: d.id, SourceQuoteID: d.sourceQuoteID})
	}
	return item.id, nil
}

// --- agenda cursor -------------------------------------------------------- //

func (d *Dispatch) executeAgendaItem(allowed map[Action]bool) (*AgendaItem, error) {
	if err := d.checkNextAction(allowed); err != nil {
		return nil, err
	}
	d.cursor++
	if d.agendaCompleted() {
		d.status = StatusCompleted
	}
	item := d.agendaItems[d.cursor-1]
	assignee, _ := d.LiveAssignee()
	item.execute(assignee.Driver, assignee.Truck)
	return item, nil
}

func (d *Dispatch) nextAgendaItem() *AgendaItem {
	if d.agendaCompleted() {
		return nil
	}
	return d.agendaItems[d.cursor]
}

func (d *Dispatch) agendaCompleted() bool {
	return d.cursor == len(d.agendaItems)
}

func (d *Dispatch) checkNextAction(allowed map[Action]bool) error {
	if d.status != StatusInProgress {
		return ErrDispatchNotInProgress
	}
	next := d.nextAgendaItem()
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
	return QueueFromSpecs(sourceQuoteID, move.ToAgendaSpec(), move.PlanType())
}

// QueueFromSpecs builds and queues a dispatch directly from an agenda spec
// (rdt: DispatchFactory.queue_from_specs).
func QueueFromSpecs(sourceQuoteID *SourceQuoteID, specs []AgendaItemSpec, planType PlanType) *Dispatch {
	items := make([]*AgendaItem, 0, len(specs))
	var containerEpisodes []*ContainerCustodyEpisode
	for i, s := range specs {
		var details itemDetails
		if s.Details != nil && s.Details.Container != nil {
			details.container = *s.Details.Container
		}
		items = append(items, &AgendaItem{
			id:         NewAgendaItemID(),
			itemNo:     i + 1,
			facilityID: s.Facility,
			action:     s.Action,
			status:     AgendaItemPending,
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
		agendaItems:       items,
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
