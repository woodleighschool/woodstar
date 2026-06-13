package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/munki/artifacts"
)

type artifactStore interface {
	GetByID(context.Context, int64) (*artifacts.Artifact, error)
}

type Store struct {
	db        *database.DB
	q         *sqlc.Queries
	artifacts artifactStore
}

func NewStore(db *database.DB, artifacts artifactStore) *Store {
	return &Store{
		db:        db,
		q:         db.Queries(),
		artifacts: artifacts,
	}
}

func (s *Store) Create(ctx context.Context, softwareID int64, params PackageMutation) (*PackageRecord, error) {
	if softwareID <= 0 {
		return nil, fmt.Errorf("%w: software_id is required", dbutil.ErrInvalidInput)
	}
	params, fields, err := s.prepareMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	row, err := qtx.CreateMunkiPackage(ctx, createMunkiPackageParams(softwareID, params, fields))
	if err != nil {
		return nil, mapMutationError(err)
	}
	if err := replacePackageRelations(
		ctx,
		qtx,
		row.ID,
		sqlc.MunkiPackageRelationKindRequires,
		params.Requires,
	); err != nil {
		return nil, mapMutationError(err)
	}
	if err := replacePackageRelations(
		ctx,
		qtx,
		row.ID,
		sqlc.MunkiPackageRelationKindUpdateFor,
		params.UpdateFor,
	); err != nil {
		return nil, mapMutationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, row.ID)
}

func (s *Store) Update(ctx context.Context, id int64, params PackageMutation) (*PackageRecord, error) {
	params, fields, err := s.prepareMutation(ctx, params)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	row, err := qtx.UpdateMunkiPackage(ctx, updateMunkiPackageParams(id, params, fields))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, mapMutationError(err)
	}
	if err := replacePackageRelations(
		ctx,
		qtx,
		row.ID,
		sqlc.MunkiPackageRelationKindRequires,
		params.Requires,
	); err != nil {
		return nil, mapMutationError(err)
	}
	if err := replacePackageRelations(
		ctx,
		qtx,
		row.ID,
		sqlc.MunkiPackageRelationKindUpdateFor,
		params.UpdateFor,
	); err != nil {
		return nil, mapMutationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, row.ID)
}

func (s *Store) GetByID(ctx context.Context, id int64) (*PackageRecord, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := s.q.GetMunkiPackageByID(ctx, sqlc.GetMunkiPackageByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	record, err := packageRecordFromRecord(packageRecordFromSQLC(row))
	if err != nil {
		return nil, err
	}
	records, err := s.attachRecordRelations(ctx, []PackageRecord{record})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	rows, err := s.q.DeleteMunkiPackage(ctx, sqlc.DeleteMunkiPackageParams{ID: id})
	if err != nil {
		return mapDeleteError(err)
	}
	if rows == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

func mapDeleteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation, pgerrcode.RestrictViolation:
		return fmt.Errorf("%w: Munki package is still referenced", dbutil.ErrConflict)
	}
	return mapMutationError(err)
}

func (s *Store) List(ctx context.Context, params PackageListParams) ([]PackageRecord, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
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
	var count int
	countSQL, countArgs := listQuery.BuildCount()
	if err := s.db.Pool().QueryRow(ctx, countSQL, countArgs...).Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := listQuery.Build()
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRecord])
	if err != nil {
		return nil, 0, err
	}
	packageRecords, err := packageRecordsFromRecords(records)
	if err != nil {
		return nil, 0, err
	}
	packageRecords, err = s.attachRecordRelations(ctx, packageRecords)
	if err != nil {
		return nil, 0, err
	}
	return packageRecords, count, nil
}

func (s *Store) prepareMutation(
	ctx context.Context,
	params PackageMutation,
) (PackageMutation, packageJSONFields, error) {
	params = cleanMutation(params)
	if err := params.Validate(); err != nil {
		return PackageMutation{}, packageJSONFields{}, err
	}
	params, err := s.normalizePackageArtifacts(ctx, params)
	if err != nil {
		return PackageMutation{}, packageJSONFields{}, err
	}
	fields, err := packageJSONFromMutation(params)
	if err != nil {
		return PackageMutation{}, packageJSONFields{}, err
	}
	return params, fields, nil
}

