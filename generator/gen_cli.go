package generator

import (
	"fmt"
	"strings"
)

// RenderCLICommands returns the source of
// internal/generated/cli/commands_gen.go: the top-level Commands()
// entrypoint plus one function per resource group and one per leaf
// command, built from tree (see BuildTree).
func RenderCLICommands(tree []*Resource) string {
	var out codeWriter

	out.writeString(generatedFileHeader)
	out.writeString("package cli\n\n")
	renderCLIImports(&out)
	renderCommandsEntrypoint(&out, tree)

	for _, resource := range tree {
		renderResource(&out, resource)
	}

	return out.String()
}

func renderCLIImports(out *codeWriter) {
	out.writeString("import (\n")
	out.writeString("\t\"context\"\n")
	out.writeString("\t\"fmt\"\n\n")
	out.writeString("\t\"github.com/urfave/cli/v3\"\n\n")
	out.writeString("\t\"github.com/onetwist-software/mailwizz-cli/internal/commands\"\n")
	out.writeString("\t\"github.com/onetwist-software/mailwizz-cli/internal/generated/api\"\n")
	out.writeString("\t\"github.com/onetwist-software/mailwizz-cli/internal/output\"\n")
	out.writeString(")\n\n")
}

func renderCommandsEntrypoint(out *codeWriter, tree []*Resource) {
	out.writeString("// Commands returns every top-level MailWizz API command, generated from\n")
	out.writeString("// openapi/schema.json.\n")
	out.writeString("func Commands() []*cli.Command {\n")
	out.writeString("\treturn []*cli.Command{\n")

	for _, resource := range tree {
		out.printf("\t\t%s(),\n", resource.FuncName())
	}

	out.writeString("\t}\n")
	out.writeString("}\n")
}

func renderResource(out *codeWriter, resource *Resource) {
	out.writeString("\n")
	out.printf("// %s returns the %q command.\n", resource.FuncName(), resource.Name)
	out.printf("func %s() *cli.Command {\n", resource.FuncName())
	out.writeString("\treturn &cli.Command{\n")
	out.printf("\t\tName:  %q,\n", resource.Name)
	out.printf("\t\tUsage: %q,\n", resource.Usage)
	out.writeString("\t\tCommands: []*cli.Command{\n")

	for _, operation := range resource.Operations {
		out.printf("\t\t\t%s(),\n", operation.FuncName())
	}

	for _, child := range resource.Children {
		out.printf("\t\t\t%s(),\n", child.FuncName())
	}

	out.writeString("\t\t},\n")
	out.writeString("\t}\n")
	out.writeString("}\n")

	for _, operation := range resource.Operations {
		renderLeafCommand(out, operation)
	}

	for _, child := range resource.Children {
		renderResource(out, child)
	}
}

func renderLeafCommand(out *codeWriter, operation *Operation) {
	fields := operation.RequestFields()

	out.writeString("\n")
	out.printf("// %s returns the %q command.\n", operation.FuncName(), operation.CommandName)
	out.printf("func %s() *cli.Command {\n", operation.FuncName())
	out.writeString("\treturn &cli.Command{\n")
	out.printf("\t\tName:  %q,\n", operation.CommandName)

	if operation.Summary != "" {
		out.printf("\t\tUsage: %q,\n", operation.Summary)
	}

	renderFlags(out, fields)
	renderAction(out, operation, fields)
	out.writeString("\t}\n")
	out.writeString("}\n")
}

func renderFlags(out *codeWriter, fields []RequestField) {
	if len(fields) == 0 {
		return
	}

	out.writeString("\t\tFlags: []cli.Flag{\n")

	for _, field := range fields {
		renderFlag(out, field)
	}

	out.writeString("\t\t},\n")
}

func renderFlag(out *codeWriter, field RequestField) {
	out.writeString("\t\t\t&cli.StringFlag{\n")
	out.printf("\t\t\t\tName: %q,\n", field.FlagName)

	if usage := flagUsage(field); usage != "" {
		out.printf("\t\t\t\tUsage: %q,\n", usage)
	}

	if field.Required {
		out.writeString("\t\t\t\tRequired: true,\n")
	}

	if len(field.Enum) > 0 {
		renderEnumValidator(out, field)
	}

	out.writeString("\t\t\t},\n")
}

func flagUsage(field RequestField) string {
	usage := field.Description
	if len(field.Enum) == 0 {
		return usage
	}

	allowed := "allowed values: " + strings.Join(field.Enum, ", ")
	if usage == "" {
		return allowed
	}

	return usage + " (" + allowed + ")"
}

func renderEnumValidator(out *codeWriter, field RequestField) {
	errMsg := fmt.Sprintf("%s must be one of: %s", field.FlagName, strings.Join(field.Enum, ", "))

	out.writeString("\t\t\t\tValidator: func(v string) error {\n")
	out.writeString("\t\t\t\t\tswitch v {\n")
	out.printf("\t\t\t\t\tcase \"\", %s:\n", quotedList(field.Enum))
	out.writeString("\t\t\t\t\t\treturn nil\n")
	out.writeString("\t\t\t\t\tdefault:\n")
	out.printf("\t\t\t\t\t\treturn fmt.Errorf(%q)\n", errMsg)
	out.writeString("\t\t\t\t\t}\n")
	out.writeString("\t\t\t\t},\n")
}

func quotedList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}

	return strings.Join(quoted, ", ")
}

func renderAction(out *codeWriter, operation *Operation, fields []RequestField) {
	out.writeString("\t\tAction: func(ctx context.Context, cmd *cli.Command) error {\n")
	out.writeString("\t\t\tclient, err := commands.ResolveClient()\n")
	out.writeString("\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\n")

	if len(fields) == 0 {
		out.printf("\t\t\treq := api.%sRequest{}\n\n", operation.GoName)
	} else {
		out.printf("\t\t\treq := api.%sRequest{\n", operation.GoName)

		for _, field := range fields {
			out.printf("\t\t\t\t%s: cmd.String(%q),\n", field.GoField, field.FlagName)
		}

		out.writeString("\t\t\t}\n\n")
	}

	out.printf("\t\t\tresp, err := client.%s(ctx, req)\n", operation.GoName)
	out.writeString("\t\t\tif err != nil {\n")
	out.printf("\t\t\t\treturn fmt.Errorf(\"%s: %%w\", err)\n", operation.OperationID)
	out.writeString("\t\t\t}\n\n")
	out.writeString("\t\t\treturn output.Handle(cmd.Root().Writer, resp)\n")
	out.writeString("\t\t},\n")
}
