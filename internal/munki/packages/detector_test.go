package packages

import "testing"

func TestInstallationDetectorFromInstallsIsConservative(t *testing.T) {
	tests := map[string]struct {
		installs []PackageInstallItem
		want     *installationDetector
	}{
		"explicit short version": {
			installs: []PackageInstallItem{{
				Type:                 PackageInstallItemApplication,
				Path:                 " /Applications/Example.app ",
				BundleIdentifier:     " com.example.app ",
				BundleShortVersion:   "1.2.3",
				BundleVersion:        "123",
				VersionComparisonKey: " CFBundleShortVersionString ",
			}},
			want: &installationDetector{
				BundleIdentifier: "com.example.app",
				ExpectedPath:     "/Applications/Example.app",
				VersionSource:    installationVersionSourceBundleShortVersion,
			},
		},
		"sole build version metadata": {
			installs: []PackageInstallItem{{
				Type:             PackageInstallItemApplication,
				Path:             "/Applications/Example.app",
				BundleIdentifier: "com.example.app",
				BundleVersion:    "123",
			}},
			want: &installationDetector{
				BundleIdentifier: "com.example.app",
				ExpectedPath:     "/Applications/Example.app",
				VersionSource:    installationVersionSourceBundleVersion,
			},
		},
		"identical candidates collapse and varying paths become optional": {
			installs: []PackageInstallItem{
				{
					Type:                 PackageInstallItemApplication,
					Path:                 "/Applications/Example.app",
					BundleIdentifier:     "com.example.app",
					VersionComparisonKey: "CFBundleVersion",
				},
				{
					Type:                 PackageInstallItemApplication,
					Path:                 "/Applications/Utilities/Example.app",
					BundleIdentifier:     "com.example.app",
					VersionComparisonKey: "CFBundleVersion",
				},
			},
			want: &installationDetector{
				BundleIdentifier: "com.example.app",
				VersionSource:    installationVersionSourceBundleVersion,
			},
		},
		"conflicting bundle identifiers": {
			installs: []PackageInstallItem{
				{Type: PackageInstallItemApplication, Path: "/Applications/One.app", BundleIdentifier: "com.example.one", BundleVersion: "1"},
				{Type: PackageInstallItemApplication, Path: "/Applications/Two.app", BundleIdentifier: "com.example.two", BundleVersion: "1"},
			},
		},
		"conflicting version sources": {
			installs: []PackageInstallItem{
				{Type: PackageInstallItemApplication, Path: "/Applications/Example.app", BundleIdentifier: "com.example.app", VersionComparisonKey: "CFBundleVersion"},
				{Type: PackageInstallItemApplication, Path: "/Applications/Example.app", BundleIdentifier: "com.example.app", VersionComparisonKey: "CFBundleShortVersionString"},
			},
		},
		"both version fields without comparison key": {
			installs: []PackageInstallItem{{
				Type:               PackageInstallItemApplication,
				Path:               "/Applications/Example.app",
				BundleIdentifier:   "com.example.app",
				BundleShortVersion: "1.2.3",
				BundleVersion:      "123",
			}},
		},
		"unsupported explicit comparison key": {
			installs: []PackageInstallItem{{
				Type:                 PackageInstallItemApplication,
				Path:                 "/Applications/Example.app",
				BundleIdentifier:     "com.example.app",
				BundleVersion:        "123",
				VersionComparisonKey: "FileVersion",
			}},
		},
		"unresolvable application invalidates otherwise valid metadata": {
			installs: []PackageInstallItem{
				{Type: PackageInstallItemApplication, Path: "/Applications/Example.app", BundleIdentifier: "com.example.app", BundleVersion: "123"},
				{Type: PackageInstallItemApplication, Path: "/Applications/Broken.app", BundleIdentifier: "com.example.broken"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := installationDetectorFromInstalls(test.installs)
			if got == nil || test.want == nil {
				if got != test.want {
					t.Fatalf("detector = %#v, want %#v", got, test.want)
				}
				return
			}
			if *got != *test.want {
				t.Fatalf("detector = %#v, want %#v", got, test.want)
			}
		})
	}
}