func (s *Store) normalizePackageArtifacts(ctx context.Context, params PackageMutation) (PackageMutation, error) {
	if params.InstallerArtifactID != nil {
		artifact, err := s.artifacts.GetByID(ctx, *params.InstallerArtifactID)
		if err != nil {
			return params, err
		}
		if artifact.Kind != artifacts.ArtifactKindPackage {
			return params, fmt.Errorf(
				"%w: installer_artifact_id must reference a package artifact",
				dbutil.ErrInvalidInput,
			)
		}
	}
	if params.UninstallerArtifactID != nil {
		artifact, err := s.artifacts.GetByID(ctx, *params.UninstallerArtifactID)
		if err != nil {
			return params, err
		}
		if artifact.Kind != artifacts.ArtifactKindPackage {
			return params, fmt.Errorf(
				"%w: uninstaller_artifact_id must reference a package artifact",
				dbutil.ErrInvalidInput,
			)
		}
	}
	return params, nil
}

func cleanMutation(params PackageMutation) PackageMutation {
	params.Version = strings.TrimSpace(params.Version)
	params.InstallerType = InstallerType(strings.TrimSpace(string(params.InstallerType)))
	if params.InstallerType == "" {
		params.InstallerType = InstallerTypePkg
	}
	params.UninstallMethod = UninstallMethod(strings.TrimSpace(string(params.UninstallMethod)))
	if params.UninstallMethod == "" {
		params.UninstallMethod = UninstallMethodNone
	}
	params.RestartAction = RestartAction(strings.TrimSpace(string(params.RestartAction)))
	params.MinimumMunkiVersion = strings.TrimSpace(params.MinimumMunkiVersion)
	params.MinimumOSVersion = strings.TrimSpace(params.MinimumOSVersion)
	params.MaximumOSVersion = strings.TrimSpace(params.MaximumOSVersion)
	params.SupportedArchitectures = cleanStringList(params.SupportedArchitectures)
	params.BlockingApplications = cleanOptionalStringList(params.BlockingApplications)
	params.Requires = cleanReferences(params.Requires)
	params.UpdateFor = cleanReferences(params.UpdateFor)
	params.PackagePath = strings.TrimSpace(params.PackagePath)
	params.Notes = strings.TrimSpace(params.Notes)
	params.InstallerChoicesXML = cleanInstallerChoices(params.InstallerChoicesXML)
	params.InstallerEnvironment = cleanInstallerEnvironment(params.InstallerEnvironment)
	params.Installs = cleanInstallItems(params.Installs)
	params.Receipts = cleanReceipts(params.Receipts)
	params.ItemsToCopy = cleanItemsToCopy(params.ItemsToCopy)
	params.PreinstallAlert = cleanAlert(params.PreinstallAlert)
	params.PreuninstallAlert = cleanAlert(params.PreuninstallAlert)
	if strings.TrimSpace(params.UninstallScript) != "" && params.UninstallMethod == UninstallMethodNone {
		params.UninstallMethod = UninstallMethodUninstallScript
	}
	return params
}

func cleanInstallerChoices(values []PackageInstallerChoice) []PackageInstallerChoice {
	out := make([]PackageInstallerChoice, 0, len(values))
	for _, value := range values {
		value.ChoiceIdentifier = strings.TrimSpace(value.ChoiceIdentifier)
		value.ChoiceAttribute = strings.TrimSpace(value.ChoiceAttribute)
		out = append(out, value)
	}
	return out
}

