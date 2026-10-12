package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/velox/internal/buildplan"
)

func parseFlags(flags *flag.FlagSet, args []string) error {
	output := flags.Output()
	// The flag package echoes invalid values before returning an error.
	flags.SetOutput(io.Discard)
	err := flags.Parse(args)
	flags.SetOutput(output)
	if errors.Is(err, flag.ErrHelp) {
		flags.Usage()
	}
	return err
}

func planFailureMessage(kind buildplan.ErrorKind, err error, target string) string {
	switch kind {
	case buildplan.ErrorConfig:
		if target != "" && target != buildplan.TargetWindowsX64 {
			return "Unsupported build target. Use --target windows-x64."
		}
		return "Build output settings are invalid. Set --out outside assets.root; neither directory may contain the other. Check directory access."
	case buildplan.ErrorManifest:
		if errors.Is(err, os.ErrNotExist) {
			return "Project manifest or branding file was not found. Check --config and branding.icon; use velox init in a new directory to create a project."
		}
		if errors.Is(err, os.ErrPermission) {
			return "Project manifest or branding file cannot be read. Check file access permissions."
		}
		var syntax *json.SyntaxError
		var valueType *json.UnmarshalTypeError
		if errors.As(err, &syntax) {
			return "Project manifest contains invalid JSON. Correct the JSON syntax at the reported location."
		}
		if errors.As(err, &valueType) {
			return "Project manifest contains an invalid value type. Check JSON value types at the reported location against the schema."
		}
		return "Project manifest is invalid. Check the manifest fields against the schema."
	case buildplan.ErrorAsset:
		return "Project assets are unavailable or invalid. Check assets.root and assets.entry, file access, and unsupported links."
	case buildplan.ErrorHost:
		return "Host template is unavailable or incompatible. Extract the complete matching Velox release with velox-host.exe and velox-host.json beside the CLI."
	default:
		return "Unexpected internal failure."
	}
}

func failureGuidance(command, code, message string) string {
	switch code {
	case "USAGE_INVALID":
		return fmt.Sprintf("%s Run 'velox %s --help' for accepted options.", message, command)
	case "INIT_FAILED":
		return message + " Use an empty writable project directory; existing files are not overwritten."
	case "PACKAGING_FAILED":
		return message + " Check output directory access and locked files; for --installer, extract the matching velox-setup.exe beside the CLI."
	case "ARTIFACT_INVALID":
		return message + " Inspect the extracted app directory or portable ZIP; use an intact artifact or rebuild the project."
	case "RUNTIME_HOST_START_FAILED", "RUNTIME_HOST_EXITED":
		return message + " Run velox doctor with the same --config to check the runtime and project."
	default:
		return message
	}
}
