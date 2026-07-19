package packages

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func (s *Store) Create(ctx context.Context, in PackageCreateMutation) (*Package, error) {
	params, err := prepareCreateMutation(in)
	if err != nil {
		return nil, err
	}
	write := newPackageWrite(in.SoftwareID, params)

	var id int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateAndLockInstallerObject(ctx, tx, params.InstallerObjectID, 0); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
INSERT INTO munki_packages (
	software_id,
	version,
	installer_type,
	uninstallable,
	uninstall_method,
	restart_action,
	minimum_munki_version,
	minimum_os_version,
	maximum_os_version,
	supported_architectures,
	blocking_applications,
	blocking_applications_none,
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
	installer_object_id
)
VALUES (
	@software_id,
	@version,
	@installer_type,
	@uninstallable,
	@uninstall_method,
	@restart_action,
	@minimum_munki_version,
	@minimum_os_version,
	@maximum_os_version,
	@supported_architectures::text[],
	@blocking_applications::text[],
	@blocking_applications_none,
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
	@force_install_after_date,
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
	@installer_object_id
)
RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
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
	params, err := prepareMutation(params)
	if err != nil {
		return nil, err
	}

	var oldObjectID *int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := validateAndLockInstallerObject(ctx, tx, params.InstallerObjectID, id); err != nil {
			return err
		}
		var softwareID int64
		if err := tx.QueryRow(ctx, `
SELECT software_id, installer_object_id
FROM munki_packages
WHERE id = $1
FOR UPDATE`, id).Scan(&softwareID, &oldObjectID); err != nil {
			return dbutil.GetError(err)
		}
		write := newPackageWrite(softwareID, params)
		write.ID = id
		var updatedID int64
		if err := tx.QueryRow(ctx, `
UPDATE munki_packages
SET
	version = @version,
	installer_type = @installer_type,
	uninstallable = @uninstallable,
	uninstall_method = @uninstall_method,
	restart_action = @restart_action,
	minimum_munki_version = @minimum_munki_version,
	minimum_os_version = @minimum_os_version,
	maximum_os_version = @maximum_os_version,
	supported_architectures = @supported_architectures::text[],
	blocking_applications = @blocking_applications::text[],
	blocking_applications_none = @blocking_applications_none,
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
	force_install_after_date = @force_install_after_date,
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
	installer_object_id = @installer_object_id,
	updated_at = now()
WHERE id = @id
RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		if err := writePackageRelations(ctx, tx, id, params); err != nil {
			return err
		}
		return s.objects.RequestDeletion(
			ctx,
			tx,
			replacedObjectID(oldObjectID, params.InstallerObjectID)...,
		)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
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
		return s.objects.RequestDeletion(ctx, tx, objectIDs...)
	})
	if err != nil {
		return deleted, err
	}
	return deleted, nil
}

func prepareMutation(params PackageMutation) (PackageMutation, error) {
	params = applyDefaults(params)
	params.normalize()
	if err := params.validate(); err != nil {
		return PackageMutation{}, err
	}
	return params, nil
}

func prepareCreateMutation(params PackageCreateMutation) (PackageMutation, error) {
	params.PackageMutation = applyDefaults(params.PackageMutation)
	params.PackageMutation.normalize()
	if err := params.validate(); err != nil {
		return PackageMutation{}, err
	}
	return params.PackageMutation, nil
}

func applyDefaults(params PackageMutation) PackageMutation {
	if params.InstallerType == "" {
		params.InstallerType = InstallerTypePkg
	}
	// supported_architectures is NOT NULL; nil means no architecture restriction.
	if params.SupportedArchitectures == nil {
		params.SupportedArchitectures = []string{}
	}
	if params.BlockingApplications == nil {
		params.BlockingApplications = []string{}
	}
	return params
}

type packageWrite struct {
	ID                           int64                       `db:"id"`
	SoftwareID                   int64                       `db:"software_id"`
	Version                      string                      `db:"version"`
	InstallerType                string                      `db:"installer_type"`
	Uninstallable                bool                        `db:"uninstallable"`
	UninstallMethod              string                      `db:"uninstall_method"`
	RestartAction                string                      `db:"restart_action"`
	MinimumMunkiVersion          string                      `db:"minimum_munki_version"`
	MinimumOSVersion             string                      `db:"minimum_os_version"`
	MaximumOSVersion             string                      `db:"maximum_os_version"`
	SupportedArchitectures       []string                    `db:"supported_architectures"`
	BlockingApplications         []string                    `db:"blocking_applications"`
	BlockingApplicationsNone     bool                        `db:"blocking_applications_none"`
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
}

func newPackageWrite(softwareID int64, params PackageMutation) packageWrite {
	return packageWrite{
		SoftwareID:                   softwareID,
		Version:                      params.Version,
		InstallerType:                string(params.InstallerType),
		Uninstallable:                params.Uninstallable,
		UninstallMethod:              string(params.UninstallMethod),
		RestartAction:                string(params.RestartAction),
		MinimumMunkiVersion:          params.MinimumMunkiVersion,
		MinimumOSVersion:             params.MinimumOSVersion,
		MaximumOSVersion:             params.MaximumOSVersion,
		SupportedArchitectures:       params.SupportedArchitectures,
		BlockingApplications:         params.BlockingApplications,
		BlockingApplicationsNone:     params.BlockingApplicationsNone,
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
	}
}