func packageFromRecord(row packageRecord) (Package, error) {
	installerChoices, err := decodePackageJSON[PackageInstallerChoice](row.InstallerChoicesXML)
	if err != nil {
		return Package{}, err
	}
	installerEnvironment, err := decodePackageJSON[PackageInstallerEnvironmentVariable](row.InstallerEnvironment)
	if err != nil {
		return Package{}, err
	}
	installs, err := decodePackageJSON[PackageInstallItem](row.Installs)
	if err != nil {
		return Package{}, err
	}
	receipts, err := decodePackageJSON[PackageReceipt](row.Receipts)
	if err != nil {
		return Package{}, err
	}
	itemsToCopy, err := decodePackageJSON[PackageItemToCopy](row.ItemsToCopy)
	if err != nil {
		return Package{}, err
	}
	return Package{
		ID:                          row.ID,
		SoftwareID:                  row.SoftwareID,
		SoftwareName:                row.SoftwareName,
		SoftwareDescription:         row.SoftwareDescription,
		SoftwareCategory:            row.SoftwareCategory,
		SoftwareDeveloper:           row.SoftwareDeveloper,
		Version:                     row.Version,
		InstallerType:               InstallerType(row.InstallerType),
		UnattendedInstall:           row.UnattendedInstall,
		UnattendedUninstall:         row.UnattendedUninstall,
		UninstallMethod:             UninstallMethod(row.UninstallMethod),
		RestartAction:               RestartAction(row.RestartAction),
		MinimumMunkiVersion:         row.MinimumMunkiVersion,
		MinimumOSVersion:            row.MinimumOSVersion,
		MaximumOSVersion:            row.MaximumOSVersion,
		SupportedArchitectures:      nonNilStrings(row.SupportedArchitectures),
		BlockingApplications:        row.BlockingApplications,
		InstallableCondition:        row.InstallableCondition,
		BlockingAppsManualQuit:      row.BlockingAppsManualQuit,
		BlockingAppsQuitScript:      row.BlockingAppsQuitScript,
		Requires:                    []PackageReference{},
		UpdateFor:                   []PackageReference{},
		OnDemand:                    row.OnDemand,
		Precache:                    row.Precache,
		Autoremove:                  row.Autoremove,
		AppleItem:                   row.AppleItem,
		SuppressBundleRelocation:    row.SuppressBundleRelocation,
		ForceInstallAfterDate:       row.ForceInstallAfterDate,
		InstalledSize:               row.InstalledSize,
		PackagePath:                 row.PackagePath,
		InstallerChoicesXML:         installerChoices,
		InstallerEnvironment:        installerEnvironment,
		Installs:                    installs,
		Receipts:                    receipts,
		ItemsToCopy:                 itemsToCopy,
		Notes:                       row.Notes,
		InstallcheckScript:          row.InstallcheckScript,
		UninstallcheckScript:        row.UninstallcheckScript,
		PreinstallScript:            row.PreinstallScript,
		PostinstallScript:           row.PostinstallScript,
		PreuninstallScript:          row.PreuninstallScript,
		PostuninstallScript:         row.PostuninstallScript,
		UninstallScript:             row.UninstallScript,
		VersionScript:               row.VersionScript,
		PreinstallAlert:             row.PreinstallAlert(),
		PreuninstallAlert:           row.PreuninstallAlert(),
		InstallerArtifactID:         row.InstallerArtifactID,
		InstallerArtifactLocation:   stringPtrValue(row.InstallerArtifactLocation),
		UninstallerArtifactID:       row.UninstallerArtifactID,
		UninstallerArtifactLocation: stringPtrValue(row.UninstallerArtifactLocation),
		Eligible:                    row.Eligible,
		CreatedAt:                   row.CreatedAt,
		UpdatedAt:                   row.UpdatedAt,
	}, nil
}

func packageRecordFromRecord(row packageRecord) (PackageRecord, error) {
	pkg, err := packageFromRecord(row)
	if err != nil {
		return PackageRecord{}, err
	}
	return PackageRecord{
		Package:      pkg,
		SoftwareIcon: softwareIconFromRecord(row),
	}, nil
}

func packageRecordsFromRecords(records []packageRecord) ([]PackageRecord, error) {
	packages := make([]PackageRecord, len(records))
	for i, row := range records {
		pkg, err := packageRecordFromRecord(row)
		if err != nil {
			return nil, err
		}
		packages[i] = pkg
	}
	return packages, nil
}

