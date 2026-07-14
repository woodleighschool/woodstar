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
	deps PackageServiceDependencies
}

// PackageServiceDependencies are the collaborators for PackageService.
type PackageServiceDependencies struct {
	Packages               packageStore
	DesiredPackagesChanged func()
}

// NewPackageService returns an app-side Munki package service.
func NewPackageService(deps PackageServiceDependencies) *PackageService {
	return &PackageService{deps: deps}
}

func (s *PackageService) List(
	ctx context.Context,
	params packages.PackageListParams,
) ([]packages.Package, int, error) {
	return s.deps.Packages.List(ctx, params)
}

func (s *PackageService) Create(
	ctx context.Context,
	mutation packages.PackageCreateMutation,
) (*packages.Package, error) {
	pkg, err := s.deps.Packages.Create(ctx, mutation)
	s.notifyDesiredPackages(err)
	return pkg, err
}

func (s *PackageService) GetByID(ctx context.Context, id int64) (*packages.Package, error) {
	return s.deps.Packages.GetByID(ctx, id)
}

func (s *PackageService) Update(
	ctx context.Context,
	id int64,
	mutation packages.PackageMutation,
) (*packages.Package, error) {
	pkg, err := s.deps.Packages.Update(ctx, id, mutation)
	s.notifyDesiredPackages(err)
	return pkg, err
}

func (s *PackageService) Delete(ctx context.Context, id int64) error {
	err := s.deps.Packages.Delete(ctx, id)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	deleted, err := s.deps.Packages.DeleteMany(ctx, ids)
	if err == nil && deleted > 0 {
		s.deps.DesiredPackagesChanged()
	}
	return deleted, err
}

func (s *PackageService) SetInstallerObject(ctx context.Context, packageID, objectID int64) error {
	err := s.deps.Packages.SetInstallerObject(ctx, packageID, objectID)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) ClearInstallerObject(ctx context.Context, packageID int64) error {
	err := s.deps.Packages.ClearInstallerObject(ctx, packageID)
	s.notifyDesiredPackages(err)
	return err
}

func (s *PackageService) notifyDesiredPackages(err error) {
	if err == nil {
		s.deps.DesiredPackagesChanged()
	}
}
