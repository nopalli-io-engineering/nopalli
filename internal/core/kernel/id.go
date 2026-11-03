// Package kernel holds the primitives every bounded context's domain layer is
// built on: identity, domain events, the aggregate-root base, and the domain
// error type. It is the Go counterpart of rdt's app/core/domain package.
package kernel

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidID is the sentinel wrapped by every ID parsing failure, so callers
// can match with errors.Is without depending on the message text.
var ErrInvalidID = errors.New("invalid id")

// ID is the shared identity primitive (rdt: core.domain.identity.EntityID).
//
// It is never used on its own. Each aggregate embeds it in a distinct named
// type — type DispatchID struct{ kernel.ID } — so DispatchID and DriverID are
// different types the compiler keeps apart, while the UUID plumbing (v7
// generation, parsing, zero-checking, stringification) lives here once.
//
// The wrapped value is unexported: an ID can only be built through NewID or
// ParseID, so a zero/invalid UUID can never masquerade as a valid identity.
type ID struct {
	v uuid.UUID
}

// NewID mints a time-ordered (v7) identity. It mirrors rdt's EntityID.generate,
// which used uuid7 so ids sort by creation time — valuable for index locality
// and for human-readable ordering of rows.
func NewID() ID {
	return ID{v: uuid.Must(uuid.NewV7())}
}

// ParseID validates a string as a non-zero UUID. The zero UUID is rejected
// (rdt: EntityID forbids the all-zero uuid) because it is indistinguishable
// from "unset" and must never stand in for a real identity.
func ParseID(raw string) (ID, error) {
	v, err := uuid.Parse(raw)
	if err != nil {
		return ID{}, fmt.Errorf("%w: %q is not a uuid", ErrInvalidID, raw)
	}
	if v == uuid.Nil {
		return ID{}, fmt.Errorf("%w: must not be the zero uuid", ErrInvalidID)
	}
	return ID{v: v}, nil
}

// FromUUID wraps an existing UUID as an ID, validating it is non-zero. Use it
// when reconstituting an aggregate from persisted rows, where the UUID is
// already in hand and a string round-trip through ParseID would be wasteful.
func FromUUID(v uuid.UUID) (ID, error) {
	if v == uuid.Nil {
		return ID{}, fmt.Errorf("%w: must not be the zero uuid", ErrInvalidID)
	}
	return ID{v: v}, nil
}

// UUID returns the underlying value, for persistence and event payloads.
func (i ID) UUID() uuid.UUID { return i.v }

// IsZero reports whether this is the zero value (no identity assigned).
func (i ID) IsZero() bool { return i.v == uuid.Nil }

// String renders the canonical UUID text, or "" for the zero value.
func (i ID) String() string {
	if i.v == uuid.Nil {
		return ""
	}
	return i.v.String()
}

// Common IDs
type TenantID struct{ uuid.UUID }
type UserID struct{ uuid.UUID }

func NewTenantID() TenantID { return TenantID{uuid.Must(uuid.NewV7())} }
func NewUserID() UserID     { return UserID{uuid.Must(uuid.NewV7())} }

func (id TenantID) IsZero() bool { return id.UUID == uuid.Nil }
func (id UserID) IsZero() bool   { return id.UUID == uuid.Nil }

func ParseTenantID(s string) (TenantID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return TenantID{}, fmt.Errorf("invalid tenant id %q: %w", s, err)
	}
	return TenantID{u}, nil
}

func ParseUserID(s string) (UserID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, fmt.Errorf("invalid user id %q: %w", s, err)
	}
	return UserID{u}, nil
}