func softwareIconFromRecord(row packageRecord) IconRef {
	return IconRef{
		Name:             row.SoftwareIconName,
		Hash:             row.SoftwareIconHash,
		ArtifactID:       row.SoftwareIconArtifactID,
		ArtifactLocation: stringPtrValue(row.SoftwareIconArtifactLocation),
	}
}

func (s *Store) attachRecordRelations(ctx context.Context, records []PackageRecord) ([]PackageRecord, error) {
	pkgs := make([]Package, len(records))
	for i, record := range records {
		pkgs[i] = record.Package
	}
	pkgs, err := s.AttachRelations(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	for i := range records {
		records[i].Package = pkgs[i]
	}
	return records, nil
}

// FromEffectiveRow maps the host-effective package query row into a joined package read model.
func FromEffectiveRow(row sqlc.ListEffectiveMunkiPackagesForHostRow) (PackageRecord, error) {
	return packageRecordFromRecord(packageRecordFromEffectiveSQLC(row))
}

func packageListWhere(params PackageListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.SoftwareID > 0 {
		where.Add("p.software_id = " + where.Arg(params.SoftwareID))
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

func packageOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":       {SQL: "lower(s.name)"},
		"version":    {SQL: "lower(p.version)"},
		"software":   {SQL: "lower(s.name)"},
		"updated_at": {SQL: "p.updated_at"},
	}
}

type packageJSONFields struct {
	InstallerChoicesXML  []byte
	InstallerEnvironment []byte
	Installs             []byte
	Receipts             []byte
	ItemsToCopy          []byte
}

func packageJSONFromMutation(params PackageMutation) (packageJSONFields, error) {
	installerChoices, err := marshalPackageJSON(params.InstallerChoicesXML)
	if err != nil {
		return packageJSONFields{}, err
	}
	installerEnvironment, err := marshalPackageJSON(params.InstallerEnvironment)
	if err != nil {
		return packageJSONFields{}, err
	}
	installs, err := marshalPackageJSON(params.Installs)
	if err != nil {
		return packageJSONFields{}, err
	}
	receipts, err := marshalPackageJSON(params.Receipts)
	if err != nil {
		return packageJSONFields{}, err
	}
	itemsToCopy, err := marshalPackageJSON(params.ItemsToCopy)
	if err != nil {
		return packageJSONFields{}, err
	}
	return packageJSONFields{
		InstallerChoicesXML:  installerChoices,
		InstallerEnvironment: installerEnvironment,
		Installs:             installs,
		Receipts:             receipts,
		ItemsToCopy:          itemsToCopy,
	}, nil
}

func marshalPackageJSON[T any](values []T) ([]byte, error) {
	if values == nil {
		values = []T{}
	}
	return json.Marshal(values)
}

func decodePackageJSON[T any](raw []byte) ([]T, error) {
	if len(raw) == 0 {
		return []T{}, nil
	}
	var out []T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return []T{}, nil
	}
	return out, nil
}

