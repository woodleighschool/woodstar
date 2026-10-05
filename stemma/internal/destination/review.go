package destination

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

// creationReview accounts for every initial field. Names come from resolution;
// ids and complete collection members remain in the change's After document.
func creationReview(kind string, fields map[string]json.RawMessage, metadata metadata) []string {
	groups := make([][]string, 6)
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		group := 5
		switch key {
		case "name", "display_name", "description", "category", "developer", "version", "installer_type", "installed_size":
			group = 0
		case "targets":
			group = 1
		case "installs", "receipts", "installcheck_script", "uninstallcheck_script", "version_script":
			group = 2
		case "requires", "update_for", "minimum_munki_version", "minimum_os_version", "maximum_os_version", "supported_architectures", "installable_condition":
			group = 3
		case "unattended_install", "unattended_uninstall", "uninstallable", "uninstall_method", "restart_action", "on_demand", "force_install_after_date", "blocking_applications", "blocking_applications_none", "blocking_applications_manual_quit_only", "blocking_applications_quit_script", "installer_choices_xml", "installer_environment", "items_to_copy", "package_path", "preinstall_script", "postinstall_script", "preuninstall_script", "postuninstall_script", "uninstall_script", "preinstall_alert", "preuninstall_alert":
			group = 4
		}
		lines := []string{key + ": " + reviewValue(fields[key])}
		switch key {
		case "targets":
			var targets software.Targets
			if json.Unmarshal(fields[key], &targets) == nil {
				lines = []string{"targets:"}
				for _, target := range targets.Include {
					name := metadata.labelNames[target.LabelID]
					if name == "" {
						name = fmt.Sprintf("label %d", target.LabelID)
					}
					actions := make([]string, len(target.Actions))
					for i, action := range target.Actions {
						actions[i] = string(action)
					}
					selection := string(target.Package.Strategy)
					if target.Package.PackageID != nil {
						selection += fmt.Sprintf(" package %d", *target.Package.PackageID)
					}
					lines = append(lines, "  include "+name+": "+strings.Join(actions, ", ")+" ("+selection+")")
				}
				if len(targets.Include) == 0 {
					lines = append(lines, "  include: none")
				}
				for _, target := range targets.Exclude {
					name := metadata.labelNames[target.LabelID]
					if name == "" {
						name = fmt.Sprintf("label %d", target.LabelID)
					}
					lines = append(lines, "  exclude "+name)
				}
				if len(targets.Exclude) == 0 {
					lines = append(lines, "  exclude: none")
				}
			}
		case "requires", "update_for":
			var refs []packages.PackageReferenceMutation
			if json.Unmarshal(fields[key], &refs) == nil {
				names := make([]string, 0, len(refs))
				for _, ref := range refs {
					name := metadata.referenceNames[ref]
					if name == "" {
						name = string(raw(ref))
					}
					names = append(names, name)
				}
				value := "none"
				if len(names) > 0 {
					value = strings.Join(names, ", ")
				}
				lines = []string{key + ": " + value}
			}
		case "installs", "receipts":
			var items []map[string]json.RawMessage
			if json.Unmarshal(fields[key], &items) == nil {
				noun := "entries"
				if len(items) == 1 {
					noun = "entry"
				}
				lines = []string{fmt.Sprintf("%s: %d %s", key, len(items), noun)}
				id := "path"
				if key == "receipts" {
					id = "package_id"
				}
				for _, item := range items {
					name := reviewValue(item[id])
					if key == "receipts" {
						if version, ok := item["version"]; ok {
							name += " · " + reviewValue(version)
						}
						if string(item["optional"]) == "true" {
							name += " (optional)"
						}
					} else {
						version := cmp.Or(string(item["bundle_short_version"]), string(item["bundle_version"]))
						var comparison string
						if json.Unmarshal(item["version_comparison_key"], &comparison) == nil && comparison != "" {
							switch comparison {
							case "CFBundleShortVersionString":
								version = string(item["bundle_short_version"])
							case "CFBundleVersion":
								version = string(item["bundle_version"])
							default:
								version = ""
							}
						}
						if version != "" {
							name += " · " + reviewValue(json.RawMessage(version))
						}
						if value, ok := item["version_comparison_key"]; ok {
							name += " (version key: " + reviewValue(value) + ")"
						}
					}
					lines = append(lines, "  "+name)
				}
			}
		case "minimum_os_version", "maximum_os_version", "minimum_munki_version", "restart_action":
			if string(fields[key]) == `""` {
				lines = []string{key + ": none"}
			}
		case "items_to_copy":
			if string(fields[key]) == "[]" {
				lines = []string{key + ": none"}
			}
		case "installed_size":
			lines[0] += " KiB"
		}
		for _, line := range lines {
			groups[group] = append(groups[group], "  "+line)
		}
	}
	var lines []string
	for i, heading := range []string{kind, "Targeting", "Detection", "Requirements", "Install behaviour", "Other fields"} {
		if len(groups[i]) > 0 {
			lines = append(lines, heading)
			lines = append(lines, groups[i]...)
		}
	}
	return lines
}

func reviewValue(value json.RawMessage) string {
	var text string
	if string(value) != "null" && json.Unmarshal(value, &text) == nil {
		if strings.Contains(text, "\n") {
			return fmt.Sprintf("%d lines", len(strings.Split(strings.TrimSuffix(text, "\n"), "\n")))
		}
		if text == "" {
			return `""`
		}
		return text
	}
	return string(value)
}
