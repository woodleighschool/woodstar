package packages

import (
	"time"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/storage"
)

func packagesFromRows(rows []packageRow) []Package {
	packages := make([]Package, len(rows))
	for i, row := range rows {
		packages[i] = packageFromRow(row)
	}
	return packages
}

// packageFromRow assembles a Package domain value from a scanned packageRow.
func packageFromRow(row packageRow) Package {
	return Package{
		ID: row.ID,
		Software: PackageSoftware{
			ID:           row.SoftwareID,
			Name:         row.SoftwareName,
			DisplayName:  row.SoftwareDisplayName,
			Description:  row.SoftwareDescription,
			Category:     row.SoftwareCategory,
			Developer:    row.SoftwareDeveloper,
			IconObjectID: row.SoftwareIconObjectID,
		},
		Version:                  row.Version,
		InstallerType:            InstallerType(row.InstallerType),
		UnattendedInstall:        row.UnattendedInstall,
		UnattendedUninstall:      row.UnattendedUninstall,
		Uninstallable:            row.Uninstallable,
		UninstallMethod:          UninstallMethod(row.UninstallMethod),
		RestartAction:            RestartAction(row.RestartAction),
		MinimumMunkiVersion:      row.MinimumMunkiVersion,
		MinimumOSVersion:         row.MinimumOSVersion,
		MaximumOSVersion:         row.MaximumOSVersion,
		SupportedArchitectures:   dbutil.NonNilSlice(row.SupportedArchitectures),
		BlockingApplications:     dbutil.NonNilSlice(row.BlockingApplications),
		BlockingApplicationsNone: row.BlockingApplicationsNone,
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
		InstallerChoicesXML:      row.InstallerChoicesXML,
		InstallerEnvironment:     row.InstallerEnvironment,
		Installs:                 row.Installs,
		Receipts:                 row.Receipts,
		ItemsToCopy:              row.ItemsToCopy,
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
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
}

// packageRow is the canonical scan target for the munki package projection.
type packageRow struct {
	ID                           int64
	SoftwareID                   int64
	SoftwareName                 string
	SoftwareDisplayName          *string
	SoftwareDescription          string
	SoftwareCategory             string
	SoftwareDeveloper            string
	Version                      string
	InstallerType                string
	Uninstallable                bool
	UninstallMethod              string
	RestartAction                string
	MinimumMunkiVersion          string
	MinimumOSVersion             string
	MaximumOSVersion             string
	SupportedArchitectures       []string
	BlockingApplications         []string
	BlockingApplicationsNone     bool
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
	InstallerChoicesXML          dbutil.JSONSlice[PackageInstallerChoice] `db:"installer_choices_xml"`
	InstallerEnvironment         dbutil.JSONSlice[PackageInstallerEnvironmentVariable]
	Installs                     dbutil.JSONSlice[PackageInstallItem]
	Receipts                     dbutil.JSONSlice[PackageReceipt]
	ItemsToCopy                  dbutil.JSONSlice[PackageItemToCopy]
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

func packageColumnsSQL() string {
	return `
	p.id,
	p.software_id,
	s.name AS software_name,
	s.display_name AS software_display_name,
	s.description AS software_description,
	s.category AS software_category,
	s.developer AS software_developer,
	s.icon_object_id AS software_icon_object_id,
	p.version,
	p.installer_type,
	p.uninstallable,
	p.uninstall_method,
	p.restart_action,
	p.minimum_munki_version,
	p.minimum_os_version,
	p.maximum_os_version,
	p.supported_architectures,
	p.blocking_applications,
	p.blocking_applications_none,
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
	p.created_at,
	p.updated_at`
}

func packageSelectSQL() string {
	return `
SELECT` + packageColumnsSQL() + `
FROM munki_packages p
JOIN munki_software s ON s.id = p.software_id
LEFT JOIN storage_objects installer_obj ON installer_obj.id = p.installer_object_id`
}
