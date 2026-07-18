package munki

import (
	"context"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	munkisoftware "github.com/woodleighschool/woodstar/internal/munki/software"
)

type packageStore interface {
	List(ctx context.Context, params packages.PackageListParams) ([]packages.Package, int, error)
	Create(ctx context.Context, mutation packages.PackageCreateMutation) (*packages.Package, error)
	GetByID(ctx context.Context, packageID int64) (*packages.Package, error)
	Update(ctx context.Context, packageID int64, mutation packages.PackageMutation) (*packages.Package, error)
	Delete(ctx context.Context, packageID int64) error
	DeleteMany(ctx context.Context, packageIDs []int64) (int, error)
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
	rows, count, err := s.deps.Packages.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		attachPackageSoftware(&rows[i])
	}
	return rows, count, nil
}

func (s *PackageService) Create(
	ctx context.Context,
	mutation packages.PackageCreateMutation,
) (*packages.Package, error) {
	pkg, err := s.deps.Packages.Create(ctx, mutation)
	s.notifyDesiredPackages(err)
	if pkg != nil {
		attachPackageSoftware(pkg)
	}
	return pkg, err
}

func (s *PackageService) GetByID(ctx context.Context, id int64) (*packages.Package, error) {
	pkg, err := s.deps.Packages.GetByID(ctx, id)
	if pkg != nil {
		attachPackageSoftware(pkg)
	}
	return pkg, err
}

func (s *PackageService) Update(
	ctx context.Context,
	id int64,
	mutation packages.PackageMutation,
) (*packages.Package, error) {
	pkg, err := s.deps.Packages.Update(ctx, id, mutation)
	s.notifyDesiredPackages(err)
	if pkg != nil {
		attachPackageSoftware(pkg)
	}
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

func (s *PackageService) notifyDesiredPackages(err error) {
	if err == nil {
		s.deps.DesiredPackagesChanged()
	}
}

func attachPackageSoftware(pkg *packages.Package) {
	pkg.Software.IconURL = munkisoftware.IconURL(pkg.Software.ID, pkg.Software.IconObjectID)
}
