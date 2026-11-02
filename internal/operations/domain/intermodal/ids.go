package intermodal

import (
	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

type RunID struct{ kernel.ID }
type StopID struct{ kernel.ID }
type TaskID struct{ kernel.ID }
type AssignmentID struct{ kernel.ID }
type DriverID struct{ kernel.ID }
type TruckID struct{ kernel.ID }
type LocationID struct{ kernel.ID }
type DocumentID struct{ kernel.ID }

func NewRunID() RunID               { return RunID{kernel.NewID()} }
func NewStopID() StopID             { return StopID{kernel.NewID()} }
func NewTaskID() TaskID             { return TaskID{kernel.NewID()} }
func NewAssignmentID() AssignmentID { return AssignmentID{kernel.NewID()} }
func NewDriverID() DriverID         { return DriverID{kernel.NewID()} }
func NewTruckID() TruckID           { return TruckID{kernel.NewID()} }
func NewLocationID() LocationID     { return LocationID{kernel.NewID()} }
func NewDocumentID() DocumentID     { return DocumentID{kernel.NewID()} }

func ParseRunID(s string) (RunID, error) {
	id, err := kernel.ParseID(s)
	return RunID{id}, err
}

func ParseStopID(s string) (StopID, error) {
	id, err := kernel.ParseID(s)
	return StopID{id}, err
}

func ParseTaskID(s string) (TaskID, error) {
	id, err := kernel.ParseID((s))
	return TaskID{id}, err
}

func ParseAssignmentID(s string) (AssignmentID, error) {
	id, err := kernel.ParseID((s))
	return AssignmentID{id}, err
}

func ParseDriverID(s string) (DriverID, error) {
	id, err := kernel.ParseID((s))
	return DriverID{id}, err
}

func ParseTruckID(s string) (TruckID, error) {
	id, err := kernel.ParseID((s))
	return TruckID{id}, err
}

func ParseLocationID(s string) (LocationID, error) {
	id, err := kernel.ParseID((s))
	return LocationID{id}, err
}

func ParseDocumentID(s string) (DocumentID, error) {
	id, err := kernel.ParseID((s))
	return DocumentID{id}, err
}

func RunIDFromUUID(v uuid.UUID) (RunID, error) {
	id, err := kernel.FromUUID(v)
	return RunID{id}, err
}

func StopIDFromUUID(v uuid.UUID) (StopID, error) {
	id, err := kernel.FromUUID(v)
	return StopID{id}, err
}

func TaskIDFromUUID(v uuid.UUID) (TaskID, error) {
	id, err := kernel.FromUUID(v)
	return TaskID{id}, err
}

func AssignmentIDFromUUID(v uuid.UUID) (AssignmentID, error) {
	id, err := kernel.FromUUID(v)
	return AssignmentID{id}, err
}

func DriverIDFromUUID(v uuid.UUID) (DriverID, error) {
	id, err := kernel.FromUUID(v)
	return DriverID{id}, err
}

func TruckIDFromUUID(v uuid.UUID) (TruckID, error) {
	id, err := kernel.FromUUID(v)
	return TruckID{id}, err
}

func LocationIDFromUUID(v uuid.UUID) (LocationID, error) {
	id, err := kernel.FromUUID(v)
	return LocationID{id}, err
}

func DocumentIDFromUUID(v uuid.UUID) (DocumentID, error) {
	id, err := kernel.FromUUID(v)
	return DocumentID{id}, err
}
