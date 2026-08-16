package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
)

// generatedDirPerm and generatedFilePerm control the permissions of the
// directories/files this package writes; generated source is not
// sensitive, so these simply match common defaults for a checked-in Go
// package.
const (
	generatedDirPerm  = 0o755
	generatedFilePerm = 0o644
)

// Generate parses the OpenAPI schema at schemaPath, applies the
// overrides at overridesPath, and writes the generated API client and
// CLI command packages under outDir (internal/generated), creating
// outDir/api and outDir/cli.
func Generate(schemaPath, overridesPath, outDir string) error {
	operations, err := ParseSchema(schemaPath)
	if err != nil {
		return err
	}

	overrides, err := LoadOverrides(overridesPath)
	if err != nil {
		return err
	}

	if err := overrides.Apply(operations); err != nil {
		return err
	}

	tree, err := BuildTree(operations)
	if err != nil {
		return err
	}

	type generatedFile struct {
		path   string
		source string
	}

	files := []generatedFile{
		{filepath.Join(outDir, "api", "client_gen.go"), RenderAPIClient()},
		{filepath.Join(outDir, "api", "operations_gen.go"), RenderAPIOperations(operations)},
		{filepath.Join(outDir, "cli", "commands_gen.go"), RenderCLICommands(tree)},
	}

	for _, file := range files {
		if err := writeFormatted(file.path, file.source); err != nil {
			return err
		}
	}

	return nil
}

func writeFormatted(path, source string) error {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), generatedDirPerm); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	if err := os.WriteFile(path, formatted, generatedFilePerm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
