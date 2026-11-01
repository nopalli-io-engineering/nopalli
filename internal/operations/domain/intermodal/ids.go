package intermodal

import (
	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/domain"
)

type RunID struct{ domain.ID }
type StopID struct{ domain.ID }
type TaskID struct{ domain.ID }
type AssignmentID struct{ domain.ID }
type DriverID struct{ domain.ID }
type TruckID struct{ domain.ID }
type LocationID struct{ domain.ID }
type DocumentID struct{ domain.ID }

func NewRunID() RunID               { return RunID{domain.NewID()} }
func NewStopID() StopID             { return StopID{domain.NewID()} }
func NewTaskID() TaskID             { return TaskID{domain.NewID()} }
func NewAssignmentID() AssignmentID { return AssignmentID{domain.NewID()} }
func NewDriverID() DriverID         { return DriverID{domain.NewID()} }
func NewTruckID() TruckID           { return TruckID{domain.NewID()} }
func NewLocationID() LocationID     { return LocationID{domain.NewID()} }
func NewDocumentID() DocumentID     { return DocumentID{domain.NewID()} }

func ParseRunID(s string) (RunID, error) {
	id, err := domain.ParseID(s)
	return RunID{id}, err
}

func ParseStopID(s string) (StopID, error) {
	id, err := domain.ParseID(s)
	return StopID{id}, err
}

func ParseTaskID(s string) (TaskID, error) {
	id, err := domain.ParseID((s))
	return TaskID{id}, err
}

func ParseAssignmentID(s string) (AssignmentID, error) {
	id, err := domain.ParseID((s))
	return AssignmentID{id}, err
}

func ParseDriverID(s string) (DriverID, error) {
	id, err := domain.ParseID((s))
	return DriverID{id}, err
}

func ParseTruckID(s string) (TruckID, error) {
	id, err := domain.ParseID((s))
	return TruckID{id}, err
}

func ParseLocationID(s string) (LocationID, error) {
	id, err := domain.ParseID((s))
	return LocationID{id}, err
}

func ParseDocumentID(s string) (DocumentID, error) {
	id, err := domain.ParseID((s))
	return DocumentID{id}, err
}

func RunIDFromUUID(v uuid.UUID) (RunID, error) {
	id, err := domain.FromUUID(v)
	return RunID{id}, err
}

func StopIDFromUUID(v uuid.UUID) (StopID, error) {
	id, err := domain.FromUUID(v)
	return StopID{id}, err
}

func TaskIDFromUUID(v uuid.UUID) (TaskID, error) {
	id, err := domain.FromUUID(v)
	return TaskID{id}, err
}

func AssignmentIDFromUUID(v uuid.UUID) (AssignmentID, error) {
	id, err := domain.FromUUID(v)
	return AssignmentID{id}, err
}

func DriverIDFromUUID(v uuid.UUID) (DriverID, error) {
	id, err := domain.FromUUID(v)
	return DriverID{id}, err
}

func TruckIDFromUUID(v uuid.UUID) (TruckID, error) {
	id, err := domain.FromUUID(v)
	return TruckID{id}, err
}

func LocationIDFromUUID(v uuid.UUID) (LocationID, error) {
	id, err := domain.FromUUID(v)
	return LocationID{id}, err
}

func DocumentIDFromUUID(v uuid.UUID) (DocumentID, error) {
	id, err := domain.FromUUID(v)
	return DocumentID{id}, err
}
