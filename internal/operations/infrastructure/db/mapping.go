package db

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Conversions between the aggregate's snapshot values and pgx's wire types.
// Typed ids bind/scan as uuid (pgtype.UUID); the equipment-number value objects
// and the task details map are encoded here, keeping the repository's SQL
// free of marshalling noise.

// identity is any typed id (all expose UUID via the embedded domain.ID).
type identity interface{ UUID() uuid.UUID }

// pgID binds a non-null typed id.
func pgID[T identity](v T) pgtype.UUID {
	return pgtype.UUID{Bytes: v.UUID(), Valid: true}
}

// pgIDOpt binds an optional typed id (nil → SQL NULL).
func pgIDOpt[T identity](p *T) pgtype.UUID {
	if p == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: (*p).UUID(), Valid: true}
}

// cursorToDB maps the aggregate's cursor (0 = none executed) to the nullable
// task_cursor column (NULL = none executed), matching rdt's column semantics.
func cursorToDB(c int) pgtype.Int4 {
	if c == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(c), Valid: true}
}

func cursorFromDB(v pgtype.Int4) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int32)
}

// --- optional typed ids from scanned uuids (DB values are trusted) -------- //

func optSourceQuote(p pgtype.UUID) *intermodal_old.SourceQuoteID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.SourceQuoteIDFromUUID(p.Bytes)
	return &id
}

func optDispatcher(p pgtype.UUID) *intermodal_old.DispatcherID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.DispatcherIDFromUUID(p.Bytes)
	return &id
}

func optAssignment(p pgtype.UUID) *intermodal_old.AssignmentID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.AssignmentIDFromUUID(p.Bytes)
	return &id
}

func optFacility(p pgtype.UUID) *intermodal_old.FacilityID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.FacilityIDFromUUID(p.Bytes)
	return &id
}

func optDriver(p pgtype.UUID) *intermodal_old.DriverID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.DriverIDFromUUID(p.Bytes)
	return &id
}

func optTruck(p pgtype.UUID) *intermodal_old.TruckID {
	if !p.Valid {
		return nil
	}
	id, _ := intermodal_old.TruckIDFromUUID(p.Bytes)
	return &id
}

// --- non-null typed ids from primary-key columns -------------------------- //

func mustFacilityID(p pgtype.UUID) intermodal_old.FacilityID {
	id, _ := intermodal_old.FacilityIDFromUUID(p.Bytes)
	return id
}

func mustTaskID(p pgtype.UUID) intermodal_old.TaskID {
	id, _ := intermodal_old.TaskIDFromUUID(p.Bytes)
	return id
}

func mustAssignmentID(p pgtype.UUID) intermodal_old.AssignmentID {
	id, _ := intermodal_old.AssignmentIDFromUUID(p.Bytes)
	return id
}

func mustChassisEpisodeID(p pgtype.UUID) intermodal_old.ChassisCustodyEpisodeID {
	id, _ := intermodal_old.ChassisCustodyEpisodeIDFromUUID(p.Bytes)
	return id
}

func mustContainerEpisodeID(p pgtype.UUID) intermodal_old.ContainerCustodyEpisodeID {
	id, _ := intermodal_old.ContainerCustodyEpisodeIDFromUUID(p.Bytes)
	return id
}

// --- value objects -------------------------------------------------------- //

func chassisText(c *intermodal_old.ChassisNumber) pgtype.Text {
	if c == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: c.String(), Valid: true}
}

func optChassis(t pgtype.Text) (*intermodal_old.ChassisNumber, error) {
	if !t.Valid || t.String == "" {
		return nil, nil
	}
	c, err := intermodal_old.NewChassisNumber(t.String)
	if err != nil {
		return nil, fmt.Errorf("operations: decode chassis number: %w", err)
	}
	return &c, nil
}

// encodeDetails serializes an task's details map to JSONB. Keyed
// "container_number" to match rdt's TaskDetailsType vocabulary.
func encodeDetails(c *intermodal_old.ContainerNumber) ([]byte, error) {
	m := map[string]string{}
	if c != nil {
		m["container_number"] = c.String()
	}
	return json.Marshal(m)
}

func decodeContainerFromDetails(raw []byte) (*intermodal_old.ContainerNumber, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("operations: decode task details: %w", err)
	}
	s, ok := m["container_number"]
	if !ok || s == "" {
		return nil, nil
	}
	c, err := intermodal_old.NewContainerNumber(s)
	if err != nil {
		return nil, fmt.Errorf("operations: decode container number: %w", err)
	}
	return &c, nil
}
