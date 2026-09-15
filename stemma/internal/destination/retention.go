package destination

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/woodleighschool/stemma/plugin"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

// apiFields maps the native pkginfo names that differ from the package API.
var apiFields = map[string]string{"RestartAction": "restart_action", "OnDemand": "on_demand"}

func (remote *client) manageDerived(metadata *metadata) error {
	fields, err := object(metadata.pkg.Bytes())
	if err != nil {
		return err
	}
	for _, field := range remote.state.Packages[remote.fingerprint].Derived {
		native := field
		for name, api := range apiFields {
			if api == field {
				native = name
			}
		}
		if _, exists := fields[field]; exists || slices.Contains(metadata.controls.Unmanaged, "pkginfo."+native) {
			continue
		}
		fields[field] = released(field)
	}
	return json.Unmarshal(raw(fields), &metadata.pkg)
}

// released clears a previously derived field. Merge patches clear only optional
// strings and pointers with null; other fields return to their zero value.
func released(field string) json.RawMessage {
	for column := range reflect.TypeFor[packages.PackageMutation]().Fields() {
		if name, _, _ := strings.Cut(column.Tag.Get("json"), ","); name != field {
			continue
		}
		kind := column.Type.Kind()
		if kind == reflect.Slice {
			return raw([]any{})
		}
		if kind == reflect.Bool || kind == reflect.Int64 {
			return raw(reflect.Zero(column.Type).Interface())
		}
		break
	}
	return raw(nil)
}

func (remote *client) recordPackage(metadata metadata, pkg *packages.Package) {
	if pkg == nil {
		return
	}
	derived := []string{}
	fields, _ := object(metadata.pkg.Bytes())
	for field, origin := range metadata.origins {
		key := strings.TrimPrefix(field, "pkginfo.")
		if api, renamed := apiFields[key]; renamed {
			key = api
		}
		if origin != "authored" {
			if _, present := fields[key]; present {
				derived = append(derived, key)
			}
		}
	}
	slices.Sort(derived)
	remote.state.Packages[remote.fingerprint] = publication{ID: pkg.ID, Version: pkg.Version, SHA256: metadata.installer.SHA256, Derived: derived}
	remote.state.PackageID, remote.state.Version = pkg.ID, pkg.Version
}

// Only durable owned IDs with successful publication order can be pruned. Native
// foreign keys protect reverse references that the scoped API cannot enumerate.
func (remote *client) prune(ctx context.Context, retention *plugin.Retention, observed observation, apply bool) ([]plugin.Change, error) {
	if retention == nil || observed.Software == nil {
		return nil, nil
	}
	history := remote.state.Publications
	if !apply {
		// Record on a copy so planning never commits a publication or changes ownership.
		history.Order = maps.Clone(remote.state.Publications.Order)
		history.Record(remote.fingerprint)
	}
	keep := history.Retained(retention.Keep)
	if len(keep) == 0 {
		return nil, nil
	}
	for key := range remote.state.Packages {
		if history.Order[key] == 0 {
			return nil, nil
		}
	}
	items, err := list[packages.Package](ctx, remote, "/api/munki/packages", url.Values{"software_id": {strconv.FormatInt(observed.Software.ID, 10)}})
	if err != nil {
		return nil, err
	}
	protected, byID := protectedPackages(observed.Software.Targets, items)
	keys := make([]string, 0, len(remote.state.Packages))
	for key := range remote.state.Packages {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var changes []plugin.Change
	for _, key := range keys {
		owned := remote.state.Packages[key]
		if keep[key] || protected[owned.ID] {
			continue
		}
		current, exists := byID[owned.ID]
		if !exists {
			continue
		}
		if !owned.matches(current, observed.Software.ID) {
			continue
		}
		if apply {
			err := remote.request(ctx, http.MethodDelete, "/api/munki/packages?ids="+strconv.FormatInt(owned.ID, 10), nil, nil)
			var status httpError
			if errors.As(err, &status) && status.status == http.StatusConflict {
				continue
			}
			if err != nil {
				return changes, err
			}
			delete(remote.state.Packages, key)
		}
		changes = append(changes, plugin.Change{Kind: "retention", Field: "package", Action: "delete", Before: raw(owned.ID)})
	}
	return changes, nil
}

func protectedPackages(targets software.Targets, items []packages.Package) (map[int64]bool, map[int64]packages.Package) {
	protected := map[int64]bool{}
	for _, include := range targets.Include {
		if include.Package.Strategy == software.PackageSpecific && include.Package.PackageID != nil {
			protected[*include.Package.PackageID] = true
		}
	}
	byID := make(map[int64]packages.Package, len(items))
	for _, item := range items {
		byID[item.ID] = item
		for _, ref := range append(slices.Clone(item.Requires), item.UpdateFor...) {
			if ref.PackageID > 0 {
				protected[ref.PackageID] = true
			}
		}
	}
	return protected, byID
}
