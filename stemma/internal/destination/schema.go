package destination

import (
	"github.com/invopop/jsonschema"
	"github.com/woodleighschool/woodstar/internal/munki/packages"
	"github.com/woodleighschool/woodstar/internal/munki/software"
)

func metadataSchema() *jsonschema.Schema {
	r := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	schema := r.Reflect(controls{})
	schema.ID = ""
	schema.Properties.Set("pkginfo", pkginfoSchema())
	targets := r.Reflect(software.Targets{})
	targets.ID = ""
	schema.Properties.Set("targets", targets)
	return schema
}

// The importer accepts native Munki names while the shared models describe the
// editable API fields. Keep their schema translation at the plugin boundary.
func pkginfoSchema() *jsonschema.Schema {
	r := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	schema := r.Reflect(packages.PackageMutation{})
	schema.ID = ""
	for _, key := range []string{"installer_object_id", "blocking_applications_none"} {
		schema.Properties.Delete(key)
	}
	for _, key := range []string{"name", "display_name", "description", "category", "developer"} {
		field := &jsonschema.Schema{Type: "string"}
		if key != "name" {
			field = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{field, {Type: "null"}}}
		}
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
		"installs":              {"bundle_identifier": "CFBundleIdentifier", "bundle_name": "CFBundleName", "bundle_short_version": "CFBundleShortVersionString", "bundle_version": "CFBundleVersion"},
		"installer_choices_xml": {"choice_identifier": "choiceIdentifier", "choice_attribute": "choiceAttribute", "attribute_setting": "attributeSetting"},
	} {
		array, _ := schema.Properties.Get(field)
		for from, to := range names {
			rename(array.Items, from, to)
		}
	}
	for _, field := range []string{"requires", "update_for"} {
		schema.Properties.Set(field, &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "string"}})
	}
	schema.Properties.Set("installer_environment", &jsonschema.Schema{Type: "object", AdditionalProperties: &jsonschema.Schema{Type: "string"}})
	for _, field := range []string{"preinstall_alert", "preuninstall_alert"} {
		alert, _ := schema.Properties.Get(field)
		alert.Properties.Delete("enabled")
		rename(alert, "title", "alert_title")
		rename(alert, "detail", "alert_detail")
		schema.Properties.Set(field, &jsonschema.Schema{AnyOf: []*jsonschema.Schema{alert, {Type: "null"}}})
	}
	for name, field := range schema.Properties.FromOldest() {
		if field.Type == "string" && name != "name" && name != "version" && name != "installer_type" {
			schema.Properties.Set(name, &jsonschema.Schema{AnyOf: []*jsonschema.Schema{field, {Type: "null"}}})
		}
	}
	return schema
}
