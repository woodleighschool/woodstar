package packages

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

// ObjectPrefix namespaces package installer objects in storage.
const ObjectPrefix = "munki/packages"

const (
	relationKindRequires  = "requires"
	relationKindUpdateFor = "update_for"
)

type objectStore interface {
	GetByID(context.Context, int64) (*storage.Object, error)
	DeleteUnreferenced(context.Context, ...int64) error
}

type Store struct {
	db      *database.DB
	objects objectStore
}

func NewStore(db *database.DB, objects objectStore) *Store {
	return &Store{
		db:      db,
		objects: objects,
	}
}

func (s *Store) Create(ctx context.Context, in PackageCreateMutation) (*Package, error) {
	if in.SoftwareID <= 0 {
		return nil, fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	params, err := s.prepareMutation(ctx, in.PackageMutation)
	if err != nil {
		return nil, err
	}
	params = carryExistingObjects(params, Package{})
	write := newPackageWrite(in.SoftwareID, params)

	var id int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, insertPackageSQL, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		return writePackageRelations(ctx, tx, id, params)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, params PackageMutation) (*Package, error) {
	params, err := s.prepareMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	params = carryExistingObjects(params, *existing)
	write := newPackageWrite(existing.SoftwareID, params)
	write.ID = id

	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updatedID int64
		if err := tx.QueryRow(ctx, updatePackageSQL, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return writePackageRelations(ctx, tx, id, params)
	})
	if err != nil {
		return nil, err
	}
	replacedIDs := storage.ReplacedObjectIDs(existing.InstallerObjectID, params.InstallerObjectID)
	if err := s.objects.DeleteUnreferenced(ctx, replacedIDs...); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Package, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[packageRow](ctx, s.db.Pool(), packageSelectSQL+"\nWHERE p.id = $1", id)
	if err != nil {
		return nil, err
	}
	packages, err := s.AttachRelations(ctx, []Package{packageFromRow(row)})
	if err != nil {
		return nil, err
	}
	return &packages[0], nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	objectIDs, err := s.packageObjectIDs(ctx, s.db.Pool(), []int64{id})
	if err != nil {
		return err
	}
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM munki_packages WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "Munki package is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return s.objects.DeleteUnreferenced(ctx, objectIDs...)
}

// DeleteMany removes multiple package rows. Missing IDs are ignored for bulk idempotency.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var deleted int
	var objectIDs []int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		objectIDs, err = s.packageObjectIDs(ctx, tx, ids)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			ctx,
			`DELETE FROM munki_package_relations WHERE package_id = ANY($1::bigint[])`,
			ids,
		); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `DELETE FROM munki_packages WHERE id = ANY($1::bigint[]) RETURNING id`, ids)
		if err != nil {
			return dbutil.DeleteConflict(err, "Munki package is still referenced")
		}
		deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return dbutil.DeleteConflict(err, "Munki package is still referenced")
		}
		deleted = len(deletedIDs)
		return nil
	})
	if err != nil {
		return deleted, err
	}
	return deleted, s.objects.DeleteUnreferenced(ctx, objectIDs...)
}

func (s *Store) List(ctx context.Context, params PackageListParams) ([]Package, int, error) {
	installerTypes, err := normalizeInstallerTypeFilters(params.InstallerTypes)
	if err != nil {
		return nil, 0, err
	}
	params.InstallerTypes = installerTypes
	where, args := packageListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL: packageSelectSQL,
		WhereSQL:  where,
		Args:      args,
		OrderKeys: packageOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{
			{SQL: "lower(s.name)"},
			{SQL: "lower(p.version)"},
			{SQL: "p.id"},
		},
		Params: params.ListParams,
	}
	rows, count, err := dbutil.ListWithCount[packageRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}
	packages, err := s.AttachRelations(ctx, packagesFromRows(rows))
	if err != nil {
		return nil, 0, err
	}
	return packages, count, nil
}

// ListRepositoryPackages returns every package that may appear in the shared
// Munki catalog.
func (s *Store) ListRepositoryPackages(ctx context.Context) ([]Package, error) {
	rows, err := s.db.Pool().Query(ctx, packageSelectSQL+`
WHERE p.eligible
  AND (p.installer_type = 'nopkg' OR installer_obj.available_at IS NOT NULL)
ORDER BY lower(s.name), s.id, p.id`)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRow])
	if err != nil {
		return nil, err
	}
	return s.AttachRelations(ctx, packagesFromRows(records))
}

func (s *Store) prepareMutation(ctx context.Context, params PackageMutation) (PackageMutation, error) {
	params = applyDefaults(params)
	if err := params.Validate(); err != nil {
		return PackageMutation{}, err
	}
	return s.validatePackageObjects(ctx, params)
}

