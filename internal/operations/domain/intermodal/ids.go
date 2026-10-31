// Package intermodal is the domain model for the operations context's
// intermodal drayage Dispatch aggregate (rdt: modules/operations/domain/
// aggregates/intermodal_dispatch). Naming the package after the move family
// (intermodal) rather than the aggregate keeps intermodal.Dispatch stutter-free
// and leaves room for sibling move types (e.g. flatbed). This file holds the
// aggregate's typed identities (rdt: identity.py).
//
// Each id is a distinct named type embedding domain.ID, so the compiler keeps,
// say, a DriverID out of a slot expecting a TruckID, while the UUID plumbing
// lives once in domain.ID. The New* constructors mint a fresh v7 id; the Parse*
// constructors validate external input at the boundary.
package intermodal

import (
	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

// DispatchID identifies a Dispatch aggregate.
type DispatchID struct{ domain.ID }

// DispatcherID identifies the dispatcher a Dispatch is delegated to.
type DispatcherID struct{ domain.ID }

// DriverID identifies a driver assigned to carry out a Dispatch.
type DriverID struct{ domain.ID }

// TruckID identifies a truck assigned to carry out a Dispatch.
type TruckID struct{ domain.ID }

// AssignmentID identifies a driver+truck Assignment within a Dispatch.
type AssignmentID struct{ domain.ID }

// FacilityID identifies a facility (ramp, dock door) an action happens at.
type FacilityID struct{ domain.ID }

// SourceQuoteID links a Dispatch back to the booking quote it originated from.
type SourceQuoteID struct{ domain.ID }

// Task ID identifies one ordered step in a Dispatch's agenda.
type TaskID struct{ domain.ID }

// ChassisCustodyEpisodeID identifies a chassis custody episode.
type ChassisCustodyEpisodeID struct{ domain.ID }

// ContainerCustodyEpisodeID identifies a container custody episode.
type ContainerCustodyEpisodeID struct{ domain.ID }

func NewDispatchID() DispatchID       { return DispatchID{domain.NewID()} }
func NewDispatcherID() DispatcherID   { return DispatcherID{domain.NewID()} }
func NewDriverID() DriverID           { return DriverID{domain.NewID()} }
func NewTruckID() TruckID             { return TruckID{domain.NewID()} }
func NewAssignmentID() AssignmentID   { return AssignmentID{domain.NewID()} }
func NewFacilityID() FacilityID       { return FacilityID{domain.NewID()} }
func NewSourceQuoteID() SourceQuoteID { return SourceQuoteID{domain.NewID()} }
func NewTaskID() TaskID               { return TaskID{domain.NewID()} }
func NewChassisCustodyEpisodeID() ChassisCustodyEpisodeID {
	return ChassisCustodyEpisodeID{domain.NewID()}
}
func NewContainerCustodyEpisodeID() ContainerCustodyEpisodeID {
	return ContainerCustodyEpisodeID{domain.NewID()}
}

func ParseDispatchID(s string) (DispatchID, error) {
	id, err := domain.ParseID(s)
	return DispatchID{id}, err
}

func ParseDispatcherID(s string) (DispatcherID, error) {
	id, err := domain.ParseID(s)
	return DispatcherID{id}, err
}

func ParseDriverID(s string) (DriverID, error) {
	id, err := domain.ParseID(s)
	return DriverID{id}, err
}

func ParseTruckID(s string) (TruckID, error) {
	id, err := domain.ParseID(s)
	return TruckID{id}, err
}

func ParseAssignmentID(s string) (AssignmentID, error) {
	id, err := domain.ParseID(s)
	return AssignmentID{id}, err
}

func ParseFacilityID(s string) (FacilityID, error) {
	id, err := domain.ParseID(s)
	return FacilityID{id}, err
}

func ParseSourceQuoteID(s string) (SourceQuoteID, error) {
	id, err := domain.ParseID(s)
	return SourceQuoteID{id}, err
}

func ParseTaskID(s string) (TaskID, error) {
	id, err := domain.ParseID(s)
	return TaskID{id}, err
}

func ParseChassisCustodyEpisodeID(s string) (ChassisCustodyEpisodeID, error) {
	id, err := domain.ParseID(s)
	return ChassisCustodyEpisodeID{id}, err
}

func ParseContainerCustodyEpisodeID(s string) (ContainerCustodyEpisodeID, error) {
	id, err := domain.ParseID(s)
	return ContainerCustodyEpisodeID{id}, err
}

// FromUUID wrappers rebuild a typed id from a UUID in hand (reconstitution from
// persisted rows), validating it is non-zero.

func DispatchIDFromUUID(v uuid.UUID) (DispatchID, error) {
	id, err := domain.FromUUID(v)
	return DispatchID{id}, err
}

func DispatcherIDFromUUID(v uuid.UUID) (DispatcherID, error) {
	id, err := domain.FromUUID(v)
	return DispatcherID{id}, err
}

func DriverIDFromUUID(v uuid.UUID) (DriverID, error) {
	id, err := domain.FromUUID(v)
	return DriverID{id}, err
}

func TruckIDFromUUID(v uuid.UUID) (TruckID, error) {
	id, err := domain.FromUUID(v)
	return TruckID{id}, err
}

func AssignmentIDFromUUID(v uuid.UUID) (AssignmentID, error) {
	id, err := domain.FromUUID(v)
	return AssignmentID{id}, err
}

func FacilityIDFromUUID(v uuid.UUID) (FacilityID, error) {
	id, err := domain.FromUUID(v)
	return FacilityID{id}, err
}

func SourceQuoteIDFromUUID(v uuid.UUID) (SourceQuoteID, error) {
	id, err := domain.FromUUID(v)
	return SourceQuoteID{id}, err
}

func AgendaItemIDFromUUID(v uuid.UUID) (TaskID, error) {
	id, err := domain.FromUUID(v)
	return TaskID{id}, err
}

func ChassisCustodyEpisodeIDFromUUID(v uuid.UUID) (ChassisCustodyEpisodeID, error) {
	id, err := domain.FromUUID(v)
	return ChassisCustodyEpisodeID{id}, err
}

func ContainerCustodyEpisodeIDFromUUID(v uuid.UUID) (ContainerCustodyEpisodeID, error) {
	id, err := domain.FromUUID(v)
	return ContainerCustodyEpisodeID{id}, err
}
