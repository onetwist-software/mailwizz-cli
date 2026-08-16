// Package generator turns MailWizz's OpenAPI schema into the internal
// model (IR) used to render the generated API client and CLI commands.
//
// The pipeline is: Parse (schema.json -> Doc via naming.go's helpers and
// parser.go) -> apply overrides.yaml -> render templates. This file
// defines the IR types and the pure naming/flattening rules that turn
// OpenAPI identifiers into CLI flag names and Go identifiers.
package generator

// Doc is the fully built, override-applied model for one OpenAPI schema.
// It is what the templates render from.
type Doc struct {
	// Resources are the top-level CLI command groups, e.g. "lists",
	// "campaigns", "countries".
	Resources []*Resource
}

// Resource is a CLI command group. A Resource with a non-empty Operations
// list and no Children is a leaf command group (e.g. "templates"); a
// Resource with Children nests further, e.g. "lists" has a child resource
// for "subscribers" reached as `mailwizz-cli lists subscribers ...`.
type Resource struct {
	// Name is this resource's own command name, e.g. "subscribers".
	Name string
	// Path is the full command path from the root to this resource, e.g.
	// ["lists", "subscribers"]. It is used to build a unique Go function
	// name for the generated command constructor.
	Path []string
	// Usage is a short, human-readable description shown in --help.
	Usage string
	// Operations are the leaf commands directly under this resource.
	Operations []*Operation
	// Children are nested resources, e.g. lists -> fields/segments/subscribers.
	Children []*Resource
}

// Operation describes a single OpenAPI operation (one HTTP method on one
// path) and how it is exposed as a CLI leaf command.
type Operation struct {
	// OperationID is the OpenAPI operationId, e.g. "createListSubscriber".
	OperationID string
	// GoName is the exported Go identifier derived from OperationID, used
	// for the generated client method and request/response type names.
	GoName string
	// Method is the HTTP method, e.g. "GET".
	Method string
	// Path is the original OpenAPI path template, e.g.
	// "/lists/{list_uid}/subscribers/{subscriber_uid}".
	Path string
	// CommandName is this operation's leaf CLI command name, e.g. "create",
	// "list", "view", "pause-unpause".
	CommandName string
	// CommandPath is the command group this operation is nested under,
	// e.g. ["lists", "subscribers"]. It is only used while building the
	// Resource tree (see BuildTree) and is not otherwise needed once an
	// Operation is attached to its Resource.
	CommandPath []string
	// Summary is the short human-readable description of the operation,
	// taken from the schema (or an override).
	Summary string

	// PathParams are the path parameters (in URL path order).
	PathParams []*Param
	// QueryParams are the query string parameters.
	QueryParams []*Param
	// BodyFields are the flattened multipart/form-data request body
	// fields; empty for operations with no request body.
	BodyFields []*Field

	// HasBody is true for operations that send a request body (POST/PUT
	// with a defined request schema).
	HasBody bool
}

// Param is a path or query parameter.
type Param struct {
	// Name is the original OpenAPI parameter name, e.g. "list_uid".
	Name string
	// FlagName is the CLI flag name, e.g. "list-uid".
	FlagName string
	// GoField is the Go identifier for this parameter, e.g. "ListUID".
	GoField string
	// Description is shown as the flag's usage text.
	Description string
	// Required marks the flag as mandatory.
	Required bool
	// Enum lists the allowed values, if the schema constrains them.
	Enum []string
}

// Field is a single flattened request body field. MailWizz's request
// schemas use PHP-style bracketed keys for nested form data, e.g.
// "general[name]" or "details[status]"; FormKey preserves that original
// key so the generated client can still send exactly what the API
// expects, while FlagName/GoField give it a normal CLI flag and Go
// identifier.
type Field struct {
	// FormKey is the original multipart/form-data field name, e.g.
	// "general[name]".
	FormKey string
	// FlagName is the CLI flag name, e.g. "general-name".
	FlagName string
	// GoField is the Go identifier for this field, e.g. "GeneralName".
	GoField string
	// Description is shown as the flag's usage text.
	Description string
	// Required marks the flag as mandatory.
	Required bool
	// Enum lists the allowed values, if the schema constrains them.
	Enum []string
}