func (s *Store) validatePackageObjects(ctx context.Context, params PackageMutation) (PackageMutation, error) {
	if params.InstallerObjectID != nil {
		if err := s.requirePackageObject(ctx, *params.InstallerObjectID, "installer_object_id"); err != nil {
			return params, err
		}
	}
	return params, nil
}

func carryExistingObjects(params PackageMutation, existing Package) PackageMutation {
	if params.InstallerType == InstallerTypeNoPkg {
		params.InstallerObjectID = nil
	} else if params.InstallerObjectID == nil {
		params.InstallerObjectID = existing.InstallerObjectID
	}
	return params
}

func (s *Store) requirePackageObject(ctx context.Context, id int64, field string) error {
	obj, err := s.objects.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if obj.Prefix != ObjectPrefix {
		return fmt.Errorf("%w: %s must reference a package installer", dbutil.ErrInvalidInput, field)
	}
	if !obj.Available() {
		return fmt.Errorf("%w: %s must reference an uploaded object", dbutil.ErrInvalidInput, field)
	}
	return nil
}

// SetInstallerObject points a package at an installer storage object.
func (s *Store) SetInstallerObject(ctx context.Context, packageID, objectID int64) error {
	if err := s.requirePackageObject(ctx, objectID, "installer_object_id"); err != nil {
		return err
	}
	var oldObjectID *int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		oldObjectID, err = installerObjectID(ctx, tx, packageID)
		if err != nil {
			return err
		}
		return setInstallerObjectID(ctx, tx, packageID, &objectID)
	})
	if err != nil {
		return err
	}
	return s.objects.DeleteUnreferenced(ctx, storage.ReplacedObjectIDs(oldObjectID, &objectID)...)
}