func createMunkiPackageParams(
	softwareID int64,
	params PackageMutation,
	fields packageJSONFields,
) sqlc.CreateMunkiPackageParams {
	return sqlc.CreateMunkiPackageParams{
		SoftwareID:                         softwareID,
		Version:                            params.Version,
		InstallerType:                      string(params.InstallerType),
		UninstallMethod:                    string(params.UninstallMethod),
		RestartAction:                      string(params.RestartAction),
		MinimumMunkiVersion:                params.MinimumMunkiVersion,
		MinimumOSVersion:                   params.MinimumOSVersion,
		MaximumOSVersion:                   params.MaximumOSVersion,
		SupportedArchitectures:             params.SupportedArchitectures,
		BlockingApplications:               params.BlockingApplications,
		InstallableCondition:               params.InstallableCondition,
		BlockingApplicationsManualQuitOnly: params.BlockingAppsManualQuit,
		BlockingApplicationsQuitScript:     params.BlockingAppsQuitScript,
		UnattendedInstall:                  params.UnattendedInstall,
		UnattendedUninstall:                params.UnattendedUninstall,
		OnDemand:                           params.OnDemand,
		Precache:                           params.Precache,
		Autoremove:                         params.Autoremove,
		AppleItem:                          params.AppleItem,
		SuppressBundleRelocation:           params.SuppressBundleRelocation,
		ForceInstallAfterDate:              params.ForceInstallAfterDate,
		InstalledSize:                      params.InstalledSize,
		PackagePath:                        params.PackagePath,
		InstallerChoicesXml:                fields.InstallerChoicesXML,
		InstallerEnvironment:               fields.InstallerEnvironment,
		Installs:                           fields.Installs,
		Receipts:                           fields.Receipts,
		ItemsToCopy:                        fields.ItemsToCopy,
		Notes:                              params.Notes,
		InstallcheckScript:                 params.InstallcheckScript,
		UninstallcheckScript:               params.UninstallcheckScript,
		PreinstallScript:                   params.PreinstallScript,
		PostinstallScript:                  params.PostinstallScript,
		PreuninstallScript:                 params.PreuninstallScript,
		PostuninstallScript:                params.PostuninstallScript,
		UninstallScript:                    params.UninstallScript,
		VersionScript:                      params.VersionScript,
		PreinstallAlertEnabled:             params.PreinstallAlert.Enabled,
		PreinstallAlertTitle:               params.PreinstallAlert.Title,
		PreinstallAlertDetail:              params.PreinstallAlert.Detail,
		PreinstallAlertOkLabel:             params.PreinstallAlert.OKLabel,
		PreinstallAlertCancelLabel:         params.PreinstallAlert.CancelLabel,
		PreuninstallAlertEnabled:           params.PreuninstallAlert.Enabled,
		PreuninstallAlertTitle:             params.PreuninstallAlert.Title,
		PreuninstallAlertDetail:            params.PreuninstallAlert.Detail,
		PreuninstallAlertOkLabel:           params.PreuninstallAlert.OKLabel,
		PreuninstallAlertCancelLabel:       params.PreuninstallAlert.CancelLabel,
		InstallerArtifactID:                params.InstallerArtifactID,
		UninstallerArtifactID:              params.UninstallerArtifactID,
		Eligible:                           params.Eligible,
	}
}

func updateMunkiPackageParams(
	id int64,
	params PackageMutation,
	fields packageJSONFields,
) sqlc.UpdateMunkiPackageParams {
	return sqlc.UpdateMunkiPackageParams{
		Version:                            params.Version,
		InstallerType:                      string(params.InstallerType),
		UninstallMethod:                    string(params.UninstallMethod),
		RestartAction:                      string(params.RestartAction),
		MinimumMunkiVersion:                params.MinimumMunkiVersion,
		MinimumOSVersion:                   params.MinimumOSVersion,
		MaximumOSVersion:                   params.MaximumOSVersion,
		SupportedArchitectures:             params.SupportedArchitectures,
		BlockingApplications:               params.BlockingApplications,
		InstallableCondition:               params.InstallableCondition,
		BlockingApplicationsManualQuitOnly: params.BlockingAppsManualQuit,
		BlockingApplicationsQuitScript:     params.BlockingAppsQuitScript,
		UnattendedInstall:                  params.UnattendedInstall,
		UnattendedUninstall:                params.UnattendedUninstall,
		OnDemand:                           params.OnDemand,
		Precache:                           params.Precache,
		Autoremove:                         params.Autoremove,
		AppleItem:                          params.AppleItem,
		SuppressBundleRelocation:           params.SuppressBundleRelocation,
		ForceInstallAfterDate:              params.ForceInstallAfterDate,
		InstalledSize:                      params.InstalledSize,
		PackagePath:                        params.PackagePath,
		InstallerChoicesXml:                fields.InstallerChoicesXML,
		InstallerEnvironment:               fields.InstallerEnvironment,
		Installs:                           fields.Installs,
		Receipts:                           fields.Receipts,
		ItemsToCopy:                        fields.ItemsToCopy,
		Notes:                              params.Notes,
		InstallcheckScript:                 params.InstallcheckScript,
		UninstallcheckScript:               params.UninstallcheckScript,
		PreinstallScript:                   params.PreinstallScript,
		PostinstallScript:                  params.PostinstallScript,
		PreuninstallScript:                 params.PreuninstallScript,
		PostuninstallScript:                params.PostuninstallScript,
		UninstallScript:                    params.UninstallScript,
		VersionScript:                      params.VersionScript,
		PreinstallAlertEnabled:             params.PreinstallAlert.Enabled,
		PreinstallAlertTitle:               params.PreinstallAlert.Title,
		PreinstallAlertDetail:              params.PreinstallAlert.Detail,
		PreinstallAlertOkLabel:             params.PreinstallAlert.OKLabel,
		PreinstallAlertCancelLabel:         params.PreinstallAlert.CancelLabel,
		PreuninstallAlertEnabled:           params.PreuninstallAlert.Enabled,
		PreuninstallAlertTitle:             params.PreuninstallAlert.Title,
		PreuninstallAlertDetail:            params.PreuninstallAlert.Detail,
		PreuninstallAlertOkLabel:           params.PreuninstallAlert.OKLabel,
		PreuninstallAlertCancelLabel:       params.PreuninstallAlert.CancelLabel,
		InstallerArtifactID:                params.InstallerArtifactID,
		UninstallerArtifactID:              params.UninstallerArtifactID,
		Eligible:                           params.Eligible,
		ID:                                 id,
	}
}

