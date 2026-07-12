package doctor

import (
	"errors"
	"fmt"

	"github.com/0disoft/velox/internal/buildplan"
)

const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusBlocked = "blocked"
)

type Check struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Actual   string `json:"actual,omitempty"`
	Expected string `json:"expected,omitempty"`
	Message  string `json:"message"`
}

type Project struct {
	AppID          string `json:"appId"`
	AppVersion     string `json:"appVersion"`
	ReleaseVersion string `json:"releaseVersion"`
	Target         string `json:"target"`
}

type Result struct {
	Ready   bool     `json:"ready"`
	Checks  []Check  `json:"checks"`
	Project *Project `json:"project,omitempty"`
}

type Failure struct {
	ExitCode int
	Code     string
	Message  string
}

type Evidence struct {
	GOOS               string
	GOARCH             string
	WebView2Version    string
	WebView2ProbeError error
	Plan               buildplan.Plan
	PlanError          error
}

func Evaluate(evidence Evidence) (Result, *Failure) {
	result := Result{Checks: make([]Check, 0, 4)}
	var failure *Failure

	platform := evidence.GOOS + "-" + evidence.GOARCH
	if evidence.GOOS == "windows" && evidence.GOARCH == "amd64" {
		result.Checks = append(result.Checks, Check{Name: "platform", Status: StatusPass, Actual: platform, Expected: "windows-amd64", Message: "Windows x64 is supported."})
	} else {
		result.Checks = append(result.Checks, Check{Name: "platform", Status: StatusFail, Actual: platform, Expected: "windows-amd64", Message: "Velox currently requires Windows x64."})
		failure = &Failure{ExitCode: 5, Code: "RUNTIME_PLATFORM_UNSUPPORTED", Message: "The current platform is unsupported."}
	}

	switch {
	case evidence.WebView2ProbeError != nil:
		result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusFail, Message: "The installed WebView2 Runtime could not be queried."})
		if failure == nil {
			failure = &Failure{ExitCode: 5, Code: "RUNTIME_WEBVIEW2_PROBE_FAILED", Message: "WebView2 Runtime detection failed."}
		}
	case evidence.WebView2Version == "":
		result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusFail, Expected: "Evergreen WebView2 Runtime", Message: "The Evergreen WebView2 Runtime is not installed."})
		if failure == nil {
			failure = &Failure{ExitCode: 5, Code: "RUNTIME_WEBVIEW2_UNAVAILABLE", Message: "WebView2 Runtime is unavailable."}
		}
	default:
		result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusPass, Actual: evidence.WebView2Version, Expected: "Evergreen WebView2 Runtime", Message: "WebView2 Runtime is installed."})
	}

	if evidence.PlanError == nil {
		snapshot := evidence.Plan.Snapshot()
		result.Checks = append(result.Checks,
			Check{Name: "project", Status: StatusPass, Actual: snapshot.Manifest.App.ID, Message: "Project manifest and assets are valid."},
			Check{Name: "host", Status: StatusPass, Actual: snapshot.HostMetadata.ReleaseVersion, Expected: snapshot.Target, Message: "Bundled host metadata and digest are compatible."},
		)
		result.Project = &Project{AppID: snapshot.Manifest.App.ID, AppVersion: snapshot.Manifest.App.Version, ReleaseVersion: snapshot.HostMetadata.ReleaseVersion, Target: snapshot.Target}
	} else {
		planFailure := classifyPlanError(evidence.PlanError)
		if planFailure.host {
			result.Checks = append(result.Checks,
				Check{Name: "project", Status: StatusPass, Message: "Project manifest and assets are valid."},
				Check{Name: "host", Status: StatusFail, Expected: buildplan.TargetWindowsX64, Message: "Bundled host metadata or digest is incompatible."},
			)
		} else {
			result.Checks = append(result.Checks,
				Check{Name: "project", Status: StatusFail, Message: planFailure.projectMessage},
				Check{Name: "host", Status: StatusBlocked, Message: "Bundled host validation was not reached because the project is invalid."},
			)
		}
		if failure == nil {
			failure = &Failure{ExitCode: planFailure.exitCode, Code: planFailure.code, Message: planFailure.message}
		}
	}

	result.Ready = failure == nil
	return result, failure
}

type planFailure struct {
	host           bool
	exitCode       int
	code           string
	message        string
	projectMessage string
}

func classifyPlanError(err error) planFailure {
	var typed *buildplan.Error
	if !errors.As(err, &typed) {
		return planFailure{exitCode: 10, code: "INTERNAL", message: "Unexpected internal failure.", projectMessage: "Project validation failed unexpectedly."}
	}
	switch typed.Kind {
	case buildplan.ErrorHost:
		return planFailure{host: true, exitCode: 4, code: "HOST_INCOMPATIBLE", message: "Host template is unavailable or incompatible."}
	case buildplan.ErrorAsset:
		return planFailure{exitCode: 3, code: "ASSET_INVALID", message: "Project assets are invalid.", projectMessage: "Project assets or entry point are invalid."}
	case buildplan.ErrorManifest, buildplan.ErrorConfig:
		return planFailure{exitCode: 2, code: "MANIFEST_INVALID", message: "Project manifest is invalid.", projectMessage: "Project manifest or configuration is invalid."}
	default:
		return planFailure{exitCode: 10, code: "INTERNAL", message: "Unexpected internal failure.", projectMessage: fmt.Sprintf("Project validation failed with unknown category %q.", typed.Kind)}
	}
}
