package doctor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/actutum/internal/buildplan"
)

const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusBlocked = "blocked"

	MinimumWindowsClientBuild = 16299
	MinimumWindowsServerBuild = 14393
	MinimumWebView2Version    = "92.0.902.49"
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
	WindowsVersion     WindowsVersion
	WebView2Version    string
	WebView2ProbeError error
	Plan               buildplan.Plan
	PlanError          error
}

type WindowsVersion struct {
	Major    uint32
	Minor    uint32
	Build    uint32
	IsServer bool
}

func Evaluate(evidence Evidence) (Result, *Failure) {
	result := Result{Checks: make([]Check, 0, 5)}
	var failure *Failure

	platform := evidence.GOOS + "-" + evidence.GOARCH
	if evidence.GOOS == "windows" && evidence.GOARCH == "amd64" {
		result.Checks = append(result.Checks, Check{Name: "platform", Status: StatusPass, Actual: platform, Expected: "windows-amd64", Message: "Windows x64 is supported."})
	} else {
		result.Checks = append(result.Checks, Check{Name: "platform", Status: StatusFail, Actual: platform, Expected: "windows-amd64", Message: "Actutum currently requires Windows x64."})
		failure = &Failure{ExitCode: 5, Code: "RUNTIME_PLATFORM_UNSUPPORTED", Message: "The current platform is unsupported."}
	}

	if evidence.GOOS != "windows" || evidence.GOARCH != "amd64" {
		result.Checks = append(result.Checks, Check{Name: "windows", Status: StatusBlocked, Message: "Windows version validation requires the supported Windows x64 platform."})
	} else if supportedWindows(evidence.WindowsVersion) {
		result.Checks = append(result.Checks, Check{Name: "windows", Status: StatusPass, Actual: formatWindowsVersion(evidence.WindowsVersion), Expected: expectedWindows(evidence.WindowsVersion.IsServer), Message: "The Windows version is supported."})
	} else {
		result.Checks = append(result.Checks, Check{Name: "windows", Status: StatusFail, Actual: formatWindowsVersion(evidence.WindowsVersion), Expected: expectedWindows(evidence.WindowsVersion.IsServer), Message: "The Windows version is unsupported."})
		if failure == nil {
			failure = &Failure{ExitCode: 5, Code: "RUNTIME_WINDOWS_VERSION_UNSUPPORTED", Message: "Actutum requires Windows 10 version 1709 or Windows Server 2016 and a supported x64 build."}
		}
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
		comparison, err := compareBrowserVersions(evidence.WebView2Version, MinimumWebView2Version)
		if err != nil {
			result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusFail, Actual: evidence.WebView2Version, Expected: "Evergreen WebView2 Runtime >= " + MinimumWebView2Version, Message: "The installed WebView2 Runtime version is invalid."})
			if failure == nil {
				failure = &Failure{ExitCode: 5, Code: "RUNTIME_WEBVIEW2_VERSION_INVALID", Message: "The installed WebView2 Runtime reported an invalid version."}
			}
		} else if comparison < 0 {
			result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusFail, Actual: evidence.WebView2Version, Expected: "Evergreen WebView2 Runtime >= " + MinimumWebView2Version, Message: "The installed WebView2 Runtime is too old."})
			if failure == nil {
				failure = &Failure{ExitCode: 5, Code: "RUNTIME_WEBVIEW2_UNSUPPORTED", Message: "Actutum requires WebView2 Runtime 92.0.902.49 or newer."}
			}
		} else {
			result.Checks = append(result.Checks, Check{Name: "webview2", Status: StatusPass, Actual: evidence.WebView2Version, Expected: "Evergreen WebView2 Runtime >= " + MinimumWebView2Version, Message: "WebView2 Runtime is installed and supported."})
		}
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

func supportedWindows(version WindowsVersion) bool {
	if version.Major != 10 {
		return false
	}
	minimum := uint32(MinimumWindowsClientBuild)
	if version.IsServer {
		minimum = MinimumWindowsServerBuild
	}
	return version.Build >= minimum
}

func expectedWindows(server bool) string {
	if server {
		return "Windows Server 2016 x64 or newer"
	}
	return "Windows 10 version 1709 x64 or newer"
}

func formatWindowsVersion(version WindowsVersion) string {
	kind := "client"
	if version.IsServer {
		kind = "server"
	}
	return fmt.Sprintf("%d.%d.%d %s", version.Major, version.Minor, version.Build, kind)
}

func compareBrowserVersions(left, right string) (int, error) {
	leftParts, err := parseBrowserVersion(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := parseBrowserVersion(right)
	if err != nil {
		return 0, err
	}
	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1, nil
		}
		if leftParts[index] > rightParts[index] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseBrowserVersion(value string) ([4]uint64, error) {
	var result [4]uint64
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return result, errors.New("browser version is empty")
	}
	parts := strings.Split(fields[0], ".")
	if len(parts) != len(result) {
		return result, fmt.Errorf("browser version %q must have four numeric parts", value)
	}
	for index, part := range parts {
		parsed, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return result, fmt.Errorf("browser version %q is invalid", value)
		}
		result[index] = parsed
	}
	return result, nil
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
