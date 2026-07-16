package munki

import "context"

// hostStateStore persists observed Munki host state.
type hostStateStore interface {
	UpsertHostObservation(ctx context.Context, observation HostObservation) error
	ClearHostObservation(ctx context.Context, hostID int64) error
	ReplaceHostItems(ctx context.Context, hostID int64, items []Item) error
}

// DetailIngestor projects osquery munki_info and munki_installs detail rows into
// observed host state. It is registered onto the osquery projector from the
// wiring layer so the projector core stays free of Munki types.
type DetailIngestor struct {
	store hostStateStore
}

// NewDetailIngestor returns an ingestor that writes observed state to store.
func NewDetailIngestor(store hostStateStore) *DetailIngestor {
	return &DetailIngestor{store: store}
}

// IngestInfo records a host's munki_info observation, clearing it when the host
// reports no Munki data.
func (i *DetailIngestor) IngestInfo(ctx context.Context, hostID int64, rows []map[string]string) error {
	status, ok := HostStatusFromInfoRows(hostID, rows)
	if !ok {
		return i.store.ClearHostObservation(ctx, hostID)
	}
	return i.store.UpsertHostObservation(ctx, status)
}

// IngestInstalls records the managed-install items a host reports.
func (i *DetailIngestor) IngestInstalls(ctx context.Context, hostID int64, rows []map[string]string) error {
	return i.store.ReplaceHostItems(ctx, hostID, ItemsFromInstallRows(hostID, rows))
}
