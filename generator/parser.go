package generator

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// methodOrder fixes the order in which operations for the same path are
// emitted, so generated output is stable across runs regardless of map
// iteration order.
var methodOrder = []string{ //nolint:gochecknoglobals // read-only lookup table
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodPatch,
}

// ErrMissingOperationID is returned when an operation in the schema has no
// operationId; the generator relies on it to derive Go and CLI names.
var ErrMissingOperationID = errors.New("operation has no operationId")

// ParseSchema loads the OpenAPI document at schemaPath and returns one
// Operation per path/method pair, in a stable order, with CommandName and
// CommandPath set to the generator's default guesses (see
// DefaultCommandName and DefaultCommandPath). Callers should apply
// generator/overrides.yaml next, before building the Resource tree.
func ParseSchema(schemaPath string) ([]*Operation, error) {
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("load schema %s: %w", schemaPath, err)
	}

	if doc.Paths == nil {
		return nil, nil
	}

	paths := doc.Paths.Map()

	pathKeys := make([]string, 0, len(paths))
	for path := range paths {
		pathKeys = append(pathKeys, path)
	}

	sort.Strings(pathKeys)

	var operations []*Operation

	for _, path := range pathKeys {
		item := paths[path]

		for _, method := range methodOrder {
			rawOperation := item.GetOperation(method)
			if rawOperation == nil {
				continue
			}

			built, err := buildOperation(method, path, rawOperation)
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", method, path, err)
			}

			operations = append(operations, built)
		}
	}

	return operations, nil
}

func buildOperation(method, path string, rawOperation *openapi3.Operation) (*Operation, error) {
	if rawOperation.OperationID == "" {
		return nil, fmt.Errorf("%s %s: %w", method, path, ErrMissingOperationID)
	}

	// A couple of paths in schema.json are missing their leading slash
	// (e.g. "lists/fields/types"); normalize so every generated request
	// path is well-formed once appended to the API base URL.
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	built := &Operation{
		OperationID: rawOperation.OperationID,
		GoName:      GoOperationName(rawOperation.OperationID),
		Method:      method,
		Path:        path,
		CommandName: DefaultCommandName(method, path),
		CommandPath: DefaultCommandPath(path),
		Summary:     rawOperation.Summary,
	}

	for _, paramRef := range rawOperation.Parameters {
		if paramRef.Value == nil {
			continue
		}

		param := buildParam(paramRef.Value)

		switch paramRef.Value.In {
		case openapi3.ParameterInPath:
			built.PathParams = append(built.PathParams, param)
		case openapi3.ParameterInQuery:
			built.QueryParams = append(built.QueryParams, param)
		}
	}

	if rawOperation.RequestBody != nil && rawOperation.RequestBody.Value != nil {
		fields := buildBodyFields(rawOperation.RequestBody.Value.Content)
		built.BodyFields = fields
		built.HasBody = len(fields) > 0
	}

	return built, nil
}

func buildParam(rawParam *openapi3.Parameter) *Param {
	return &Param{
		Name:        rawParam.Name,
		FlagName:    FlagName(rawParam.Name),
		GoField:     GoFieldName(rawParam.Name),
		Description: rawParam.Description,
		Required:    rawParam.Required,
		Enum:        schemaEnum(rawParam.Schema),
	}
}

// buildBodyFields flattens a multipart/form-data request schema into a
// stable, sorted list of Fields.
func buildBodyFields(content openapi3.Content) []*Field {
	media := content.Get("multipart/form-data")
	if media == nil {
		// Fall back to whatever single content type is declared, in case a
		// future schema update stops using multipart/form-data.
		for _, m := range content {
			media = m

			break
		}
	}

	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		return nil
	}

	fieldsByKey := map[string]*Field{}
	collectSchemaFields(media.Schema.Value, fieldsByKey)

	keys := make([]string, 0, len(fieldsByKey))
	for k := range fieldsByKey {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	fields := make([]*Field, 0, len(keys))
	for _, k := range keys {
		fields = append(fields, fieldsByKey[k])
	}

	return fields
}

// collectSchemaFields recursively walks schema.AllOf and merges every
// referenced schema's properties into fieldsByKey (keyed by the original
// form field name), then applies this schema's own properties and
// required list. Applying "required" last, against the full merged set,
// is what lets a schema such as ListsRequest declare "required" for
// fields whose "properties" entry actually lives on BaseListsRequest via
// allOf.
func collectSchemaFields(schema *openapi3.Schema, fieldsByKey map[string]*Field) {
	for _, sub := range schema.AllOf {
		if sub.Value != nil {
			collectSchemaFields(sub.Value, fieldsByKey)
		}
	}

	for key, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}

		field, ok := fieldsByKey[key]
		if !ok {
			field = &Field{
				FormKey:  key,
				FlagName: FlagName(key),
				GoField:  GoFieldName(key),
			}
			fieldsByKey[key] = field
		}

		field.Description = firstNonEmpty(propRef.Value.Description, propRef.Value.Title)
		field.Enum = schemaEnum(propRef)
	}

	for _, name := range schema.Required {
		if field, ok := fieldsByKey[name]; ok {
			field.Required = true
		}
	}
}

func schemaEnum(ref *openapi3.SchemaRef) []string {
	if ref == nil || ref.Value == nil {
		return nil
	}

	values := make([]string, 0, len(ref.Value.Enum))
	seen := make(map[string]bool, len(ref.Value.Enum))

	for _, v := range ref.Value.Enum {
		value, ok := v.(string)
		if !ok || seen[value] {
			// schema.json has at least one enum with a literal duplicate
			// value (campaign[options][autoresponder_event]); skip repeats
			// so generated code (e.g. a switch statement) stays valid.
			continue
		}

		seen[value] = true

		values = append(values, value)
	}

	return values
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
