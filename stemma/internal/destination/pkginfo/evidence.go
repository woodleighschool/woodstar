package pkginfo

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/woodleighschool/stemma/plugin"
)

// The preparation kind selects an application once; native publishers retain
// responsibility for translating that evidence to their own detection fields.
func macEvidence(artifact plugin.Artifact) (*plugin.Subject, string, error) {
	var app *plugin.Subject
	var versionKey string
	if data, ok := artifact.Evidence["macos.application"]; ok {
		if err := json.Unmarshal(data, &app); err != nil || app == nil || app.App == nil {
			return nil, "", errors.New("macos.application evidence requires an application subject")
		}
	}
	if data, ok := artifact.Evidence["macos.version_key"]; ok {
		if err := json.Unmarshal(data, &versionKey); err != nil || (versionKey != "CFBundleVersion" && versionKey != "CFBundleShortVersionString") {
			return nil, "", errors.New("unsupported macos.version_key evidence")
		}
	}

	return app, versionKey, nil
}

// selectApplication finds the application a derivation names, or without one
// the artifact's only application.
func selectApplication(facts plugin.Facts, selectors map[string]plugin.SubjectSelector, options *AppDerivation) (*plugin.Subject, error) {
	if options != nil {
		selector, exists := selectors[options.Subject]
		if !exists {
			return nil, fmt.Errorf("derive.app references unknown subject %q", options.Subject)
		}
		subject, err := plugin.SelectSubject(facts, selector)
		if err != nil {
			return nil, err
		}
		if subject.App == nil {
			return nil, errors.New("derive.app requires an application subject")
		}
		return &subject, nil
	}
	var selected *plugin.Subject
	for _, subject := range facts.Subjects {
		if subject.App == nil {
			continue
		}
		if selected != nil {
			return nil, nil
		}
		selected = &subject
	}
	return selected, nil
}
