package kernel

// Error is a domain rule violation (rdt: core.domain.error.DomainError). It
// carries a stable, machine-readable "namespace.code" — e.g. "operations.
// dispatch_action_out_of_sequence" — that interface layers map to HTTP status
// and that clients can switch on, independent of the human-readable Message.
//
// Each bounded context defines its own constructors over this type (rdt's
// per-module *Error subclasses); the namespace is the context, the code is the
// specific rule.
type Error struct {
	Namespace string
	Code      string
	Message   string
}

// Error implements the error interface with the human-readable message.
func (e *Error) Error() string { return e.Message }

// Qualified is the stable "namespace.code" identifier (rdt: DomainError.code).
func (e *Error) Qualified() string { return e.Namespace + "." + e.Code }

// NotAuthenticated: no authenticated principal for an operation that needs one
// (rdt maps this to HTTP 401).
func NotAuthenticated() *Error {
	return &Error{
		Namespace: "auth",
		Code:      "not_authenticated",
		Message:   "authentication required",
	}
}

// PermissionDenied: an authenticated principal lacks a required permission
// (rdt maps this to HTTP 403).
func PermissionDenied(permission string) *Error {
	return &Error{
		Namespace: "auth",
		Code:      "forbidden",
		Message:   "missing permission: " + permission,
	}
}
