package generator

import (
	"regexp"
	"strconv"
	"strings"
)

// RequestField is a unified view over one path parameter, query
// parameter, or request body field, for rendering: all three ultimately
// become one string field on a generated Request struct and one CLI
// flag.
type RequestField struct {
	// GoField is the Request struct field name, e.g. "ListUID".
	GoField string
	// FlagName is the CLI flag name, e.g. "list-uid".
	FlagName string
	// Description is the flag's usage text.
	Description string
	// Required marks the field/flag as mandatory.
	Required bool
	// Enum lists the allowed values, if the schema constrains them.
	Enum []string
}

// RequestFields returns every path param, query param, and body field of
// op as a single ordered list: path params first (in path order), then
// query params, then body fields. Both the API and CLI generators use
// this so a Request struct's fields and a command's flags can never
// drift apart.
func (op *Operation) RequestFields() []RequestField {
	fields := make([]RequestField, 0, len(op.PathParams)+len(op.QueryParams)+len(op.BodyFields))

	for _, p := range op.PathParams {
		fields = append(fields, requestFieldFromParam(p))
	}

	for _, p := range op.QueryParams {
		fields = append(fields, requestFieldFromParam(p))
	}

	for _, field := range op.BodyFields {
		fields = append(fields, RequestField{
			GoField:     field.GoField,
			FlagName:    field.FlagName,
			Description: field.Description,
			Required:    field.Required,
			Enum:        field.Enum,
		})
	}

	return fields
}

func requestFieldFromParam(param *Param) RequestField {
	return RequestField{
		GoField:     param.GoField,
		FlagName:    param.FlagName,
		Description: param.Description,
		Required:    param.Required,
		Enum:        param.Enum,
	}
}

// pathParamPattern matches an OpenAPI path placeholder such as
// "{list_uid}".
var pathParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// PathExpr returns a Go expression, as source text, that builds this
// operation's request path from a "req" variable of the generated
// Request type, substituting every {param} in Path with
// url.PathEscape(req.<GoField>). For an operation with no path
// parameters, it returns a plain quoted string.
//
// For example, "/lists/{list_uid}/subscribers/{subscriber_uid}" becomes:
//
//	"/lists/" + url.PathEscape(req.ListUID) + "/subscribers/" + url.PathEscape(req.SubscriberUID)
func (op *Operation) PathExpr() string {
	if len(op.PathParams) == 0 {
		return strconv.Quote(op.Path)
	}

	byName := make(map[string]*Param, len(op.PathParams))
	for _, p := range op.PathParams {
		byName[p.Name] = p
	}

	var expr codeWriter

	remaining := op.Path
	for {
		loc := pathParamPattern.FindStringIndex(remaining)
		if loc == nil {
			break
		}

		if literal := remaining[:loc[0]]; literal != "" {
			expr.writeString(strconv.Quote(literal))
			expr.writeString(" + ")
		}

		name := remaining[loc[0]+1 : loc[1]-1]

		goField := GoFieldName(name)
		if p, ok := byName[name]; ok {
			goField = p.GoField
		}

		expr.writeString("url.PathEscape(req.")
		expr.writeString(goField)
		expr.writeString(")")

		remaining = remaining[loc[1]:]

		if remaining != "" {
			expr.writeString(" + ")
		}
	}

	if remaining != "" {
		expr.writeString(strconv.Quote(remaining))
	}

	return expr.String()
}

// FuncName returns the unique, unexported Go function name used for the
// generated command constructor of this operation's leaf command, e.g.
// "listsSubscribersCreateCommand".
func (op *Operation) FuncName() string {
	return funcName(append(append([]string{}, op.CommandPath...), op.CommandName))
}

// FuncName returns the unique, unexported Go function name used for the
// generated command constructor of this resource group, e.g.
// "listsSubscribersCommand".
func (r *Resource) FuncName() string {
	return funcName(r.Path)
}

func funcName(segments []string) string {
	var name codeWriter
	for _, s := range segments {
		name.writeString(GoFieldName(s))
	}

	built := name.String()
	if built == "" {
		return "Command"
	}

	return strings.ToLower(built[:1]) + built[1:] + "Command"
}