type packageRecord struct {
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
	PackagePath                  string
	InstallerChoicesXML          []byte `db:"installer_choices_xml"`
	InstallerEnvironment         []byte
	Installs                     []byte
	Receipts                     []byte
	ItemsToCopy                  []byte
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
	InstallerArtifactID          *int64
	InstallerArtifactLocation    *string
	UninstallerArtifactID        *int64
	UninstallerArtifactLocation  *string
	SoftwareIconName             string
	SoftwareIconHash             string
	SoftwareIconArtifactID       *int64
	SoftwareIconArtifactLocation *string
	Eligible                     bool
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

func (row packageRecord) PreinstallAlert() PackageAlert {
	return PackageAlert{
		Enabled:     row.PreinstallAlertEnabled,
		Title:       row.PreinstallAlertTitle,
		Detail:      row.PreinstallAlertDetail,
		OKLabel:     row.PreinstallAlertOKLabel,
		CancelLabel: row.PreinstallAlertCancelLabel,
	}
}

func (row packageRecord) PreuninstallAlert() PackageAlert {
	return PackageAlert{
		Enabled:     row.PreuninstallAlertEnabled,
		Title:       row.PreuninstallAlertTitle,
		Detail:      row.PreuninstallAlertDetail,
		OKLabel:     row.PreuninstallAlertOKLabel,
		CancelLabel: row.PreuninstallAlertCancelLabel,
	}
}

func packageRecordFromSQLC(row sqlc.GetMunkiPackageByIDRow) packageRecord {
	return packageRecord{
		ID:                           row.ID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDescription:          row.SoftwareDescription,
		SoftwareCategory:             row.SoftwareCategory,
		SoftwareDeveloper:            row.SoftwareDeveloper,
		SoftwareIconName:             row.SoftwareIconName,
		SoftwareIconHash:             row.SoftwareIconHash,
		SoftwareIconArtifactID:       row.SoftwareIconArtifactID,
		SoftwareIconArtifactLocation: row.SoftwareIconArtifactLocation,
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
		InstallerChoicesXML:          row.InstallerChoicesXml,
		InstallerEnvironment:         row.InstallerEnvironment,
		Installs:                     row.Installs,
		Receipts:                     row.Receipts,
		ItemsToCopy:                  row.ItemsToCopy,
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
		InstallerArtifactID:          row.InstallerArtifactID,
		InstallerArtifactLocation:    row.InstallerArtifactLocation,
		UninstallerArtifactID:        row.UninstallerArtifactID,
		UninstallerArtifactLocation:  row.UninstallerArtifactLocation,
		Eligible:                     row.Eligible,
		CreatedAt:                    row.CreatedAt,
		UpdatedAt:                    row.UpdatedAt,
	}
}

func packageRecordFromEffectiveSQLC(row sqlc.ListEffectiveMunkiPackagesForHostRow) packageRecord {
	return packageRecord{
		ID:                           row.PackageID,
		SoftwareID:                   row.SoftwareID,
		SoftwareName:                 row.SoftwareName,
		SoftwareDescription:          row.SoftwareDescription,
		SoftwareCategory:             row.SoftwareCategory,
		SoftwareDeveloper:            row.SoftwareDeveloper,
		SoftwareIconName:             row.SoftwareIconName,
		SoftwareIconHash:             row.SoftwareIconHash,
		SoftwareIconArtifactID:       row.SoftwareIconArtifactID,
		SoftwareIconArtifactLocation: row.SoftwareIconArtifactLocation,
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
		InstallerChoicesXML:          row.InstallerChoicesXml,
		InstallerEnvironment:         row.InstallerEnvironment,
		Installs:                     row.Installs,
		Receipts:                     row.Receipts,
		ItemsToCopy:                  row.ItemsToCopy,
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
		InstallerArtifactID:          row.InstallerArtifactID,
		InstallerArtifactLocation:    row.InstallerArtifactLocation,
		UninstallerArtifactID:        row.UninstallerArtifactID,
		UninstallerArtifactLocation:  row.UninstallerArtifactLocation,
		Eligible:                     true,
	}
}

const packageSelectSQL = `
SELECT
	p.id,
	p.software_id,
	s.name AS software_name,
	s.description AS software_description,
	s.category AS software_category,
	s.developer AS software_developer,
	s.icon_name AS software_icon_name,
	s.icon_hash AS software_icon_hash,
	s.icon_artifact_id AS software_icon_artifact_id,
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
	p.installer_artifact_id,
	art.location AS installer_artifact_location,
	p.uninstaller_artifact_id,
	uninstaller.location AS uninstaller_artifact_location,
	software_icon.location AS software_icon_artifact_location,
	p.eligible,
	p.created_at,
	p.updated_at
FROM munki_packages p
JOIN munki_software s ON s.id = p.software_id
LEFT JOIN munki_artifacts art ON art.id = p.installer_artifact_id
LEFT JOIN munki_artifacts uninstaller ON uninstaller.id = p.uninstaller_artifact_id
LEFT JOIN munki_artifacts software_icon ON software_icon.id = s.icon_artifact_id`

func replacePackageRelations(
	ctx context.Context,
	q *sqlc.Queries,
	packageID int64,
	kind sqlc.MunkiPackageRelationKind,
	references []PackageReference,
) error {
	if err := q.DeleteMunkiPackageRelationsByKind(ctx, sqlc.DeleteMunkiPackageRelationsByKindParams{
		PackageID:    packageID,
		RelationKind: kind,
	}); err != nil {
		return err
	}
	for position, ref := range references {
		if err := q.CreateMunkiPackageRelation(ctx, sqlc.CreateMunkiPackageRelationParams{
			PackageID:        packageID,
			RelationKind:     kind,
			TargetSoftwareID: ref.SoftwareID,
			TargetPackageID:  optionalPositiveInt64(ref.PackageID),
			Position:         int32(position),
		}); err != nil {
			return err
		}
	}
	return nil
}

type packageRelationRecord struct {
	PackageID       int64
	RelationKind    sqlc.MunkiPackageRelationKind
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
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[packageRelationRecord])
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
		case sqlc.MunkiPackageRelationKindRequires:
			rel.requires = append(rel.requires, reference)
		case sqlc.MunkiPackageRelationKindUpdateFor:
			rel.updateFor = append(rel.updateFor, reference)
		}
		out[record.PackageID] = rel
	}
	return out, nil
}

