package intermodal_old

// This file is the aggregate's persistence contract — the Go analogue of rdt's
// imperative mapping. Rather than let infrastructure reach into the aggregate's
// unexported fields (or reflect into them), the domain exposes a flat snapshot
// and a reconstitution constructor:
//
//	snap := d.Snapshot()        // aggregate -> plain data (for writing rows)
//	d := Reconstitute(snap)     // plain data -> aggregate (from read rows)
//
// Infrastructure maps Snapshot <-> SQL rows and never sees aggregate internals.
// Reconstitute raises no domain events (rdt had to re-init the transient event
// buffer on load; here a reconstituted aggregate simply starts with an empty
// domain.Root).
//
// Optionality here uses pointers (nil = SQL NULL), the persistence-boundary
// convention; the aggregate's own zero-value sentinels are translated at the
// edges.

// DispatchSnapshot is the full persisted state of a Dispatch aggregate.
type DispatchSnapshot struct {
	ID                  DispatchID
	SourceQuoteID       *SourceQuoteID
	DispatcherID        *DispatcherID
	PlanType            PlanType
	Status              Status
	Cursor              int
	Version             int
	CurrentAssignmentID *AssignmentID
	Tasks               []TaskSnapshot
	Assignments         []AssignmentSnapshot
	ChassisEpisodes     []ChassisEpisodeSnapshot
	ContainerEpisodes   []ContainerEpisodeSnapshot
}

// TaskSnapshot is one task row.
type TaskSnapshot struct {
	ID               TaskID
	No               int
	Action           Action
	FacilityID       *FacilityID
	Status           TaskStatus
	Container        *ContainerNumber // the task's details.container, if any
	AttributedDriver *DriverID
	AttributedTruck  *TruckID
}

// AssignmentSnapshot is one assignment row.
type AssignmentSnapshot struct {
	ID       AssignmentID
	DriverID *DriverID
	TruckID  *TruckID
	Status   AssignmentStatus
}

// ChassisEpisodeSnapshot is one chassis custody row.
type ChassisEpisodeSnapshot struct {
	ID            ChassisCustodyEpisodeID
	Status        CustodyEpisodeStatus
	SourcedFrom   FacilityID
	StationedAt   *FacilityID
	ChassisNumber *ChassisNumber
}

// ContainerEpisodeSnapshot is one container custody row.
type ContainerEpisodeSnapshot struct {
	ID              ContainerCustodyEpisodeID
	Status          CustodyEpisodeStatus
	SourcedFrom     FacilityID
	StationedAt     *FacilityID
	ContainerNumber ContainerNumber
}

// Snapshot captures the aggregate's state as plain data for persistence.
func (d *Dispatch) Snapshot() DispatchSnapshot {
	s := DispatchSnapshot{
		ID:                  d.id,
		SourceQuoteID:       nonZeroSourceQuote(d.sourceQuoteID),
		DispatcherID:        d.dispatcherID,
		PlanType:            d.planType,
		Status:              d.status,
		Cursor:              d.cursor,
		Version:             d.version,
		CurrentAssignmentID: d.currentAssignmentID,
	}
	for _, t := range d.tasks {
		s.Tasks = append(s.Tasks, TaskSnapshot{
			ID:               t.id,
			No:               t.no,
			Action:           t.action,
			FacilityID:       t.facilityID,
			Status:           t.status,
			Container:        nonZeroContainer(t.details.container),
			AttributedDriver: nonZeroDriver(t.attributedDriver),
			AttributedTruck:  nonZeroTruck(t.attributedTruck),
		})
	}
	for _, a := range d.assignments {
		s.Assignments = append(s.Assignments, AssignmentSnapshot{
			ID:       a.id,
			DriverID: nonZeroDriver(a.driverID),
			TruckID:  nonZeroTruck(a.truckID),
			Status:   a.status,
		})
	}
	for _, e := range d.chassisEpisodes {
		s.ChassisEpisodes = append(s.ChassisEpisodes, ChassisEpisodeSnapshot{
			ID:            e.id,
			Status:        e.status,
			SourcedFrom:   e.sourcedFrom,
			StationedAt:   e.stationedAt,
			ChassisNumber: nonZeroChassis(e.chassisNumber),
		})
	}
	for _, e := range d.containerEpisodes {
		s.ContainerEpisodes = append(s.ContainerEpisodes, ContainerEpisodeSnapshot{
			ID:              e.id,
			Status:          e.status,
			SourcedFrom:     e.sourcedFrom,
			StationedAt:     e.stationedAt,
			ContainerNumber: e.containerNumber,
		})
	}
	return s
}

// Reconstitute rebuilds a Dispatch from persisted state, without raising any
// domain events. It trusts the snapshot (it came from our own store).
func Reconstitute(s DispatchSnapshot) *Dispatch {
	d := &Dispatch{
		id:                  s.ID,
		sourceQuoteID:       derefSourceQuote(s.SourceQuoteID),
		dispatcherID:        s.DispatcherID,
		planType:            s.PlanType,
		status:              s.Status,
		cursor:              s.Cursor,
		version:             s.Version,
		currentAssignmentID: s.CurrentAssignmentID,
	}
	for _, t := range s.Tasks {
		var details taskDetails
		if t.Container != nil {
			details.container = *t.Container
		}
		d.tasks = append(d.tasks, &Task{
			id:               t.ID,
			no:               t.No,
			action:           t.Action,
			facilityID:       t.FacilityID,
			status:           t.Status,
			details:          details,
			attributedDriver: derefDriver(t.AttributedDriver),
			attributedTruck:  derefTruck(t.AttributedTruck),
		})
	}
	for _, a := range s.Assignments {
		d.assignments = append(d.assignments, &Assignment{
			id:       a.ID,
			driverID: derefDriver(a.DriverID),
			truckID:  derefTruck(a.TruckID),
			status:   a.Status,
		})
	}
	for _, e := range s.ChassisEpisodes {
		var n ChassisNumber
		if e.ChassisNumber != nil {
			n = *e.ChassisNumber
		}
		d.chassisEpisodes = append(d.chassisEpisodes, &ChassisCustodyEpisode{
			id:            e.ID,
			status:        e.Status,
			sourcedFrom:   e.SourcedFrom,
			stationedAt:   e.StationedAt,
			chassisNumber: n,
		})
	}
	for _, e := range s.ContainerEpisodes {
		d.containerEpisodes = append(d.containerEpisodes, &ContainerCustodyEpisode{
			id:              e.ID,
			status:          e.Status,
			sourcedFrom:     e.SourcedFrom,
			stationedAt:     e.StationedAt,
			containerNumber: e.ContainerNumber,
		})
	}
	return d
}

// --- zero-value <-> pointer helpers at the persistence boundary ----------- //

func nonZeroDriver(d DriverID) *DriverID {
	if d.IsZero() {
		return nil
	}
	return &d
}

func nonZeroTruck(t TruckID) *TruckID {
	if t.IsZero() {
		return nil
	}
	return &t
}

func nonZeroSourceQuote(s SourceQuoteID) *SourceQuoteID {
	if s.IsZero() {
		return nil
	}
	return &s
}

func nonZeroContainer(c ContainerNumber) *ContainerNumber {
	if c.IsZero() {
		return nil
	}
	return &c
}

func nonZeroChassis(c ChassisNumber) *ChassisNumber {
	if c.IsZero() {
		return nil
	}
	return &c
}
