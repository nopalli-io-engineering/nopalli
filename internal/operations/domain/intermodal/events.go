package intermodal

import (
	"time"

	"github.com/google/uuid"
	"github.com/nopalli-io/nopalli/internal/core/kernel"
)

type DriverAssigned struct {
	RunID  RunID
	Driver DriverID
	Truck  TruckID
	By     kernel.UserID
	At     time.Time
}

func (e DriverAssigned) SubjectID() uuid.UUID { return e.RunID.UUID }