func packageIDs(packages []Package) []int64 {
	ids := make([]int64, 0, len(packages))
	seen := make(map[int64]struct{}, len(packages))
	for _, pkg := range packages {
		if pkg.ID <= 0 {
			continue
		}
		if _, ok := seen[pkg.ID]; ok {
			continue
		}
		seen[pkg.ID] = struct{}{}
		ids = append(ids, pkg.ID)
	}
	return ids
}

func cleanReferences(values []PackageReference) []PackageReference {
	out := make([]PackageReference, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := packageReferenceKey(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, PackageReference{
			SoftwareID: value.SoftwareID,
			PackageID:  value.PackageID,
		})
	}
	return out
}

func packageReferenceKey(ref PackageReference) string {
	if ref.PackageID > 0 {
		return fmt.Sprintf("package:%d", ref.PackageID)
	}
	if ref.SoftwareID > 0 {
		return fmt.Sprintf("software:%d", ref.SoftwareID)
	}
	return fmt.Sprintf("invalid:%d:%d", ref.SoftwareID, ref.PackageID)
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

func cleanInstallerEnvironment(
	values []PackageInstallerEnvironmentVariable,
) []PackageInstallerEnvironmentVariable {
	out := make([]PackageInstallerEnvironmentVariable, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, PackageInstallerEnvironmentVariable{Name: name, Value: value.Value})
	}
	return out
}

