package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

// updateGoldenEnvVar lets a developer refresh the golden files after an
// intentional generator change: UPDATE_GOLDEN=1 go test ./generator/...
const updateGoldenEnvVar = "UPDATE_GOLDEN"

// goldenFiles lists every file Generate produces, relative to its outDir
// argument and to generator/testdata/golden.
var goldenFiles = []string{ //nolint:gochecknoglobals // read-only test fixture data
	filepath.Join("api", "client_gen.go"),
	filepath.Join("api", "operations_gen.go"),
	filepath.Join("cli", "commands_gen.go"),
}

func TestGenerateMatchesGoldenFiles(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()

	err := generator.Generate(
		filepath.Join("testdata", "fixture_schema.json"),
		filepath.Join("testdata", "fixture_overrides.yaml"),
		outDir,
	)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, rel := range goldenFiles {
		golden := filepath.Join("testdata", "golden", rel)

		got, err := os.ReadFile(filepath.Join(outDir, rel)) //nolint:gosec // fixed, non-attacker-controlled test path
		if err != nil {
			t.Fatalf("read generated %s: %v", rel, err)
		}

		if os.Getenv(updateGoldenEnvVar) != "" {
			if err := os.WriteFile(golden, got, 0o644); err != nil { //nolint:gosec // golden fixture, not sensitive
				t.Fatalf("update golden %s: %v", golden, err)
			}

			continue
		}

		want, err := os.ReadFile(golden) //nolint:gosec // fixed, non-attacker-controlled test path
		if err != nil {
			t.Fatalf("read golden %s: %v (run with UPDATE_GOLDEN=1 to create it)", golden, err)
		}

		if string(got) != string(want) {
			t.Errorf("%s does not match golden file %s; run with UPDATE_GOLDEN=1 to review and update it", rel, golden)
		}
	}
}

// TestGenerateRealSchemaMatchesCommittedOutput regenerates
// internal/generated/** from the real openapi/schema.json and
// generator/overrides.yaml and confirms it is byte-identical to what is
// committed in the repository. This is the same check `task
// generate:check` runs in CI; failing here means someone edited
// internal/generated/** by hand, or forgot to regenerate after changing
// openapi/schema.json or generator/overrides.yaml.
func TestGenerateRealSchemaMatchesCommittedOutput(t *testing.T) {
	t.Parallel()

	outDir := t.TempDir()

	err := generator.Generate(schemaPath, overridesPath, outDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	committedDir := filepath.Join("..", "internal", "generated")

	for _, rel := range goldenFiles {
		got, err := os.ReadFile(filepath.Join(outDir, rel)) //nolint:gosec // fixed, non-attacker-controlled test path
		if err != nil {
			t.Fatalf("read generated %s: %v", rel, err)
		}

		want, err := os.ReadFile(filepath.Join(committedDir, rel)) //nolint:gosec // fixed, non-attacker-controlled test path
		if err != nil {
			t.Fatalf("read committed %s: %v", rel, err)
		}

		if string(got) != string(want) {
			t.Errorf("committed internal/generated/%s is stale; run `task generate` and commit the result", rel)
		}
	}
}
