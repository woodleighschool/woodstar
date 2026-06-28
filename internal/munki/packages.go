package munki

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

type packageStore interface {
	List(context.Context, packages.PackageListParams) ([]packages.Package, int, error)
	Create(context.Context, packages.PackageCreateMutation) (*packages.Package, error)
	GetByID(context.Context, int64) (*packages.Package, error)
	Update(context.Context, int64, packages.PackageMutation) (*packages.Package, error)
	Delete(context.Context, int64) error
	DeleteMany(context.Context, []int64) (int, error)
	SetInstallerObject(context.Context, int64, int64) error
	ClearInstallerObject(context.Context, int64) error
}

// PackageService owns app-side Munki package mutations and signals when the
// repository installer set should be re-pushed to distribution workers.
type PackageService struct {
	store                  packageStore
	desiredPackagesChanged func()
}

// PackageServiceDependencies are the collaborators for PackageService.
type PackageServiceDependencies struct {
	Packages               packageStore
	DesiredPackagesChanged func()
}

// NewPackageService returns an app-side Munki package service.
func NewPackageService(deps PackageServiceDependencies) *PackageService {
	desiredPackagesChanged := deps.DesiredPackagesChanged
	if desiredPackagesChanged == nil {
		desiredPackagesChanged = func() {}
	}
	return &PackageService{
		store:                  deps.Packages,
		desiredPackagesChanged: desiredPackagesChanged,
	}
}

func (s *PackageService) List(
	ctx context.Context,
	params packages.PackageListParams,
) ([]packages.Package, int, error) {
	return s.store.List(ctx, params)
}

func (s *PackageService) Create(
	ctx context.Context,
	mutation packages.PackageCreateMutation,
) (*packages.Package, error) {
	pkg, err := s.store.Create(ctx, mutation)
	s.notifyDesiredPackages(err)
	return pkg, err
}

func (s *PackageService) GetByID(ctx context.Context, id int64) (*packages.Package, error) {
	return s.store.GetByID(ctx, id)
}

func (s *PackageService) Update(
	ctx context.Context,
	id int64,
	mutation packages.PackageMutation,
) (*packages.Package, error) {
	pkg, err := s.store.Update(ctx, id, mutation)
	s.notifyDesiredPackages(err)
	return pkg, err
}

func (s *PackageService) Delete(ctx context.Context, id int64) error {
	err := s.store.Delete(ctx, id)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	deleted, err := s.store.DeleteMany(ctx, ids)
	if err == nil && deleted > 0 {
		s.desiredPackagesChanged()
	}
	return deleted, err
}

func (s *PackageService) SetInstallerObject(ctx context.Context, packageID, objectID int64) error {
	err := s.store.SetInstallerObject(ctx, packageID, objectID)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) ClearInstallerObject(ctx context.Context, packageID int64) error {
	err := s.store.ClearInstallerObject(ctx, packageID)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) notifyDesiredPackages(err error) {
	if err == nil {
		s.desiredPackagesChanged()
	}
}
