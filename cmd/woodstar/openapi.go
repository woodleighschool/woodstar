package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodstar/internal/api"
	"github.com/woodleighschool/woodstar/internal/buildinfo"
)

func openAPICommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Print the OpenAPI document for the Woodstar admin API",
		Long: `Builds the same Huma admin API the server registers and writes its OpenAPI 3.1
document as YAML to stdout (or to the path given by --output). Handlers are
not invoked, so this command does not require a database.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload, err := api.BuildSchemaAPI(buildinfo.Version).OpenAPI().YAML()
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
