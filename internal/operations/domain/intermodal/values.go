package intermodal

import (
	"time"
)

// Supporting VOs
type DocumentKind string

const (
	DocGateTicket      DocumentKind = "gate_ticket"
	DocProofOfDelivery DocumentKind = "proof_of_delivery"
)

func (k DocumentKind) Valid() bool {
	switch k {
	case DocGateTicket, DocProofOfDelivery:
		return true
	default:
		return false
	}
}

type DocumentRef struct {
	ID         DocumentID
	Kind       DocumentKind
	CapturedAt time.Time
}

type AppointmentWindow struct {
	Start     time.Time
	End       time.Time
	Reference string
}

func NewAppointmentWindow(start, end time.Time, ref string) (AppointmentWindow, error) {
	return AppointmentWindow{Start: start, End: end, Reference: ref}, nil
}

type Coordinates struct {
	Lat float64
	Lng float64
}

func NewCoordinates(lat, lng float64) (Coordinates, error) {
	return Coordinates{Lat: lat, Lng: lng}, nil
}

type Address struct {
	Line1      string
	Line2      string
	City       string
	Region     string
	PostalCode string
	Country    string
}

func NewAddress(line1, line2, city, region, postal, country string) (Address, error) {
	a := Address{
		Line1:      line1,
		Line2:      line2,
		City:       city,
		Region:     region,
		PostalCode: postal,
		Country:    country,
	}
	return a, nil
}

type LocationKind string

const (
	LocationRailRamp         LocationKind = "rail_ramp"
	LocationCustomerFacility LocationKind = "customer_facility"
	LocationChassisPool      LocationKind = "chassis_pool"
	LocationContainerDepot   LocationKind = "container_depot"
	LocationCarrierYard      LocationKind = "carrier_yard"
)

func (k LocationKind) Valid() bool {
	switch k {
	case LocationRailRamp, LocationCustomerFacility, LocationChassisPool, LocationContainerDepot, LocationCarrierYard:
		return true
	default:
		return false
	}
}

type LocationRef struct {
	ID      LocationID
	Kind    LocationKind
	Name    string
	Address Address
	Geo     Coordinates
	Code    string
}

type ContainerNumber string // ISO 6346 U

func NewContainerNumber(s string) (ContainerNumber, error) {
	return ContainerNumber(s), nil
}

type ChassisNumber string // ISO 6346 Z

func NewChassisNumber(s string) (ChassisNumber, error) {
	return ChassisNumber(s), nil
}

type LoadState int

const (
	Empty LoadState = iota + 1
	Loaded
)

type ContainerRef struct {
	Number ContainerNumber
	State  LoadState
}

type ChassisRef struct {
	Number ChassisNumber
}

// Task entity
type TaskStatus string

const (
	TaskPlanned   TaskStatus = "planned"
	TaskCompleted TaskStatus = "completed"
	TaskSkipped   TaskStatus = "skipped"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskPlanned, TaskCompleted, TaskSkipped:
		return true
	default:
		return false
	}
}

type TaskKind string

const (
	TaskPickUpContainer  TaskKind = "pick_up_container"
	TaskDropOffContainer TaskKind = "deliver_container"
	TaskPickUpChassis    TaskKind = "pick_up_chassis"
	TaskDropOffChassis   TaskKind = "drop_off_chassis"
)

func (k TaskKind) Valid() bool {
	switch k {
	case TaskPickUpContainer, TaskDropOffContainer, TaskPickUpChassis, TaskDropOffChassis:
		return true
	default:
		return false
	}
}

type TaskSubject interface{ isTaskSubject() }

type ContainerSubject struct {
	container ContainerRef
	chassis   ChassisRef
}

type ChassisSubject struct {
	chassis ChassisRef
}

type NoSubject struct{}

func (ContainerSubject) isTaskSubject() {}
func (ChassisSubject) isTaskSubject()   {}
func (NoSubject) isTaskSubject()        {}

// Stop entity
type StopStatus string

const (
	StopPending  StopStatus = "pending"
	StopArrived  StopStatus = "arrived"
	StopDeparted StopStatus = "departed"
	StopSkipped  StopStatus = "skipped"
)

func (s StopStatus) Valid() bool {
	switch s {
	case StopPending, StopArrived, StopDeparted, StopSkipped:
		return true
	default:
		return false
	}
}

// Assignment entity
type AssignmentStatus string

const (
	AssignmentOffered   AssignmentStatus = "offered"
	AssignmentAccepted  AssignmentStatus = "accepted"
	AssignmentDeclined  AssignmentStatus = "declined"
	AssignmentWithdrawn AssignmentStatus = "withdrawn"
	AssignmentReleased  AssignmentStatus = "released"
)

func (s AssignmentStatus) Valid() bool {
	switch s {
	case AssignmentOffered, AssignmentAccepted, AssignmentDeclined, AssignmentWithdrawn, AssignmentReleased:
		return true
	default:
		return false
	}
}

// Run aggregate
type RunStatus string

const (
	RunPlanned    RunStatus = "planned"
	RunApproved   RunStatus = "approved"
	RunInProgress RunStatus = "in_progress"
	RunCompleted  RunStatus = "completed"
	RunAborted    RunStatus = "aborted"
	RunCanceled   RunStatus = "canceled"
)

func (s RunStatus) Valid() bool {
	switch s {
	case RunPlanned, RunApproved, RunInProgress, RunCompleted, RunAborted, RunCanceled:
		return true
	default:
		return false
	}
}

// RunPlan
type RunPlan struct {
	Stops []Stop
}
