package munki

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

func TestPackageServiceSignalsDesiredPackagesAfterSuccessfulMutations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &packageServiceTestStore{}
	var changes int
	service := NewPackageService(PackageServiceDependencies{
		Packages: store,
		DesiredPackagesChanged: func() {
			changes++
		},
	})

	if _, err := service.Create(ctx, packages.PackageCreateMutation{}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.Update(ctx, 1, packages.PackageMutation{}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := service.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.DeleteMany(ctx, []int64{1}); err != nil {
		t.Fatalf("DeleteMany() error = %v", err)
	}
	if err := service.SetInstallerObject(ctx, 1, 2); err != nil {
		t.Fatalf("SetInstallerObject() error = %v", err)
	}
	if err := service.ClearInstallerObject(ctx, 1); err != nil {
		t.Fatalf("ClearInstallerObject() error = %v", err)
	}

	if changes != 6 {
		t.Fatalf("desired package changes = %d, want 6", changes)
	}
}

func TestPackageServiceDoesNotSignalDesiredPackagesAfterFailedMutation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &packageServiceTestStore{err: errors.New("boom")}
	var changes int
	service := NewPackageService(PackageServiceDependencies{
		Packages: store,
		DesiredPackagesChanged: func() {
			changes++
		},
	})

	if _, err := service.Create(ctx, packages.PackageCreateMutation{}); err == nil {
		t.Fatal("Create() error = nil, want error")
	}
	if _, err := service.Update(ctx, 1, packages.PackageMutation{}); err == nil {
		t.Fatal("Update() error = nil, want error")
	}
	if err := service.Delete(ctx, 1); err == nil {
		t.Fatal("Delete() error = nil, want error")
	}
	if _, err := service.DeleteMany(ctx, []int64{1}); err == nil {
		t.Fatal("DeleteMany() error = nil, want error")
	}
	if err := service.SetInstallerObject(ctx, 1, 2); err == nil {
		t.Fatal("SetInstallerObject() error = nil, want error")
	}
	if err := service.ClearInstallerObject(ctx, 1); err == nil {
		t.Fatal("ClearInstallerObject() error = nil, want error")
	}

	if changes != 0 {
		t.Fatalf("desired package changes = %d, want 0", changes)
	}
}

type packageServiceTestStore struct {
	err error
}

func (s *packageServiceTestStore) List(
	context.Context,
	packages.PackageListParams,
) ([]packages.Package, int, error) {
	return nil, 0, s.err
}

func (s *packageServiceTestStore) Create(
	context.Context,
	packages.PackageCreateMutation,
) (*packages.Package, error) {
	return &packages.Package{}, s.err
}

func (s *packageServiceTestStore) GetByID(context.Context, int64) (*packages.Package, error) {
	return &packages.Package{}, s.err
}

func (s *packageServiceTestStore) Update(
	context.Context,
	int64,
	packages.PackageMutation,
) (*packages.Package, error) {
	return &packages.Package{}, s.err
}

func (s *packageServiceTestStore) Delete(context.Context, int64) error {
	return s.err
}

func (s *packageServiceTestStore) DeleteMany(context.Context, []int64) (int, error) {
	return 1, s.err
}

func (s *packageServiceTestStore) SetInstallerObject(context.Context, int64, int64) error {
	return s.err
}

func (s *packageServiceTestStore) ClearInstallerObject(context.Context, int64) error {
	return s.err
}
