package generator_test

import (
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

func TestBuildTreeGroupsAndOrdersOperations(t *testing.T) {
	t.Parallel()

	ops := parsedAndOverriddenOps(t)

	tree, err := generator.BuildTree(ops)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	var lists *generator.Resource

	for _, r := range tree {
		if r.Name == "lists" {
			lists = r
		}
	}

	if lists == nil {
		t.Fatalf("expected a top-level 'lists' resource")
	}

	if len(lists.Operations) == 0 {
		t.Fatalf("expected lists to have direct operations (list/view/create/update/delete/copy)")
	}

	// The CRUD verbs should be ordered list, view, create, update, delete
	// ahead of anything else, so --help reads naturally.
	wantPrefix := []string{"list", "view", "create", "update", "delete"}
	for i, want := range wantPrefix {
		if lists.Operations[i].CommandName != want {
			t.Fatalf("lists.Operations[%d].CommandName = %q, want %q", i, lists.Operations[i].CommandName, want)
		}
	}

	var subscribers *generator.Resource

	for _, r := range lists.Children {
		if r.Name == "subscribers" {
			subscribers = r
		}
	}

	if subscribers == nil {
		t.Fatalf("expected a 'lists subscribers' resource")
	}

	names := map[string]bool{}
	for _, op := range subscribers.Operations {
		names[op.CommandName] = true
	}

	for _, want := range []string{"list", "create", "view", "update", "delete", "unsubscribe", "search-by-email"} {
		if !names[want] {
			t.Errorf("lists subscribers is missing command %q", want)
		}
	}
}

func TestBuildTreeRejectsDuplicateCommandNames(t *testing.T) {
	t.Parallel()

	ops := []*generator.Operation{
		{OperationID: "a", CommandPath: []string{"lists"}, CommandName: "create"},
		{OperationID: "b", CommandPath: []string{"lists"}, CommandName: "create"},
	}

	if _, err := generator.BuildTree(ops); err == nil {
		t.Fatal("expected an error for two operations sharing a command name in the same group")
	}
}

func TestBuildTreeAllowsSameCommandNameInDifferentGroups(t *testing.T) {
	t.Parallel()

	ops := []*generator.Operation{
		{OperationID: "a", CommandPath: []string{"lists"}, CommandName: "create"},
		{OperationID: "b", CommandPath: []string{"templates"}, CommandName: "create"},
	}

	tree, err := generator.BuildTree(ops)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	if len(tree) != 2 {
		t.Fatalf("got %d top-level resources, want 2", len(tree))
	}
}

func parsedAndOverriddenOps(t *testing.T) []*generator.Operation {
	t.Helper()

	ops := mustParseSchema(t)

	overrides, err := generator.LoadOverrides(overridesPath)
	if err != nil {
		t.Fatalf("LoadOverrides: %v", err)
	}

	if err := overrides.Apply(ops); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	return ops
}
