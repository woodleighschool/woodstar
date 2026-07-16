package munki

import "context"

type softwareDeleter interface {
	Delete(ctx context.Context, softwareID int64) error
	DeleteMany(ctx context.Context, softwareIDs []int64) (int, error)
}

// SoftwareDeletionService removes software and signals when cascaded package
// deletion changes the distribution workers' desired installer set.
type SoftwareDeletionService struct {
	software               softwareDeleter
	desiredPackagesChanged func()
}

// NewSoftwareDeletionService returns the cross-entity software deletion service.
func NewSoftwareDeletionService(
	software softwareDeleter,
	desiredPackagesChanged func(),
) *SoftwareDeletionService {
	return &SoftwareDeletionService{
		software:               software,
		desiredPackagesChanged: desiredPackagesChanged,
	}
}

// Delete removes one software record and its packages.
func (s *SoftwareDeletionService) Delete(ctx context.Context, id int64) error {
	if err := s.software.Delete(ctx, id); err != nil {
		return err
	}
	s.desiredPackagesChanged()
	return nil
}

// DeleteMany removes software records and their packages.
func (s *SoftwareDeletionService) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	deleted, err := s.software.DeleteMany(ctx, ids)
	if err == nil && deleted > 0 {
		s.desiredPackagesChanged()
	}
	return deleted, err
}
