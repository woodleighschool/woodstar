package pkginfo

import (
	"maps"
	"slices"

	"github.com/invopop/jsonschema"

	"github.com/woodleighschool/woodstar/internal/munki/packages"
)

// Schema describes the native pkginfo the importer accepts. The shared models
// describe the editable API fields, so their schema is translated here at the
// plugin boundary.
func Schema() *jsonschema.Schema {
	r := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	schema := r.Reflect(packages.PackageMutation{})
	schema.ID = ""
	for _, key := range []string{"installer_object_id", "blocking_applications_none"} {
		schema.Properties.Delete(key)
	}
	for _, property := range []struct{ name, description string }{
		{"name", "Stable Munki software identity shared by all versions."},
		{"display_name", "Software name shown to users in Managed Software Center."},
		{"description", "User-facing description of this software."},
		{"category", "Managed Software Center category."},
		{"developer", "Publisher or developer shown to users."},
	} {
		key, description := property.name, property.description
		field := &jsonschema.Schema{Type: "string"}
		if key != "name" {
			field = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{field, {Type: "null"}}}
		}
		field.Description = description
		schema.Properties.Set(key, field)
	}
	rename := func(target *jsonschema.Schema, from, to string) {
		if field, ok := target.Properties.Get(from); ok {
			target.Properties.Delete(from)
			target.Properties.Set(to, field)
		}
	}
	rename(schema, "restart_action", "RestartAction")
	rename(schema, "on_demand", "OnDemand")
	for field, names := range map[string]map[string]string{
		"receipts":              {"package_id": "packageid"},
		"installs":              {"bundle_identifier": "CFBundleIdentifier", "bundle_name": "CFBundleName", "bundle_short_version": "CFBundleShortVersionString", "bundle_version": "CFBundleVersion", "minimum_os_version": "minosversion"},
		"installer_choices_xml": {"choice_identifier": "choiceIdentifier", "choice_attribute": "choiceAttribute", "attribute_setting": "attributeSetting"},
	} {
		array, _ := schema.Properties.Get(field)
		for _, from := range slices.Sorted(maps.Keys(names)) {
			rename(array.Items, from, names[from])
		}
	}
	link := r.Reflect(ResourceRelationship{})
	link.ID = ""
	for _, field := range []string{"requires", "update_for"} {
		schema.Properties.Set(field, &jsonschema.Schema{Description: "Munki software names or references to other catalog resources.", Type: "array", Items: &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: "string"}, link}}})
	}
	schema.Properties.Set("installer_environment", &jsonschema.Schema{Description: "Environment variables supplied to the installer process.", Type: "object", AdditionalProperties: &jsonschema.Schema{Type: "string"}})
	for _, field := range []string{"preinstall_alert", "preuninstall_alert"} {
		alert, _ := schema.Properties.Get(field)
		alert.Properties.Delete("enabled")
		rename(alert, "title", "alert_title")
		rename(alert, "detail", "alert_detail")
		schema.Properties.Set(field, &jsonschema.Schema{AnyOf: []*jsonschema.Schema{alert, {Type: "null"}}})
	}
	for name, field := range schema.Properties.FromOldest() {
		if field.Type == "string" && name != "name" && name != "version" && name != "installer_type" {
			schema.Properties.Set(name, &jsonschema.Schema{AnyOf: []*jsonschema.Schema{field, {Type: "null"}}, Description: field.Description})
		}
	}
	return schema
}
