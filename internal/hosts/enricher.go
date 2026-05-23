package hosts

import "context"

// DetailEnricher attaches capability-specific data to a host detail response.
type DetailEnricher[T any] interface {
	EnrichHostDetail(ctx context.Context, hostID int64, detail *T) error
}
