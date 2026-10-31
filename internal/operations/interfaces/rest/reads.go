package rest

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/nopalli-io/nopalli/internal/core/interface/httpx"
	"github.com/nopalli-io/nopalli/internal/operations/application"
	"github.com/nopalli-io/nopalli/internal/operations/domain/intermodal_old"
)

// Query (read) endpoints. Reads are served from flat projections, never by
// hydrating the aggregate, and are scoped by Visibility (filter, don't reject):
// an out-of-scope principal gets an empty list / a 404, never a 403.

// ListDispatches handles GET /dispatches?status=&limit=&offset=.
func (h *Handlers) ListDispatches(w http.ResponseWriter, r *http.Request) {
	filter, ok := parseListFilter(w, r)
	if !ok {
		return
	}
	summaries, err := h.queries.ListDispatches(r.Context(), h.visibility(r), filter)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	out := make([]DispatchSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, newDispatchSummaryResponse(s))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// GetDispatch handles GET /dispatches/{dispatchID}. An unknown or out-of-scope
// id is a 404 — the read side returns nil for both, so an out-of-scope id never
// reads differently from a missing one.
func (h *Handlers) GetDispatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("dispatchID"))
	if err != nil {
		badRequest(w, "invalid dispatch id")
		return
	}
	detail, err := h.queries.GetDispatch(r.Context(), id, h.visibility(r))
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if detail == nil {
		httpx.WriteProblem(w, http.StatusNotFound, "dispatch_not_found", "dispatch not found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newDispatchDetailResponse(*detail))
}

// parseListFilter reads the status/limit/offset query params. A non-numeric or
// negative page bound and an unknown status are 400s; the adapter still clamps
// limit/offset to its own bounds.
func parseListFilter(w http.ResponseWriter, r *http.Request) (application.ListFilter, bool) {
	q := r.URL.Query()
	var filter application.ListFilter

	if raw := q.Get("status"); raw != "" {
		status := intermodal_old.Status(raw)
		if !status.Valid() {
			badRequest(w, "invalid status")
			return application.ListFilter{}, false
		}
		filter.Status = &status
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			badRequest(w, "invalid limit")
			return application.ListFilter{}, false
		}
		filter.Limit = n
	}
	if raw := q.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			badRequest(w, "invalid offset")
			return application.ListFilter{}, false
		}
		filter.Offset = n
	}
	return filter, true
}

// visibility derives the read scope for the request. Until auth is wired every
// read is in scope (parity with the command side's AllowAll); this is the seam
// where a principal's read_all grant will yield VisibleAll vs VisibleNothing.
func (h *Handlers) visibility(_ *http.Request) application.Visibility {
	return application.VisibleAll()
}
