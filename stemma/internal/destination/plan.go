package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

type desired struct {
	targets software.Targets
	pkg     packages.PackageMutation
	icon    bool
	content bool
	changes []plugin.Change
}

func plan(metadata metadata, observed api.Observation) (desired, error) {
	var result desired
	var currentSoftware software.UpdateMutation
	var currentPackage packages.PackageMutation
	if observed.Software != nil {
		currentSoftware = observed.Software.Mutation(observed.Software.Targets)
	}
	if observed.Package != nil {
		currentPackage = observed.Package.Mutation()
	}
	nextSoftware, err := metadata.software.Apply(currentSoftware)
	if err != nil {
		return result, err
	}
	nextSoftware.Normalize(metadata.name)
	result.targets = nextSoftware.Targets
	if err := nextSoftware.Validate(); err != nil {
		return result, fmt.Errorf("software: %w", err)
	}
	result.pkg, err = metadata.pkg.Apply(currentPackage)
	if err != nil {
		return result, err
	}
	result.pkg.Normalize()
	if err := result.pkg.ValidateMetadata(); err != nil {
		return result, fmt.Errorf("package: %w", err)
	}
	currentSoftware.Normalize(metadata.name)
	currentPackage.Normalize()
	if observed.Software == nil {
		initial := initialMetadata(nextSoftware, metadata.software.Bytes())
		initial["name"] = raw(metadata.name)
		result.changes = append(result.changes, plugin.Change{Kind: "metadata", Field: "software", Action: "create", After: raw(initial)})
	} else {
		result.changes = append(result.changes, diff("software", currentSoftware, nextSoftware)...)
	}
	if observed.Package == nil {
		result.changes = append(result.changes, plugin.Change{Kind: "metadata", Field: "package", Action: "create", After: raw(initialMetadata(result.pkg, metadata.pkg.Bytes()))})
	} else {
		result.changes = append(result.changes, diff("package", currentPackage, result.pkg)...)
	}
	if change, ok := iconChange(metadata.icon, observed); ok {
		result.icon = true
		result.changes = append(result.changes, change)
	}
	if result.pkg.InstallerType == packages.InstallerTypeNoPkg {
		return result, nil
	}
	artifact := metadata.installer
	if err := validateArtifact(artifact); err != nil {
		return result, err
	}
	var file *packages.InstallerFile
	if observed.Package != nil {
		file = observed.Package.InstallerFile
	}
	result.content = file == nil || file.SHA256 != artifact.SHA256 || file.SizeBytes != artifact.Size
	if result.content {
		change := plugin.Change{Kind: "content", Field: "package.installer", Action: "upload", After: raw(artifact.SHA256)}
		if file != nil {
			change.Before = raw(file.SHA256)
		}
		result.changes = append(result.changes, change)
	}
	return result, nil
}

// iconChange reports a declared icon whose bytes the software does not
// publish, naming the icon it replaces.
func iconChange(icon plugin.Artifact, observed api.Observation) (plugin.Change, bool) {
	if icon.Path == "" {
		return plugin.Change{}, false
	}
	var current *software.IconFile
	if observed.Software != nil && observed.Software.IconObjectID != nil {
		current = observed.Software.IconFile
	}
	if current != nil && current.SHA256 == icon.SHA256 && current.SizeBytes == icon.Size {
		return plugin.Change{}, false
	}
	change := plugin.Change{Kind: "content", Field: "software.icon", Action: "upload", After: raw(icon.SHA256)}
	if current != nil {
		change.Before = raw(current.SHA256)
	}
	return change, true
}

func initialMetadata(mutation any, supplied json.RawMessage) map[string]json.RawMessage {
	fields, _ := object(raw(mutation))
	declared, _ := object(supplied)
	// Restore declared fields that omitempty hides, using their normalized values.
	value := reflect.ValueOf(mutation)
	for i := range value.NumField() {
		key, _, _ := strings.Cut(value.Type().Field(i).Tag.Get("json"), ",")
		if _, ok := declared[key]; ok {
			fields[key] = raw(value.Field(i).Interface())
		}
	}
	delete(fields, "installer_object_id")
	delete(fields, "icon_object_id")
	return fields
}

// Both inputs are the same editable API type. Comparing field values retains
// explicit false and empty values that omitempty would hide in a whole mutation.
func diff(resource string, before, after any) []plugin.Change {
	var changes []plugin.Change
	old, next := reflect.ValueOf(before), reflect.ValueOf(after)
	for i := range next.NumField() {
		key, _, _ := strings.Cut(next.Type().Field(i).Tag.Get("json"), ",")
		if key == "installer_object_id" {
			continue
		}
		left, right := old.Field(i), next.Field(i)
		if left.Kind() == reflect.Slice && left.Len() == 0 && right.Len() == 0 {
			continue
		}
		previous, desired := raw(left.Interface()), raw(right.Interface())
		if bytes.Equal(previous, desired) {
			continue
		}
		kind := "metadata"
		if key == "targets" {
			kind = "targets"
		}
		changes = append(changes, plugin.Change{Kind: kind, Field: resource + "." + key, Action: "set", Before: previous, After: desired})
	}
	return changes
}

func metadataChanged(changes []plugin.Change, resource string) bool {
	return slices.ContainsFunc(changes, func(change plugin.Change) bool {
		return change.Kind != "content" && (change.Field == resource || strings.HasPrefix(change.Field, resource+"."))
	})
}
