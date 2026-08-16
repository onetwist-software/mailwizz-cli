// Command mailwizz-cli-gen generates the MailWizz API client and CLI
// commands (internal/generated/**) from an OpenAPI schema. Run it via
// `task generate` rather than invoking it directly.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

func main() {
	schemaPath := flag.String("schema", "openapi/schema.json", "path to the OpenAPI schema")
	overridesPath := flag.String("overrides", "generator/overrides.yaml", "path to the generator overrides file")
	outDir := flag.String("out", "internal/generated", "output directory for generated packages")

	flag.Parse()

	if err := generator.Generate(*schemaPath, *overridesPath, *outDir); err != nil {
		// This is the final error-reporting step before the process
		// exits; if writing to stderr itself fails there is nowhere left
		// to report that failure to, so the write error is deliberately
		// discarded here rather than silently ignored elsewhere.
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
