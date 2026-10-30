package db

import "errors"

// ErrConcurrencyConflict is returned by a repository's Save when the row's
// version no longer matches the aggregate's — another writer committed first
// (optimistic lock). Generic across contexts, so it lives here rather than in
// any one module's adapter.
var ErrConcurrencyConflict = errors.New("db: optimistic concurrency conflict")
