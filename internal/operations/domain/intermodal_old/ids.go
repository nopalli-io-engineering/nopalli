// Package intermodal is the domain model for the operations context's
// intermodal drayage Dispatch aggregate (rdt: modules/operations/domain/
// aggregates/intermodal_dispatch). Naming the package after the move family
// (intermodal) rather than the aggregate keeps intermodal_old.Dispatch stutter-free
// and leaves room for sibling move types (e.g. flatbed). This file holds the
// aggregate's typed identities (rdt: identity.py).
//
// Each id is a distinct named type embedding kernel.ID, so the compiler keeps,
// say, a DriverID out of a slot expecting a TruckID, while the UUID plumbing
// lives once in kernel.ID. The New* constructors mint a fresh v7 id; the Parse*
// constructors validate external input at the boundary.
package intermodal_old

import (
	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

// DispatchID identifies a Dispatch aggregate.
type DispatchID struct{ kernel.ID }

// DispatcherID identifies the dispatcher a Dispatch is delegated to.
type DispatcherID struct{ kernel.ID }

// DriverID identifies a driver assigned to carry out a Dispatch.
type DriverID struct{ kernel.ID }

// TruckID identifies a truck assigned to carry out a Dispatch.
type TruckID struct{ kernel.ID }

// AssignmentID identifies a driver+truck Assignment within a Dispatch.
type AssignmentID struct{ kernel.ID }

// FacilityID identifies a facility (ramp, dock door) an action happens at.
type FacilityID struct{ kernel.ID }

// SourceQuoteID links a Dispatch back to the booking quote it originated from.
type SourceQuoteID struct{ kernel.ID }

// Task ID identifies one ordered step in a Dispatch's tasks.
type TaskID struct{ kernel.ID }

// ChassisCustodyEpisodeID identifies a chassis custody episode.
type ChassisCustodyEpisodeID struct{ kernel.ID }

// ContainerCustodyEpisodeID identifies a container custody episode.
type ContainerCustodyEpisodeID struct{ kernel.ID }

func NewDispatchID() DispatchID       { return DispatchID{kernel.NewID()} }
func NewDispatcherID() DispatcherID   { return DispatcherID{kernel.NewID()} }
func NewDriverID() DriverID           { return DriverID{kernel.NewID()} }
func NewTruckID() TruckID             { return TruckID{kernel.NewID()} }
func NewAssignmentID() AssignmentID   { return AssignmentID{kernel.NewID()} }
func NewFacilityID() FacilityID       { return FacilityID{kernel.NewID()} }
func NewSourceQuoteID() SourceQuoteID { return SourceQuoteID{kernel.NewID()} }
func NewTaskID() TaskID               { return TaskID{kernel.NewID()} }
func NewChassisCustodyEpisodeID() ChassisCustodyEpisodeID {
	return ChassisCustodyEpisodeID{kernel.NewID()}
}
func NewContainerCustodyEpisodeID() ContainerCustodyEpisodeID {
	return ContainerCustodyEpisodeID{kernel.NewID()}
}

func ParseDispatchID(s string) (DispatchID, error) {
	id, err := kernel.ParseID(s)
	return DispatchID{id}, err
}

func ParseDispatcherID(s string) (DispatcherID, error) {
	id, err := kernel.ParseID(s)
	return DispatcherID{id}, err
}

func ParseDriverID(s string) (DriverID, error) {
	id, err := kernel.ParseID(s)
	return DriverID{id}, err
}

func ParseTruckID(s string) (TruckID, error) {
	id, err := kernel.ParseID(s)
	return TruckID{id}, err
}

func ParseAssignmentID(s string) (AssignmentID, error) {
	id, err := kernel.ParseID(s)
	return AssignmentID{id}, err
}

func ParseFacilityID(s string) (FacilityID, error) {
	id, err := kernel.ParseID(s)
	return FacilityID{id}, err
}

func ParseSourceQuoteID(s string) (SourceQuoteID, error) {
	id, err := kernel.ParseID(s)
	return SourceQuoteID{id}, err
}

func ParseTaskID(s string) (TaskID, error) {
	id, err := kernel.ParseID(s)
	return TaskID{id}, err
}

func ParseChassisCustodyEpisodeID(s string) (ChassisCustodyEpisodeID, error) {
	id, err := kernel.ParseID(s)
	return ChassisCustodyEpisodeID{id}, err
}

func ParseContainerCustodyEpisodeID(s string) (ContainerCustodyEpisodeID, error) {
	id, err := kernel.ParseID(s)
	return ContainerCustodyEpisodeID{id}, err
}

// FromUUID wrappers rebuild a typed id from a UUID in hand (reconstitution from
// persisted rows), validating it is non-zero.

func DispatchIDFromUUID(v uuid.UUID) (DispatchID, error) {
	id, err := kernel.FromUUID(v)
	return DispatchID{id}, err
}

func DispatcherIDFromUUID(v uuid.UUID) (DispatcherID, error) {
	id, err := kernel.FromUUID(v)
	return DispatcherID{id}, err
}

func DriverIDFromUUID(v uuid.UUID) (DriverID, error) {
	id, err := kernel.FromUUID(v)
	return DriverID{id}, err
}

func TruckIDFromUUID(v uuid.UUID) (TruckID, error) {
	id, err := kernel.FromUUID(v)
	return TruckID{id}, err
}

func AssignmentIDFromUUID(v uuid.UUID) (AssignmentID, error) {
	id, err := kernel.FromUUID(v)
	return AssignmentID{id}, err
}

func FacilityIDFromUUID(v uuid.UUID) (FacilityID, error) {
	id, err := kernel.FromUUID(v)
	return FacilityID{id}, err
}

func SourceQuoteIDFromUUID(v uuid.UUID) (SourceQuoteID, error) {
	id, err := kernel.FromUUID(v)
	return SourceQuoteID{id}, err
}

func TaskIDFromUUID(v uuid.UUID) (TaskID, error) {
	id, err := kernel.FromUUID(v)
	return TaskID{id}, err
}

func ChassisCustodyEpisodeIDFromUUID(v uuid.UUID) (ChassisCustodyEpisodeID, error) {
	id, err := kernel.FromUUID(v)
	return ChassisCustodyEpisodeID{id}, err
}

func ContainerCustodyEpisodeIDFromUUID(v uuid.UUID) (ContainerCustodyEpisodeID, error) {
	id, err := kernel.FromUUID(v)
	return ContainerCustodyEpisodeID{id}, err
}
