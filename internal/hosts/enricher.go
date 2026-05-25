package hosts

import "context"

// DetailContributor adds capability-specific sections to a host detail response.
type DetailContributor[T any] interface {
	ContributeHostDetail(ctx context.Context, hostID int64, detail *T) error
}
