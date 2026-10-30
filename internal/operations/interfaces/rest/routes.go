package rest

import "net/http"

// RegisterRoutes mounts the operations command endpoints on mux, using Go
// 1.22 method+path patterns. Each POST is a state transition modeled as a
// sub-resource of the dispatch.
func RegisterRoutes(mux *http.ServeMux, h *Handlers) {
	const base = "/v1/operations/dispatches"

	mux.HandleFunc("GET "+base, h.ListDispatches)
	mux.HandleFunc("GET "+base+"/{dispatchID}", h.GetDispatch)

	mux.HandleFunc("POST "+base, h.CreateDispatch)

	mux.HandleFunc("POST "+base+"/{dispatchID}/delegation", h.Delegate)
	mux.HandleFunc("POST "+base+"/{dispatchID}/assignment", h.Assign)
	mux.HandleFunc("POST "+base+"/{dispatchID}/approval", h.Approve)
	mux.HandleFunc("POST "+base+"/{dispatchID}/start", h.Start)
	mux.HandleFunc("POST "+base+"/{dispatchID}/verification", h.Verify)

	mux.HandleFunc("POST "+base+"/{dispatchID}/chassis-acquisition", h.AcquireChassis)
	mux.HandleFunc("POST "+base+"/{dispatchID}/container-pickup", h.PickUpContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/premounted-container-pickup", h.PickUpPremountedContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/container-delivery", h.DeliverContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/hold", h.Hold)
	mux.HandleFunc("POST "+base+"/{dispatchID}/container-take", h.TakeContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/container-drop", h.DropContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/premounted-container-drop", h.DropPremountedContainer)
	mux.HandleFunc("POST "+base+"/{dispatchID}/chassis-return", h.ReturnChassis)
}
