package intermodal

import (
	"fmt"

	"github.com/google/uuid"
)

type RunID struct{ uuid.UUID }
type StopID struct{ uuid.UUID }
type TaskID struct{ uuid.UUID }
type AssignmentID struct{ uuid.UUID }
type DriverID struct{ uuid.UUID }
type TruckID struct{ uuid.UUID }
type LocationID struct{ uuid.UUID }
type DocumentID struct{ uuid.UUID }

func NewRunID() RunID               { return RunID{uuid.Must(uuid.NewV7())} }
func NewStopID() StopID             { return StopID{uuid.Must(uuid.NewV7())} }
func NewTaskID() TaskID             { return TaskID{uuid.Must(uuid.NewV7())} }
func NewAssignmentID() AssignmentID { return AssignmentID{uuid.Must(uuid.NewV7())} }
func NewDriverID() DriverID         { return DriverID{uuid.Must(uuid.NewV7())} }
func NewTruckID() TruckID           { return TruckID{uuid.Must(uuid.NewV7())} }
func NewLocationID() LocationID     { return LocationID{uuid.Must(uuid.NewV7())} }
func NewDocumentID() DocumentID     { return DocumentID{uuid.Must(uuid.NewV7())} }

func (id RunID) IsZero() bool        { return id.UUID == uuid.Nil }
func (id StopID) IsZero() bool       { return id.UUID == uuid.Nil }
func (id TaskID) IsZero() bool       { return id.UUID == uuid.Nil }
func (id AssignmentID) IsZero() bool { return id.UUID == uuid.Nil }
func (id DriverID) IsZero() bool     { return id.UUID == uuid.Nil }
func (id TruckID) IsZero() bool      { return id.UUID == uuid.Nil }
func (id LocationID) IsZero() bool   { return id.UUID == uuid.Nil }
func (id DocumentID) IsZero() bool   { return id.UUID == uuid.Nil }

func (id RunID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id StopID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id TaskID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id AssignmentID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id DriverID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id TruckID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id LocationID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func (id DocumentID) String() string {
	if id.UUID == uuid.Nil {
		return ""
	}
	return id.UUID.String()
}

func ParseRunID(s string) (RunID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return RunID{}, fmt.Errorf("invalid run id %q: %w", s, err)
	}
	return RunID{u}, nil
}

func ParseStopID(s string) (StopID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return StopID{}, fmt.Errorf("invalid stop id %q: %w", s, err)
	}
	return StopID{u}, nil
}

func ParseTaskID(s string) (TaskID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return TaskID{}, fmt.Errorf("invalid task id %q: %w", s, err)
	}
	return TaskID{u}, nil
}

func ParseAssignmentID(s string) (AssignmentID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return AssignmentID{}, fmt.Errorf("invalid assignment id %q: %w", s, err)
	}
	return AssignmentID{u}, nil
}

func ParseDriverID(s string) (DriverID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return DriverID{}, fmt.Errorf("invalid driver id %q: %w", s, err)
	}
	return DriverID{u}, nil
}

func ParseTruckID(s string) (TruckID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return TruckID{}, fmt.Errorf("invalid truck id %q: %w", s, err)
	}
	return TruckID{u}, nil
}

func ParseLocationID(s string) (LocationID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return LocationID{}, fmt.Errorf("invalid location id %q: %w", s, err)
	}
	return LocationID{u}, nil
}

func ParseDocumentID(s string) (DocumentID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return DocumentID{}, fmt.Errorf("invalid document id %q: %w", s, err)
	}
	return DocumentID{u}, nil
}
