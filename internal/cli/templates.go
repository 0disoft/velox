package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/velox/internal/initializer"
)

type TemplatesResult struct {
	Templates []initializer.Template `json:"templates"`
}

func runTemplates(args []string, dependencies Dependencies) int {
	flags := flag.NewFlagSet("templates", flag.ContinueOnError)
	flags.SetOutput(dependencies.Stderr)
	jsonOutput := flags.Bool("json", false, "emit one JSON document")
	quiet := flags.Bool("quiet", false, "suppress successful human output")
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "templates", *jsonOutput || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 0 {
		return emitFailure(dependencies, "templates", *jsonOutput || jsonRequested(args), 2, "USAGE_INVALID", "Templates does not accept positional arguments.", errors.New("unexpected positional arguments"))
	}
	result := TemplatesResult{Templates: initializer.Templates()}
	if *jsonOutput {
		return emitSuccessJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "templates", Result: result, Diagnostics: []Diagnostic{}})
	}
	if !*quiet {
		for index, template := range result.Templates {
			if index > 0 {
				fmt.Fprintln(dependencies.Stdout)
			}
			permissions := strings.Join(template.Permissions, ", ")
			if permissions == "" {
				permissions = "none"
			}
			fmt.Fprintf(dependencies.Stdout, "%s - %s\n  Permissions: %s\n  %s\n", template.Name, template.Description, permissions, template.InitCommand)
		}
	}
	return 0
}
