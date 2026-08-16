package generator

import "strings"

// DefaultCommandName derives the CLI leaf command name for an operation
// from its HTTP method and path, for operations that don't have an
// explicit name in overrides.yaml. It covers the common CRUD shapes; any
// operation that doesn't fit (copy, pause-unpause, stats, tracking,
// search-by-*, and so on) is expected to be named explicitly in
// overrides.yaml.
func DefaultCommandName(method, path string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "DELETE":
		return "delete"
	case "GET":
		if pathEndsInParam(path) {
			return "view"
		}

		return "list"
	default:
		return strings.ToLower(method)
	}
}

// DefaultCommandPath derives the CLI command group for an operation from
// its path, keeping only the literal (non-parameter) path segments, e.g.
// "/lists/{list_uid}/subscribers/{subscriber_uid}" -> ["lists",
// "subscribers"]. Operations whose grouping doesn't match their literal
// path segments (for example "lists/fields/types", which belongs under
// "lists fields" rather than three levels deep) need an explicit
// CommandPath in overrides.yaml.
func DefaultCommandPath(path string) []string {
	var segments []string

	for s := range strings.SplitSeq(strings.Trim(path, "/"), "/") {
		if s == "" || isPathParamSegment(s) {
			continue
		}

		segments = append(segments, s)
	}

	return segments
}

func pathEndsInParam(path string) bool {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return false
	}

	segments := strings.Split(trimmed, "/")

	return isPathParamSegment(segments[len(segments)-1])
}

func isPathParamSegment(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}
