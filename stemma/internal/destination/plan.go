package destination

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/woodleighschool/stemma/plugin"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

type desired struct {
	targets software.Targets
	pkg     packages.PackageMutation
	content bool
	changes []plugin.Change
}

func (remote *client) plan(artifact plugin.Artifact, metadata metadata, observed observation) (desired, error) {
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
	nextSoftware.Normalize(remote.config.Name)
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
	currentSoftware.Normalize(remote.config.Name)
	currentPackage.Normalize()
	if observed.Software == nil {
		result.changes = append(result.changes, plugin.Change{Kind: "metadata", Field: "software.name", Action: "create", After: raw(remote.config.Name)})
	}
	if observed.Package == nil {
		result.changes = append(result.changes, plugin.Change{Kind: "metadata", Field: "package.version", Action: "create", After: raw(remote.config.Version)})
	}
	result.changes = append(result.changes, diff("software", currentSoftware, nextSoftware)...)
	result.changes = append(result.changes, diff("package", currentPackage, result.pkg)...)
	if result.pkg.InstallerType == packages.InstallerTypeNoPkg {
		return result, nil
	}
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

func validateArtifact(artifact plugin.Artifact) error {
	digest, err := hex.DecodeString(artifact.SHA256)
	if err != nil || len(digest) != 32 || strings.ToLower(artifact.SHA256) != artifact.SHA256 {
		return errors.New("installer artifact requires a lowercase SHA-256 digest")
	}
	if artifact.Filename == "" || strings.TrimSpace(artifact.Filename) != artifact.Filename || filepath.Base(artifact.Filename) != artifact.Filename || artifact.Size < 0 {
		return errors.New("installer artifact requires an unpadded base filename and nonnegative size")
	}
	return nil
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

func changed(changes []plugin.Change, resource string) bool {
	return slices.ContainsFunc(changes, func(change plugin.Change) bool { return strings.HasPrefix(change.Field, resource+".") })
}
