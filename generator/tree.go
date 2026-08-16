package generator

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrDuplicateCommand is returned by BuildTree when two operations under
// the same command group resolve to the same leaf command name; this
// signals a naming collision that overrides.yaml must resolve.
var ErrDuplicateCommand = errors.New("duplicate command name")

// commandRankOf lists, in order, the common CRUD verbs so --help lists
// them in a predictable order; any command name not listed here sorts
// alphabetically after all of them (see commandRank).
//
//nolint:gochecknoglobals // read-only lookup table
var commandRankOf = []string{"list", "view", "create", "update", "delete"}

// resourceNode is the mutable tree used while grouping operations, before
// it is converted into the public, read-only Resource tree.
type resourceNode struct {
	operations []*Operation
	children   map[string]*resourceNode
}

func newResourceNode() *resourceNode {
	return &resourceNode{children: map[string]*resourceNode{}}
}

// BuildTree groups a flat, override-applied operation list into the
// nested Resource tree that mirrors the CLI's command structure, using
// each operation's CommandPath and CommandName.
func BuildTree(operations []*Operation) ([]*Resource, error) {
	root := newResourceNode()

	for _, operation := range operations {
		node := root
		for _, segment := range operation.CommandPath {
			child, ok := node.children[segment]
			if !ok {
				child = newResourceNode()
				node.children[segment] = child
			}

			node = child
		}

		node.operations = append(node.operations, operation)
	}

	return buildChildren(root, nil)
}

func buildChildren(node *resourceNode, parentPath []string) ([]*Resource, error) {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}

	sort.Strings(names)

	resources := make([]*Resource, 0, len(names))

	for _, name := range names {
		child := node.children[name]
		path := append(append([]string{}, parentPath...), name)

		if err := checkDuplicateCommands(child.operations, path); err != nil {
			return nil, err
		}

		sortOperations(child.operations)

		children, err := buildChildren(child, path)
		if err != nil {
			return nil, err
		}

		resources = append(resources, &Resource{
			Name:       name,
			Path:       path,
			Usage:      "Manage " + strings.ReplaceAll(name, "-", " "),
			Operations: child.operations,
			Children:   children,
		})
	}

	return resources, nil
}

func checkDuplicateCommands(operations []*Operation, path []string) error {
	seen := make(map[string]bool, len(operations))

	for _, op := range operations {
		if seen[op.CommandName] {
			return fmt.Errorf("%s %s: %w", strings.Join(path, " "), op.CommandName, ErrDuplicateCommand)
		}

		seen[op.CommandName] = true
	}

	return nil
}

func sortOperations(operations []*Operation) {
	sort.SliceStable(operations, func(i, j int) bool {
		return commandRank(operations[i].CommandName) < commandRank(operations[j].CommandName)
	})
}

func commandRank(name string) string {
	for i, known := range commandRankOf {
		if known == name {
			return fmt.Sprintf("0%02d", i)
		}
	}

	return "1" + name
}
