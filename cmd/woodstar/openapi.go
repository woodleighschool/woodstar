package main

import (
	"fmt"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"

	activityapi "github.com/woodleighschool/woodstar/internal/activity/httpapi"
	agentauthapi "github.com/woodleighschool/woodstar/internal/agentauth/httpapi"
	"github.com/woodleighschool/woodstar/internal/api"
	authapi "github.com/woodleighschool/woodstar/internal/auth/httpapi"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
	directoryapi "github.com/woodleighschool/woodstar/internal/directory/httpapi"
	hostsapi "github.com/woodleighschool/woodstar/internal/hosts/httpapi"
	inventoryapi "github.com/woodleighschool/woodstar/internal/inventory/httpapi"
	labelsapi "github.com/woodleighschool/woodstar/internal/labels/httpapi"
	munkiapi "github.com/woodleighschool/woodstar/internal/munki/httpapi"
	osqueryapi "github.com/woodleighschool/woodstar/internal/osquery/httpapi"
	santaapi "github.com/woodleighschool/woodstar/internal/santa/httpapi"
)

func openAPICommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Print the OpenAPI document for the Woodstar API",
		Long: `Builds the same Huma app API the server registers and writes its OpenAPI 3.1
document as YAML to stdout (or to the path given by --output). Handlers are
not invoked, so this command does not require a database.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload, err := buildOpenAPI(buildinfo.Version).OpenAPI().YAML()
			if err != nil {
				return fmt.Errorf("encode openapi: %w", err)
			}
			if len(payload) == 0 || payload[len(payload)-1] != '\n' {
				payload = append(payload, '\n')
			}

			if output == "" || output == "-" {
				_, err := os.Stdout.Write(payload)
				return err
			}
			return os.WriteFile(output, payload, 0o600)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "write OpenAPI YAML to this path (default stdout)")

	return cmd
}

func buildOpenAPI(version string) huma.API {
	schema, routes := api.NewSchema(version)
	authapi.RegisterOpenAPI(routes)
	activityapi.RegisterOpenAPI(routes)
	directoryapi.RegisterOpenAPI(routes)
	hostsapi.RegisterOpenAPI(routes)
	inventoryapi.RegisterOpenAPI(routes)
	labelsapi.RegisterOpenAPI(routes)
	agentauthapi.RegisterOpenAPI(routes)
	osqueryapi.RegisterOpenAPI(routes)
	munkiapi.RegisterOpenAPI(routes)
	santaapi.RegisterOpenAPI(routes)
	return schema
}
