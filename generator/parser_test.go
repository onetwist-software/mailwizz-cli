package generator_test

import (
	"testing"

	"github.com/onetwist-software/mailwizz-cli/generator"
)

const schemaPath = "../openapi/schema.json"

func mustParseSchema(t *testing.T) []*generator.Operation {
	t.Helper()

	ops, err := generator.ParseSchema(schemaPath)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}

	return ops
}

func findOperation(t *testing.T, ops []*generator.Operation, operationID string) *generator.Operation {
	t.Helper()

	for _, op := range ops {
		if op.OperationID == operationID {
			return op
		}
	}

	t.Fatalf("operation %q not found", operationID)

	return nil
}

func TestParseSchemaFindsEveryOperation(t *testing.T) {
	t.Parallel()

	ops := mustParseSchema(t)

	// openapi/schema.json declares 39 paths; several have more than one
	// method, for a fixed total of 60 operations. A change to this number
	// should be a deliberate, reviewed schema update, not a silent parser
	// regression.
	const want = 60
	if len(ops) != want {
		t.Fatalf("got %d operations, want %d", len(ops), want)
	}
}

func TestParseSchemaFlattensBracketedRequestFields(t *testing.T) {
	t.Parallel()

	ops := mustParseSchema(t)
	op := findOperation(t, ops, "createList")

	fieldsByFlag := map[string]*generator.Field{}
	for _, f := range op.BodyFields {
		fieldsByFlag[f.FlagName] = f
	}

	tests := []struct {
		flagName string
		formKey  string
		goField  string
		required bool
	}{
		{flagName: "general-name", formKey: "general[name]", goField: "GeneralName", required: true},
		{flagName: "general-description", formKey: "general[description]", goField: "GeneralDescription", required: true},
		{flagName: "defaults-from-email", formKey: "defaults[from_email]", goField: "DefaultsFromEmail", required: true},
		{flagName: "defaults-subject", formKey: "defaults[subject]", goField: "DefaultsSubject", required: false},
		{flagName: "company-zip-code", formKey: "company[zip_code]", goField: "CompanyZipCode", required: false},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			t.Parallel()

			field, ok := fieldsByFlag[tt.flagName]
			if !ok {
				t.Fatalf("createList has no field with flag name %q", tt.flagName)
			}

			if field.FormKey != tt.formKey {
				t.Errorf("FormKey = %q, want %q", field.FormKey, tt.formKey)
			}

			if field.GoField != tt.goField {
				t.Errorf("GoField = %q, want %q", field.GoField, tt.goField)
			}

			if field.Required != tt.required {
				t.Errorf("Required = %v, want %v", field.Required, tt.required)
			}
		})
	}
}

func TestParseSchemaMergesRequiredAcrossAllOf(t *testing.T) {
	t.Parallel()

	// ListsRequest declares "required" for fields whose "properties" entry
	// actually lives on BaseListsRequest via allOf; createList must still
	// see them as required.
	ops := mustParseSchema(t)
	op := findOperation(t, ops, "createList")

	var found bool

	for _, f := range op.BodyFields {
		if f.FlagName == "general-name" {
			found = true

			if !f.Required {
				t.Errorf("general-name should be required")
			}
		}
	}

	if !found {
		t.Fatalf("general-name field not found")
	}

	// ListsUpdateRequest has no top-level "required", and updateList must
	// not inherit createList's required fields.
	updateOp := findOperation(t, ops, "updateList")
	for _, f := range updateOp.BodyFields {
		if f.Required {
			t.Errorf("updateList field %q should not be required", f.FlagName)
		}
	}
}

func TestParseSchemaCapturesEnumsAndParams(t *testing.T) {
	t.Parallel()

	ops := mustParseSchema(t)
	op := findOperation(t, ops, "viewListSubscribers")

	if len(op.PathParams) != 1 || op.PathParams[0].FlagName != "list-uid" {
		t.Fatalf("PathParams = %+v, want a single list-uid param", op.PathParams)
	}

	if !op.PathParams[0].Required {
		t.Errorf("list-uid path param should be required")
	}

	var statusParam *generator.Param

	for _, p := range op.QueryParams {
		if p.Name == "status" {
			statusParam = p
		}
	}

	if statusParam == nil {
		t.Fatalf("expected a status query param")
	}

	wantEnum := []string{"unconfirmed", "confirmed", "blacklisted", "unsubscribed", "unapproved", "disabled", "moved"}
	if len(statusParam.Enum) != len(wantEnum) {
		t.Fatalf("status enum = %v, want %v", statusParam.Enum, wantEnum)
	}
}

func TestParseSchemaHandlesCustomFieldFormKeysWithoutBrackets(t *testing.T) {
	t.Parallel()

	ops := mustParseSchema(t)
	op := findOperation(t, ops, "createListSubscriber")

	var email *generator.Field

	for _, f := range op.BodyFields {
		if f.FormKey == "EMAIL" {
			email = f
		}
	}

	if email == nil {
		t.Fatalf("expected an EMAIL field")
	}

	if email.FlagName != "email" || email.GoField != "Email" || !email.Required {
		t.Errorf("EMAIL field = %+v, want flag=email go=Email required=true", email)
	}
}

func TestParseSchemaRejectsMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := generator.ParseSchema("does-not-exist.json"); err == nil {
		t.Fatal("expected an error for a missing schema file")
	}
}