// ClearInstallerObject detaches a package installer object and deletes its bytes
// when no other Munki resource references it.
func (s *Store) ClearInstallerObject(ctx context.Context, packageID int64) error {
	var oldObjectID *int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		oldObjectID, err = installerObjectID(ctx, tx, packageID)
		if err != nil {
			return err
		}
		if oldObjectID == nil {
			return nil
		}
		return setInstallerObjectID(ctx, tx, packageID, nil)
	})
	if err != nil {
		return err
	}
	return s.objects.DeleteUnreferenced(ctx, storage.ReplacedObjectIDs(oldObjectID, nil)...)
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (s *Store) packageObjectIDs(ctx context.Context, q rowQuerier, ids []int64) ([]int64, error) {
	rows, err := q.Query(ctx, `
		SELECT refs.object_id::bigint AS object_id
		FROM munki_packages p
		CROSS JOIN LATERAL unnest(array_remove(ARRAY[p.installer_object_id], NULL)::bigint[]) AS refs(object_id)
		WHERE p.id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func installerObjectID(ctx context.Context, tx pgx.Tx, packageID int64) (*int64, error) {
	var objectID *int64
	err := tx.QueryRow(ctx, `SELECT installer_object_id FROM munki_packages WHERE id = $1`, packageID).Scan(&objectID)
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	return objectID, nil
}

func setInstallerObjectID(ctx context.Context, tx pgx.Tx, packageID int64, objectID *int64) error {
	tag, err := tx.Exec(
		ctx,
		`UPDATE munki_packages SET installer_object_id = $2, updated_at = now() WHERE id = $1`,
		packageID,
		objectID,
	)
	if err != nil {
		return dbutil.MutationError(err)
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

func applyDefaults(params PackageMutation) PackageMutation {
	if params.InstallerType == "" {
		params.InstallerType = InstallerTypePkg
	}
	// supported_architectures is NOT NULL; nil means no architecture restriction.
	if params.SupportedArchitectures == nil {
		params.SupportedArchitectures = []string{}
	}
	return params
}

func packagesFromRows(rows []packageRow) []Package {
	packages := make([]Package, len(rows))
	for i, row := range rows {
		packages[i] = packageFromRow(row)
	}
	return packages
}

func packageFromRow(row packageRow) Package {
	return Package{
		ID:                       row.ID,
		SoftwareID:               row.SoftwareID,
		SoftwareName:             row.SoftwareName,
		SoftwareDescription:      row.SoftwareDescription,
		SoftwareCategory:         row.SoftwareCategory,
		SoftwareDeveloper:        row.SoftwareDeveloper,
		Version:                  row.Version,
		InstallerType:            InstallerType(row.InstallerType),
		UnattendedInstall:        row.UnattendedInstall,
		UnattendedUninstall:      row.UnattendedUninstall,
		UninstallMethod:          UninstallMethod(row.UninstallMethod),
		RestartAction:            RestartAction(row.RestartAction),
		MinimumMunkiVersion:      row.MinimumMunkiVersion,
		MinimumOSVersion:         row.MinimumOSVersion,
		MaximumOSVersion:         row.MaximumOSVersion,
		SupportedArchitectures:   dbutil.NonNilSlice(row.SupportedArchitectures),
		BlockingApplications:     row.BlockingApplications,
		InstallableCondition:     row.InstallableCondition,
		BlockingAppsManualQuit:   row.BlockingAppsManualQuit,
		BlockingAppsQuitScript:   row.BlockingAppsQuitScript,
		Requires:                 []PackageReference{},
		UpdateFor:                []PackageReference{},
		OnDemand:                 row.OnDemand,
		Precache:                 row.Precache,
		Autoremove:               row.Autoremove,
		AppleItem:                row.AppleItem,
		SuppressBundleRelocation: row.SuppressBundleRelocation,
		ForceInstallAfterDate:    row.ForceInstallAfterDate,
		InstalledSize:            row.InstalledSize,
		InstallerFile:            row.InstallerFile(),
		PackagePath:              row.PackagePath,
		InstallerChoicesXML:      []PackageInstallerChoice(row.InstallerChoicesXML),
		InstallerEnvironment:     []PackageInstallerEnvironmentVariable(row.InstallerEnvironment),
		Installs:                 []PackageInstallItem(row.Installs),
		Receipts:                 []PackageReceipt(row.Receipts),
		ItemsToCopy:              []PackageItemToCopy(row.ItemsToCopy),
		Notes:                    row.Notes,
		InstallcheckScript:       row.InstallcheckScript,
		UninstallcheckScript:     row.UninstallcheckScript,
		PreinstallScript:         row.PreinstallScript,
		PostinstallScript:        row.PostinstallScript,
		PreuninstallScript:       row.PreuninstallScript,
		PostuninstallScript:      row.PostuninstallScript,
		UninstallScript:          row.UninstallScript,
		VersionScript:            row.VersionScript,
		PreinstallAlert:          row.PreinstallAlert(),
		PreuninstallAlert:        row.PreuninstallAlert(),
		InstallerObjectID:        row.InstallerObjectID,
		SoftwareIconObjectID:     row.SoftwareIconObjectID,
		Eligible:                 row.Eligible,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
}

// FromEffectiveRow maps the host-effective package query row into a package read model.
//
// TODO(store-rewrite): convert with the munki host-effective slice.
func FromEffectiveRow(row sqlc.ListEffectiveMunkiPackagesForHostRow) (Package, error) {
	return packageFromRow(packageRowFromEffectiveSQLC(row)), nil
}

func packageListWhere(params PackageListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("p.software_id = " + where.Arg(params.SoftwareID))
	}
	if len(params.InstallerTypes) > 0 {
		where.Add("p.installer_type = ANY(" + where.Arg(params.InstallerTypes) + "::text[])")
	}
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			p.version ILIKE ` + search + `
			OR p.installer_type ILIKE ` + search + `
			OR s.name ILIKE ` + search + `
			OR s.description ILIKE ` + search + `
			OR s.category ILIKE ` + search + `
			OR s.developer ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func normalizeInstallerTypeFilters(raw []string) ([]string, error) {
	values := dbutil.SplitListValues(raw)
	for _, value := range values {
		if !validInstallerType(InstallerType(value)) {
			return nil, fmt.Errorf("%w: unsupported type %q", dbutil.ErrInvalidInput, value)
		}
	}
	return values, nil
}

func packageOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":       {SQL: "lower(s.name)"},
		"package":    {SQL: "lower(s.name)"},
		"version":    {SQL: "lower(p.version)"},
		"type":       {SQL: "lower(p.installer_type)"},
		"size":       {SQL: "COALESCE(installer_obj.size_bytes, 0)"},
		"software":   {SQL: "lower(s.name)"},
		"updated_at": {SQL: "p.updated_at"},
	}
}

type packageRow struct {
	ID                           int64
	SoftwareID                   int64
	SoftwareName                 string
	SoftwareDescription          string
	SoftwareCategory             string
	SoftwareDeveloper            string
	Version                      string
	InstallerType                string
	UninstallMethod              string
	RestartAction                string
	MinimumMunkiVersion          string
	MinimumOSVersion             string
	MaximumOSVersion             string
	SupportedArchitectures       []string
	BlockingApplications         []string
	InstallableCondition         string
	BlockingAppsManualQuit       bool   `db:"blocking_applications_manual_quit_only"`
	BlockingAppsQuitScript       string `db:"blocking_applications_quit_script"`
	UnattendedInstall            bool
	UnattendedUninstall          bool
	OnDemand                     bool
	Precache                     bool
	Autoremove                   bool
	AppleItem                    bool
	SuppressBundleRelocation     bool
	ForceInstallAfterDate        *time.Time
	InstalledSize                int64
	InstallerFilename            *string
	InstallerSizeBytes           *int64
	InstallerSHA256              *string `db:"installer_sha256"`
	PackagePath                  string
	InstallerChoicesXML          packageInstallerChoices `db:"installer_choices_xml"`
	InstallerEnvironment         packageInstallerEnvironment
	Installs                     packageInstallItems
	Receipts                     packageReceipts
	ItemsToCopy                  packageItemsToCopy
	Notes                        string
	InstallcheckScript           string
	UninstallcheckScript         string
	PreinstallScript             string
	PostinstallScript            string
	PreuninstallScript           string
	PostuninstallScript          string
	UninstallScript              string
	VersionScript                string
	PreinstallAlertEnabled       bool
	PreinstallAlertTitle         string
	PreinstallAlertDetail        string
	PreinstallAlertOKLabel       string `db:"preinstall_alert_ok_label"`
	PreinstallAlertCancelLabel   string
	PreuninstallAlertEnabled     bool
	PreuninstallAlertTitle       string
	PreuninstallAlertDetail      string
	PreuninstallAlertOKLabel     string `db:"preuninstall_alert_ok_label"`
	PreuninstallAlertCancelLabel string
	InstallerObjectID            *int64
	SoftwareIconObjectID         *int64 `db:"software_icon_object_id"`
	Eligible                     bool
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

func (row packageRow) PreinstallAlert() PackageAlert {
	return PackageAlert{
		Enabled:     row.PreinstallAlertEnabled,
		Title:       row.PreinstallAlertTitle,
		Detail:      row.PreinstallAlertDetail,
		OKLabel:     row.PreinstallAlertOKLabel,
		CancelLabel: row.PreinstallAlertCancelLabel,
	}
}

func (row packageRow) PreuninstallAlert() PackageAlert {
	return PackageAlert{
		Enabled:     row.PreuninstallAlertEnabled,
		Title:       row.PreuninstallAlertTitle,
		Detail:      row.PreuninstallAlertDetail,
		OKLabel:     row.PreuninstallAlertOKLabel,
		CancelLabel: row.PreuninstallAlertCancelLabel,
	}
}

func (row packageRow) InstallerFile() *InstallerFile {
	if row.InstallerObjectID == nil || row.InstallerFilename == nil {
		return nil
	}
	pkg := Package{ID: row.ID}
	obj := storage.Object{
		ID:        *row.InstallerObjectID,
		Filename:  *row.InstallerFilename,
		SizeBytes: row.InstallerSizeBytes,
		SHA256:    row.InstallerSHA256,
	}
	var size int64
	if row.InstallerSizeBytes != nil {
		size = *row.InstallerSizeBytes
	}
	var sha string
	if row.InstallerSHA256 != nil {
		sha = *row.InstallerSHA256
	}
	return &InstallerFile{
		Filename:              *row.InstallerFilename,
		InstallerItemLocation: InstallerItemLocation(pkg, obj),
		SizeBytes:             size,
		SHA256:                sha,
	}
}

func packageRowFromEffectiveSQLC(row sqlc.ListEffectiveMunkiPackagesForHostRow) packageRow {
	out := packageRow{
		ID:                           row.PackageID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDescription:          row.SoftwareDescription,
		SoftwareCategory:             row.SoftwareCategory,
		SoftwareDeveloper:            row.SoftwareDeveloper,
		SoftwareIconObjectID:         row.SoftwareIconObjectID,
		Version:                      row.Version,
		InstallerType:                row.InstallerType,
		UninstallMethod:              row.UninstallMethod,
		RestartAction:                row.RestartAction,
		MinimumMunkiVersion:          row.MinimumMunkiVersion,
		MinimumOSVersion:             row.MinimumOSVersion,
		MaximumOSVersion:             row.MaximumOSVersion,
		SupportedArchitectures:       row.SupportedArchitectures,
		BlockingApplications:         row.BlockingApplications,
		InstallableCondition:         row.InstallableCondition,
		BlockingAppsManualQuit:       row.BlockingApplicationsManualQuitOnly,
		BlockingAppsQuitScript:       row.BlockingApplicationsQuitScript,
		UnattendedInstall:            row.UnattendedInstall,
		UnattendedUninstall:          row.UnattendedUninstall,
		OnDemand:                     row.OnDemand,
		Precache:                     row.Precache,
		Autoremove:                   row.Autoremove,
		AppleItem:                    row.AppleItem,
		SuppressBundleRelocation:     row.SuppressBundleRelocation,
		ForceInstallAfterDate:        row.ForceInstallAfterDate,
		InstalledSize:                row.InstalledSize,
		PackagePath:                  row.PackagePath,
		Notes:                        row.Notes,
		InstallcheckScript:           row.InstallcheckScript,
		UninstallcheckScript:         row.UninstallcheckScript,
		PreinstallScript:             row.PreinstallScript,
		PostinstallScript:            row.PostinstallScript,
		PreuninstallScript:           row.PreuninstallScript,
		PostuninstallScript:          row.PostuninstallScript,
		UninstallScript:              row.UninstallScript,
		VersionScript:                row.VersionScript,
		PreinstallAlertEnabled:       row.PreinstallAlertEnabled,
		PreinstallAlertTitle:         row.PreinstallAlertTitle,
		PreinstallAlertDetail:        row.PreinstallAlertDetail,
		PreinstallAlertOKLabel:       row.PreinstallAlertOkLabel,
		PreinstallAlertCancelLabel:   row.PreinstallAlertCancelLabel,
		PreuninstallAlertEnabled:     row.PreuninstallAlertEnabled,
		PreuninstallAlertTitle:       row.PreuninstallAlertTitle,
		PreuninstallAlertDetail:      row.PreuninstallAlertDetail,
		PreuninstallAlertOKLabel:     row.PreuninstallAlertOkLabel,
		PreuninstallAlertCancelLabel: row.PreuninstallAlertCancelLabel,
		InstallerObjectID:            row.InstallerObjectID,
		Eligible:                     true,
	}
	_ = out.InstallerChoicesXML.Scan(row.InstallerChoicesXml)
	_ = out.InstallerEnvironment.Scan(row.InstallerEnvironment)
	_ = out.Installs.Scan(row.Installs)
	_ = out.Receipts.Scan(row.Receipts)
	_ = out.ItemsToCopy.Scan(row.ItemsToCopy)
	return out
}

type packageWrite struct {
	ID                           int64                       `db:"id"`
	SoftwareID                   int64                       `db:"software_id"`
	Version                      string                      `db:"version"`
	InstallerType                string                      `db:"installer_type"`
	UninstallMethod              string                      `db:"uninstall_method"`
	RestartAction                string                      `db:"restart_action"`
	MinimumMunkiVersion          string                      `db:"minimum_munki_version"`
	MinimumOSVersion             string                      `db:"minimum_os_version"`
	MaximumOSVersion             string                      `db:"maximum_os_version"`
	SupportedArchitectures       []string                    `db:"supported_architectures"`
	BlockingApplications         []string                    `db:"blocking_applications"`
	InstallableCondition         string                      `db:"installable_condition"`
	BlockingAppsManualQuit       bool                        `db:"blocking_applications_manual_quit_only"`
	BlockingAppsQuitScript       string                      `db:"blocking_applications_quit_script"`
	UnattendedInstall            bool                        `db:"unattended_install"`
	UnattendedUninstall          bool                        `db:"unattended_uninstall"`
	OnDemand                     bool                        `db:"on_demand"`
	Precache                     bool                        `db:"precache"`
	Autoremove                   bool                        `db:"autoremove"`
	AppleItem                    bool                        `db:"apple_item"`
	SuppressBundleRelocation     bool                        `db:"suppress_bundle_relocation"`
	ForceInstallAfterDate        *time.Time                  `db:"force_install_after_date"`
	InstalledSize                int64                       `db:"installed_size"`
	PackagePath                  string                      `db:"package_path"`
	InstallerChoicesXML          packageInstallerChoices     `db:"installer_choices_xml"`
	InstallerEnvironment         packageInstallerEnvironment `db:"installer_environment"`
	Installs                     packageInstallItems         `db:"installs"`
	Receipts                     packageReceipts             `db:"receipts"`
	ItemsToCopy                  packageItemsToCopy          `db:"items_to_copy"`
	Notes                        string                      `db:"notes"`
	InstallcheckScript           string                      `db:"installcheck_script"`
	UninstallcheckScript         string                      `db:"uninstallcheck_script"`
	PreinstallScript             string                      `db:"preinstall_script"`
	PostinstallScript            string                      `db:"postinstall_script"`
	PreuninstallScript           string                      `db:"preuninstall_script"`
	PostuninstallScript          string                      `db:"postuninstall_script"`
	UninstallScript              string                      `db:"uninstall_script"`
	VersionScript                string                      `db:"version_script"`
	PreinstallAlertEnabled       bool                        `db:"preinstall_alert_enabled"`
	PreinstallAlertTitle         string                      `db:"preinstall_alert_title"`
	PreinstallAlertDetail        string                      `db:"preinstall_alert_detail"`
	PreinstallAlertOKLabel       string                      `db:"preinstall_alert_ok_label"`
	PreinstallAlertCancelLabel   string                      `db:"preinstall_alert_cancel_label"`
	PreuninstallAlertEnabled     bool                        `db:"preuninstall_alert_enabled"`
	PreuninstallAlertTitle       string                      `db:"preuninstall_alert_title"`
	PreuninstallAlertDetail      string                      `db:"preuninstall_alert_detail"`
	PreuninstallAlertOKLabel     string                      `db:"preuninstall_alert_ok_label"`
	PreuninstallAlertCancelLabel string                      `db:"preuninstall_alert_cancel_label"`
	InstallerObjectID            *int64                      `db:"installer_object_id"`
	Eligible                     bool                        `db:"eligible"`
}

func newPackageWrite(softwareID int64, params PackageMutation) packageWrite {
	return packageWrite{
		SoftwareID:                   softwareID,
		Version:                      params.Version,
		InstallerType:                string(params.InstallerType),
		UninstallMethod:              string(params.UninstallMethod),
		RestartAction:                string(params.RestartAction),
		MinimumMunkiVersion:          params.MinimumMunkiVersion,
		MinimumOSVersion:             params.MinimumOSVersion,
		MaximumOSVersion:             params.MaximumOSVersion,
		SupportedArchitectures:       params.SupportedArchitectures,
		BlockingApplications:         params.BlockingApplications,
		InstallableCondition:         params.InstallableCondition,
		BlockingAppsManualQuit:       params.BlockingAppsManualQuit,
		BlockingAppsQuitScript:       params.BlockingAppsQuitScript,
		UnattendedInstall:            params.UnattendedInstall,
		UnattendedUninstall:          params.UnattendedUninstall,
		OnDemand:                     params.OnDemand,
		Precache:                     params.Precache,
		Autoremove:                   params.Autoremove,
		AppleItem:                    params.AppleItem,
		SuppressBundleRelocation:     params.SuppressBundleRelocation,
		ForceInstallAfterDate:        params.ForceInstallAfterDate,
		InstalledSize:                params.InstalledSize,
		PackagePath:                  params.PackagePath,
		InstallerChoicesXML:          packageInstallerChoices(params.InstallerChoicesXML),
		InstallerEnvironment:         packageInstallerEnvironment(params.InstallerEnvironment),
		Installs:                     packageInstallItems(params.Installs),
		Receipts:                     packageReceipts(params.Receipts),
		ItemsToCopy:                  packageItemsToCopy(params.ItemsToCopy),
		Notes:                        params.Notes,
		InstallcheckScript:           params.InstallcheckScript,
		UninstallcheckScript:         params.UninstallcheckScript,
		PreinstallScript:             params.PreinstallScript,
		PostinstallScript:            params.PostinstallScript,
		PreuninstallScript:           params.PreuninstallScript,
		PostuninstallScript:          params.PostuninstallScript,
		UninstallScript:              params.UninstallScript,
		VersionScript:                params.VersionScript,
		PreinstallAlertEnabled:       params.PreinstallAlert.Enabled,
		PreinstallAlertTitle:         params.PreinstallAlert.Title,
		PreinstallAlertDetail:        params.PreinstallAlert.Detail,
		PreinstallAlertOKLabel:       params.PreinstallAlert.OKLabel,
		PreinstallAlertCancelLabel:   params.PreinstallAlert.CancelLabel,
		PreuninstallAlertEnabled:     params.PreuninstallAlert.Enabled,
		PreuninstallAlertTitle:       params.PreuninstallAlert.Title,
		PreuninstallAlertDetail:      params.PreuninstallAlert.Detail,
		PreuninstallAlertOKLabel:     params.PreuninstallAlert.OKLabel,
		PreuninstallAlertCancelLabel: params.PreuninstallAlert.CancelLabel,
		InstallerObjectID:            params.InstallerObjectID,
		Eligible:                     params.Eligible,
	}
}

const insertPackageSQL = `
INSERT INTO munki_packages (
	software_id,
	version,
	installer_type,
	uninstall_method,
	restart_action,
	minimum_munki_version,
	minimum_os_version,
	maximum_os_version,
	supported_architectures,
	blocking_applications,
	installable_condition,
	blocking_applications_manual_quit_only,
	blocking_applications_quit_script,
	unattended_install,
	unattended_uninstall,
	on_demand,
	precache,
	autoremove,
	apple_item,
	suppress_bundle_relocation,
	force_install_after_date,
	installed_size,
	package_path,
	installer_choices_xml,
	installer_environment,
	installs,
	receipts,
	items_to_copy,
	notes,
	installcheck_script,
	uninstallcheck_script,
	preinstall_script,
	postinstall_script,
	preuninstall_script,
	postuninstall_script,
	uninstall_script,
	version_script,
	preinstall_alert_enabled,
	preinstall_alert_title,
	preinstall_alert_detail,
	preinstall_alert_ok_label,
	preinstall_alert_cancel_label,
	preuninstall_alert_enabled,
	preuninstall_alert_title,
	preuninstall_alert_detail,
	preuninstall_alert_ok_label,
	preuninstall_alert_cancel_label,
	installer_object_id,
	eligible
)
VALUES (
	@software_id,
	@version,
	@installer_type,
	@uninstall_method,
	@restart_action,
	@minimum_munki_version,
	@minimum_os_version,
	@maximum_os_version,
	@supported_architectures::text[],
	@blocking_applications::text[],
	@installable_condition,
	@blocking_applications_manual_quit_only,
	@blocking_applications_quit_script,
	@unattended_install,
	@unattended_uninstall,
	@on_demand,
	@precache,
	@autoremove,
	@apple_item,
	@suppress_bundle_relocation,
	@force_install_after_date::timestamptz,
	@installed_size,
	@package_path,
	@installer_choices_xml::jsonb,
	@installer_environment::jsonb,
	@installs::jsonb,
	@receipts::jsonb,
	@items_to_copy::jsonb,
	@notes,
	@installcheck_script,
	@uninstallcheck_script,
	@preinstall_script,
	@postinstall_script,
	@preuninstall_script,
	@postuninstall_script,
	@uninstall_script,
	@version_script,
	@preinstall_alert_enabled,
	@preinstall_alert_title,
	@preinstall_alert_detail,
	@preinstall_alert_ok_label,
	@preinstall_alert_cancel_label,
	@preuninstall_alert_enabled,
	@preuninstall_alert_title,
	@preuninstall_alert_detail,
	@preuninstall_alert_ok_label,
	@preuninstall_alert_cancel_label,
	@installer_object_id::bigint,
	@eligible
)
RETURNING id`

const updatePackageSQL = `
UPDATE munki_packages
SET
	version = @version,
	installer_type = @installer_type,
	uninstall_method = @uninstall_method,
	restart_action = @restart_action,
	minimum_munki_version = @minimum_munki_version,
	minimum_os_version = @minimum_os_version,
	maximum_os_version = @maximum_os_version,
	supported_architectures = @supported_architectures::text[],
	blocking_applications = @blocking_applications::text[],
	installable_condition = @installable_condition,
	blocking_applications_manual_quit_only = @blocking_applications_manual_quit_only,
	blocking_applications_quit_script = @blocking_applications_quit_script,
	unattended_install = @unattended_install,
	unattended_uninstall = @unattended_uninstall,
	on_demand = @on_demand,
	precache = @precache,
	autoremove = @autoremove,
	apple_item = @apple_item,
	suppress_bundle_relocation = @suppress_bundle_relocation,
	force_install_after_date = @force_install_after_date::timestamptz,
	installed_size = @installed_size,
	package_path = @package_path,
	installer_choices_xml = @installer_choices_xml::jsonb,
	installer_environment = @installer_environment::jsonb,
	installs = @installs::jsonb,
	receipts = @receipts::jsonb,
	items_to_copy = @items_to_copy::jsonb,
	notes = @notes,
	installcheck_script = @installcheck_script,
	uninstallcheck_script = @uninstallcheck_script,
	preinstall_script = @preinstall_script,
	postinstall_script = @postinstall_script,
	preuninstall_script = @preuninstall_script,
	postuninstall_script = @postuninstall_script,
	uninstall_script = @uninstall_script,
	version_script = @version_script,
	preinstall_alert_enabled = @preinstall_alert_enabled,
	preinstall_alert_title = @preinstall_alert_title,
	preinstall_alert_detail = @preinstall_alert_detail,
	preinstall_alert_ok_label = @preinstall_alert_ok_label,
	preinstall_alert_cancel_label = @preinstall_alert_cancel_label,
	preuninstall_alert_enabled = @preuninstall_alert_enabled,
	preuninstall_alert_title = @preuninstall_alert_title,
	preuninstall_alert_detail = @preuninstall_alert_detail,
	preuninstall_alert_ok_label = @preuninstall_alert_ok_label,
	preuninstall_alert_cancel_label = @preuninstall_alert_cancel_label,
	installer_object_id = @installer_object_id::bigint,
	eligible = @eligible,
	updated_at = now()
WHERE id = @id
RETURNING id`

const packageSelectSQL = `
SELECT
	p.id,
	p.software_id,
	s.name AS software_name,
	s.description AS software_description,
	s.category AS software_category,
	s.developer AS software_developer,
	s.icon_object_id AS software_icon_object_id,
	p.version,
	p.installer_type,
	p.uninstall_method,
	p.restart_action,
	p.minimum_munki_version,
	p.minimum_os_version,
	p.maximum_os_version,
	p.supported_architectures,
	p.blocking_applications,
	p.installable_condition,
	p.blocking_applications_manual_quit_only,
	p.blocking_applications_quit_script,
	p.unattended_install,
	p.unattended_uninstall,
	p.on_demand,
	p.precache,
	p.autoremove,
	p.apple_item,
	p.suppress_bundle_relocation,
	p.force_install_after_date,
	p.installed_size,
	installer_obj.filename AS installer_filename,
	installer_obj.size_bytes AS installer_size_bytes,
	installer_obj.sha256 AS installer_sha256,
	p.package_path,
	p.installer_choices_xml,
	p.installer_environment,
	p.installs,
	p.receipts,
	p.items_to_copy,
	p.notes,
	p.installcheck_script,
	p.uninstallcheck_script,
	p.preinstall_script,
	p.postinstall_script,
	p.preuninstall_script,
	p.postuninstall_script,
	p.uninstall_script,
	p.version_script,
	p.preinstall_alert_enabled,
	p.preinstall_alert_title,
	p.preinstall_alert_detail,
	p.preinstall_alert_ok_label,
	p.preinstall_alert_cancel_label,
	p.preuninstall_alert_enabled,
	p.preuninstall_alert_title,
	p.preuninstall_alert_detail,
	p.preuninstall_alert_ok_label,
	p.preuninstall_alert_cancel_label,
	p.installer_object_id,
	p.eligible,
	p.created_at,
	p.updated_at
FROM munki_packages p
JOIN munki_software s ON s.id = p.software_id
LEFT JOIN storage_objects installer_obj ON installer_obj.id = p.installer_object_id`

func writePackageRelations(ctx context.Context, tx pgx.Tx, packageID int64, params PackageMutation) error {
	if err := replacePackageRelations(ctx, tx, packageID, relationKindRequires, params.Requires); err != nil {
		return dbutil.MutationError(err)
	}
	if err := replacePackageRelations(ctx, tx, packageID, relationKindUpdateFor, params.UpdateFor); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

func replacePackageRelations(
	ctx context.Context,
	tx pgx.Tx,
	packageID int64,
	kind string,
	references []PackageReference,
) error {
	if _, err := tx.Exec(
		ctx,
		`DELETE FROM munki_package_relations
		WHERE package_id = $1 AND relation_kind = $2::munki_package_relation_kind`,
		packageID,
		kind,
	); err != nil {
		return err
	}
	for position, ref := range references {
		if _, err := tx.Exec(ctx, `
			INSERT INTO munki_package_relations (
				package_id,
				relation_kind,
				target_software_id,
				target_package_id,
				position
			) VALUES ($1, $2::munki_package_relation_kind, $3, $4, $5)`,
			packageID,
			kind,
			ref.SoftwareID,
			optionalPositiveInt64(ref.PackageID),
			int32(position),
		); err != nil {
			return err
		}
	}
	return nil
}

type packageRelationRow struct {
	PackageID       int64
	RelationKind    string
	SoftwareID      int64
	SoftwareName    string
	TargetPackageID *int64
	TargetVersion   string
}

type packageRelations struct {
	requires  []PackageReference
	updateFor []PackageReference
}

// AttachRelations loads requires and update_for references for package rows.
func (s *Store) AttachRelations(ctx context.Context, packages []Package) ([]Package, error) {
	relations, err := s.packageRelationsByPackage(ctx, packageIDs(packages))
	if err != nil {
		return nil, err
	}
	for i := range packages {
		rel := relations[packages[i].ID]
		packages[i].Requires = nonNilReferences(rel.requires)
		packages[i].UpdateFor = nonNilReferences(rel.updateFor)
	}
	return packages, nil
}

func (s *Store) packageRelationsByPackage(
	ctx context.Context,
	packageIDs []int64,
) (map[int64]packageRelations, error) {
	if len(packageIDs) == 0 {
		return map[int64]packageRelations{}, nil
	}
	rows, err := s.db.Pool().Query(ctx, `
		SELECT
			r.package_id,
			r.relation_kind,
			r.target_software_id AS software_id,
			target_software.name AS software_name,
			r.target_package_id,
			COALESCE(target.version, '') AS target_version
		FROM munki_package_relations r
		JOIN munki_software target_software ON target_software.id = r.target_software_id
		LEFT JOIN munki_packages target ON target.id = r.target_package_id
		WHERE r.package_id = ANY($1::bigint[])
		ORDER BY r.package_id, r.relation_kind, r.position, r.id
	`, packageIDs)
	if err != nil {
		return nil, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRelationRow])
	if err != nil {
		return nil, err
	}
	out := make(map[int64]packageRelations, len(packageIDs))
	for _, record := range records {
		reference := PackageReference{
			SoftwareID:     record.SoftwareID,
			SoftwareName:   record.SoftwareName,
			PackageVersion: record.TargetVersion,
		}
		if record.TargetPackageID != nil {
			reference.PackageID = *record.TargetPackageID
		}
		rel := out[record.PackageID]
		switch record.RelationKind {
		case relationKindRequires:
			rel.requires = append(rel.requires, reference)
		case relationKindUpdateFor:
			rel.updateFor = append(rel.updateFor, reference)
		}
		out[record.PackageID] = rel
	}
	return out, nil
}

func packageIDs(packages []Package) []int64 {
	ids := make([]int64, 0, len(packages))
	for _, pkg := range packages {
		if pkg.ID <= 0 {
			continue
		}
		ids = append(ids, pkg.ID)
	}
	return dbutil.Dedup(ids)
}

func optionalPositiveInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func nonNilReferences(values []PackageReference) []PackageReference {
	if values == nil {
		return []PackageReference{}
	}
	return values
}
