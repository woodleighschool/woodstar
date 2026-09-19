package destination

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/woodleighschool/woodstar/internal/munki/software"
	"github.com/woodleighschool/woodstar/internal/targeting"
	"github.com/woodleighschool/woodstar/stemma/internal/destination/api"
)

// targets is the declared form of label targeting: labels by name, with every
// include following the latest package.
type targets struct {
	Include []include `json:"include,omitempty" jsonschema_description:"Labels whose hosts receive these Munki actions. Each assignment follows the latest package."`
	Exclude []exclude `json:"exclude,omitempty" jsonschema_description:"Labels whose hosts are excluded from this software. An empty list removes all exclusions."`
}

type include struct {
	LabelName string            `json:"label_name" jsonschema:"required,minLength=1,description=Exact name of the label whose hosts receive the actions."`
	Actions   []software.Action `json:"actions"    jsonschema:"required,minItems=1,description=Munki manifest sections the software joins for those hosts."`
}

type exclude struct {
	LabelName string `json:"label_name" jsonschema:"required,minLength=1,description=Exact name of a label whose hosts are left out."`
}

var actions = []software.Action{
	software.ActionManagedInstalls, software.ActionManagedUninstalls, software.ActionManagedUpdates,
	software.ActionOptionalInstalls, software.ActionFeaturedItems, software.ActionDefaultInstalls,
}

// decodeTargets validates declared targeting without contacting the repository.
// Omitted targets are left unchanged; declared empty lists clear them.
func decodeTargets(data json.RawMessage) (*targets, error) {
	if len(data) == 0 {
		return nil, nil
	}
	fields, err := object(data)
	if err != nil {
		return nil, fmt.Errorf("targets: %w", err)
	}
	for name, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("targets.%s must be a list", name)
		}
	}
	var declared targets
	if err := decode(data, &declared); err != nil {
		return nil, fmt.Errorf("targets: %w", err)
	}
	for _, entry := range declared.Include {
		if strings.TrimSpace(entry.LabelName) == "" || len(entry.Actions) == 0 {
			return nil, errors.New("targets.include entries require label_name and actions")
		}
		for _, action := range entry.Actions {
			if !slices.Contains(actions, action) {
				return nil, fmt.Errorf("targets: unsupported action %q", action)
			}
		}
	}
	for _, entry := range declared.Exclude {
		if strings.TrimSpace(entry.LabelName) == "" {
			return nil, errors.New("targets.exclude entries require label_name")
		}
	}
	return &declared, nil
}

// resolveTargets replaces declared lists, resolving labels to the instance's ids.
func resolveTargets(ctx context.Context, remote *api.Client, metadata *metadata) error {
	declared := metadata.controls.targets
	if declared == nil {
		return nil
	}
	var names []string
	for _, entry := range declared.Include {
		names = append(names, entry.LabelName)
	}
	for _, entry := range declared.Exclude {
		names = append(names, entry.LabelName)
	}
	ids, err := remote.LabelIDs(ctx, names)
	if err != nil {
		return fmt.Errorf("targets: %w", err)
	}
	native := map[string]any{}
	if declared.Include != nil {
		entries := make([]software.Include, 0, len(declared.Include))
		for _, entry := range declared.Include {
			entries = append(entries, software.Include{LabelID: ids[entry.LabelName], Package: software.PackageSelector{Strategy: software.PackageLatest}, Actions: entry.Actions})
		}
		native["include"] = entries
	}
	if declared.Exclude != nil {
		entries := make([]targeting.LabelRef, 0, len(declared.Exclude))
		for _, entry := range declared.Exclude {
			entries = append(entries, targeting.LabelRef{LabelID: ids[entry.LabelName]})
		}
		native["exclude"] = entries
	}
	fields, err := object(metadata.software.Bytes())
	if err != nil {
		return err
	}
	fields["targets"] = raw(native)
	if err := json.Unmarshal(raw(fields), &metadata.software); err != nil {
		return fmt.Errorf("targets: %w", err)
	}
	return nil
}
