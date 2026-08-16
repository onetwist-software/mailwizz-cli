package generator_test

import (
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

const overridesPath = "overrides.yaml"

func TestApplyOverridesToRealSchema(t *testing.T) {
	t.Parallel()

	ops := mustParseSchema(t)

	overrides, err := generator.LoadOverrides(overridesPath)
	if err != nil {
		t.Fatalf("LoadOverrides: %v", err)
	}

	if err := overrides.Apply(ops); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	copyOp := findOperation(t, ops, "copyCampaign")
	if copyOp.CommandName != "copy" || len(copyOp.CommandPath) != 1 || copyOp.CommandPath[0] != "campaigns" {
		t.Errorf("copyCampaign command = %v %v, want [campaigns] copy", copyOp.CommandPath, copyOp.CommandName)
	}

	paginated := findOperation(t, ops, "viewLists")
	if len(paginated.QueryParams) != 2 {
		t.Fatalf("viewLists query params = %v, want page and per_page", paginated.QueryParams)
	}

	names := map[string]bool{}
	for _, p := range paginated.QueryParams {
		names[p.Name] = true
	}

	if !names["page"] || !names["per_page"] {
		t.Errorf("viewLists query params = %v, want page and per_page", paginated.QueryParams)
	}
}

func TestApplyOverridesRejectsUnknownOperation(t *testing.T) {
	t.Parallel()

	ops := []*generator.Operation{{OperationID: "realOperation"}}
	overrides := &generator.Overrides{
		Operations: map[string]generator.OperationOverride{
			"doesNotExist": {CommandName: "whatever"},
		},
	}

	if err := overrides.Apply(ops); err == nil {
		t.Fatal("expected an error for an override referencing an unknown operationId")
	}
}

func TestApplyOverridesRejectsUnknownQueryParam(t *testing.T) {
	t.Parallel()

	ops := []*generator.Operation{{OperationID: "realOperation"}}
	overrides := &generator.Overrides{
		Operations: map[string]generator.OperationOverride{
			"realOperation": {AddQueryParams: []string{"does-not-exist"}},
		},
	}

	if err := overrides.Apply(ops); err == nil {
		t.Fatal("expected an error for an override referencing an undefined query param")
	}
}

func TestApplyOverridesAddsQueryParamFromDefinition(t *testing.T) {
	t.Parallel()

	ops := []*generator.Operation{{OperationID: "realOperation"}}
	overrides := &generator.Overrides{
		QueryParamDefinitions: map[string]generator.QueryParamDefinition{
			"page": {Description: "Page number"},
		},
		Operations: map[string]generator.OperationOverride{
			"realOperation": {AddQueryParams: []string{"page"}},
		},
	}

	if err := overrides.Apply(ops); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(ops[0].QueryParams) != 1 {
		t.Fatalf("QueryParams = %v, want one entry", ops[0].QueryParams)
	}

	got := ops[0].QueryParams[0]
	if got.Name != "page" || got.FlagName != "page" || got.GoField != "Page" || got.Description != "Page number" {
		t.Errorf("QueryParams[0] = %+v, unexpected", got)
	}
}

func TestLoadOverridesRejectsMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := generator.LoadOverrides("does-not-exist.yaml"); err == nil {
		t.Fatal("expected an error for a missing overrides file")
	}
}
