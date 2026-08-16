package generator

import "strings"

// initialisms lists abbreviations that should be rendered fully
// upper-cased in generated Go identifiers, following the same convention
// as the standard library and common Go style guides (e.g. "ID", not
// "Id"). It is a read-only lookup table, not mutable shared state.
var initialisms = map[string]string{ //nolint:gochecknoglobals
	"id":    "ID",
	"uid":   "UID",
	"url":   "URL",
	"uri":   "URI",
	"api":   "API",
	"ip":    "IP",
	"html":  "HTML",
	"http":  "HTTP",
	"https": "HTTPS",
	"json":  "JSON",
	"xml":   "XML",
}

// kebabSegments splits an OpenAPI identifier (a path/query parameter name
// such as "list_uid", or a bracketed form field such as "general[name]")
// into lower-case word segments, ready to be joined into a CLI flag name
// or a Go identifier.
func kebabSegments(name string) []string {
	segments := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '[' || r == ']' || r == '-'
	})

	for i, s := range segments {
		segments[i] = strings.ToLower(s)
	}

	return segments
}

// FlagName converts an OpenAPI parameter name or bracketed form field key
// into a kebab-case CLI flag name, e.g. "list_uid" -> "list-uid" and
// "general[name]" -> "general-name".
func FlagName(name string) string {
	return strings.Join(kebabSegments(name), "-")
}

// GoFieldName converts an OpenAPI parameter name or bracketed form field
// key into an exported Go identifier, e.g. "list_uid" -> "ListUID" and
// "general[name]" -> "GeneralName".
func GoFieldName(name string) string {
	segments := kebabSegments(name)

	var identifier codeWriter

	for _, segment := range segments {
		if up, ok := initialisms[segment]; ok {
			identifier.writeString(up)

			continue
		}

		identifier.writeString(strings.ToUpper(segment[:1]))
		identifier.writeString(segment[1:])
	}

	return identifier.String()
}

// GoOperationName converts an OpenAPI operationId into an exported Go
// identifier, e.g. "viewListSubscriber" -> "ViewListSubscriber". Unlike
// GoFieldName it does not re-split the identifier into words, since
// operationIds are already valid camelCase Go-ish identifiers.
func GoOperationName(operationID string) string {
	if operationID == "" {
		return ""
	}

	return strings.ToUpper(operationID[:1]) + operationID[1:]
}