func cleanInstallItems(values []PackageInstallItem) []PackageInstallItem {
	out := make([]PackageInstallItem, 0, len(values))
	for _, value := range values {
		value.Type = PackageInstallItemType(strings.TrimSpace(string(value.Type)))
		if value.Type == "" {
			value.Type = PackageInstallItemFile
		}
		value.Path = strings.TrimSpace(value.Path)
		value.BundleIdentifier = strings.TrimSpace(value.BundleIdentifier)
		value.BundleName = strings.TrimSpace(value.BundleName)
		value.BundleShortVersion = strings.TrimSpace(value.BundleShortVersion)
		value.BundleVersion = strings.TrimSpace(value.BundleVersion)
		value.VersionComparisonKey = strings.TrimSpace(value.VersionComparisonKey)
		value.MD5Checksum = strings.TrimSpace(value.MD5Checksum)
		value.MinimumOSVersion = strings.TrimSpace(value.MinimumOSVersion)
		value.InstallerItemLocation = strings.TrimSpace(value.InstallerItemLocation)
		if value.Path != "" {
			out = append(out, value)
		}
	}
	return out
}

func cleanReceipts(values []PackageReceipt) []PackageReceipt {
	out := make([]PackageReceipt, 0, len(values))
	for _, value := range values {
		value.PackageID = strings.TrimSpace(value.PackageID)
		value.Version = strings.TrimSpace(value.Version)
		if value.PackageID != "" {
			out = append(out, value)
		}
	}
	return out
}

func cleanItemsToCopy(values []PackageItemToCopy) []PackageItemToCopy {
	out := make([]PackageItemToCopy, 0, len(values))
	for _, value := range values {
		value.SourceItem = strings.TrimSpace(value.SourceItem)
		value.DestinationPath = strings.TrimSpace(value.DestinationPath)
		value.DestinationItem = strings.TrimSpace(value.DestinationItem)
		value.User = strings.TrimSpace(value.User)
		value.Group = strings.TrimSpace(value.Group)
		value.Mode = strings.TrimSpace(value.Mode)
		if value.SourceItem != "" || value.DestinationPath != "" {
			out = append(out, value)
		}
	}
	return out
}

func cleanAlert(alert PackageAlert) PackageAlert {
	alert.Title = strings.TrimSpace(alert.Title)
	alert.Detail = strings.TrimSpace(alert.Detail)
	alert.OKLabel = strings.TrimSpace(alert.OKLabel)
	alert.CancelLabel = strings.TrimSpace(alert.CancelLabel)
	return alert
}

func mapMutationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	switch database.SQLState(err) {
	case pgerrcode.ForeignKeyViolation:
		return dbutil.ErrNotFound
	case pgerrcode.UniqueViolation:
		return dbutil.ErrAlreadyExists
	case pgerrcode.InvalidTextRepresentation,
		pgerrcode.NotNullViolation,
		pgerrcode.CheckViolation:
		return fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	return err
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
