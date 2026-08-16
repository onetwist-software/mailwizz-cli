package generator

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// ErrUnknownOperation is returned when overrides.yaml references an
// operationId that does not exist in the parsed schema.
var ErrUnknownOperation = errors.New("operation does not exist in the schema")

// ErrUnknownQueryParam is returned when overrides.yaml references a query
// parameter definition that does not exist.
var ErrUnknownQueryParam = errors.New("query param is not defined in query_param_definitions")

// Overrides holds generator/overrides.yaml: hand-maintained corrections
// applied on top of the generator's default naming and parameter
// inference. See overrides.yaml for why each entry exists.
type Overrides struct {
	QueryParamDefinitions map[string]QueryParamDefinition `yaml:"query_param_definitions"`
	Operations            map[string]OperationOverride    `yaml:"operations"`
}

// QueryParamDefinition describes a query parameter that Apply can attach
// to an operation via OperationOverride.AddQueryParams.
type QueryParamDefinition struct {
	Description string `yaml:"description"`
}

// OperationOverride corrects one operation, looked up by its operationId.
type OperationOverride struct {
	// CommandPath replaces the operation's default command group.
	CommandPath []string `yaml:"command_path"`
	// CommandName replaces the operation's default leaf command name.
	CommandName string `yaml:"command_name"`
	// AddQueryParams names entries in QueryParamDefinitions to attach to
	// this operation as additional query parameters.
	AddQueryParams []string `yaml:"add_query_params"`
}

// LoadOverrides reads and parses an overrides file.
func LoadOverrides(path string) (*Overrides, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a caller-supplied generator input, not untrusted request data
	if err != nil {
		return nil, fmt.Errorf("read overrides file %s: %w", path, err)
	}

	var overrides Overrides
	if err := yaml.Unmarshal(data, &overrides); err != nil {
		return nil, fmt.Errorf("parse overrides file %s: %w", path, err)
	}

	return &overrides, nil
}

// Apply folds the overrides into operations in place. It returns an error
// if an override references an operationId or query parameter definition
// that does not exist, so a schema change that removes or renames an
// operation is caught immediately instead of silently going stale.
func (o *Overrides) Apply(operations []*Operation) error {
	byID := make(map[string]*Operation, len(operations))
	for _, op := range operations {
		byID[op.OperationID] = op
	}

	ids := make([]string, 0, len(o.Operations))
	for id := range o.Operations {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, operationID := range ids {
		override := o.Operations[operationID]

		operation, ok := byID[operationID]
		if !ok {
			return fmt.Errorf("overrides.yaml: %s: %w", operationID, ErrUnknownOperation)
		}

		if len(override.CommandPath) > 0 {
			operation.CommandPath = override.CommandPath
		}

		if override.CommandName != "" {
			operation.CommandName = override.CommandName
		}

		if err := o.addQueryParams(operation, operationID, override.AddQueryParams); err != nil {
			return err
		}
	}

	return nil
}

func (o *Overrides) addQueryParams(operation *Operation, operationID string, names []string) error {
	for _, name := range names {
		def, ok := o.QueryParamDefinitions[name]
		if !ok {
			return fmt.Errorf("overrides.yaml: %s references %s: %w", operationID, name, ErrUnknownQueryParam)
		}

		operation.QueryParams = append(operation.QueryParams, &Param{
			Name:        name,
			FlagName:    FlagName(name),
			GoField:     GoFieldName(name),
			Description: def.Description,
		})
	}

	return nil
}
