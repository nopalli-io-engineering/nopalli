package kernel

type Root struct {
	tenantID TenantID
	version  int
	events   []RecordedEvent
}

func NewRoot(t TenantID) Root      { return Root{tenantID: t} }
func (r *Root) TenantID() TenantID { return r.tenantID }
func (r *Root) Version() int       { return r.version }

func (r *Root) RecordEvents(events ...DomainEvent) {
	for _, e := range events {
		r.events = append(r.events, recordEvent(e))
	}
}

func (r *Root) Raise(events ...DomainEvent) {
	for _, e := range events {
		r.events = append(r.events, recordEvent(e))
	}
}

func (r *Root) Events() []RecordedEvent {
	out := make([]RecordedEvent, len(r.events))
	copy(out, r.events)
	return out
}

func (r *Root) PullEvents() []RecordedEvent {
	out := r.events
	r.events = nil
	return out
}

func (r *Root) ClearEvents() {
	r.events = nil
}
