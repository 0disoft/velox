package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

type Launcher func(hostPath, configPath string, stdout, stderr io.Writer) (int, error)

type Result struct {
	ExitCode int `json:"exitCode"`
}

type HostExitError struct {
	Code int
}

func (err *HostExitError) Error() string {
	return fmt.Sprintf("host exited with code %d", err.Code)
}

func Execute(plan buildplan.Plan, launcher Launcher, stdout, stderr io.Writer) (Result, error) {
	if launcher == nil {
		launcher = Launch
	}
	snapshot := plan.Snapshot()
	configFile, err := os.CreateTemp(snapshot.Manifest.ProjectRoot, ".velox-run-*.json")
	if err != nil {
		return Result{}, fmt.Errorf("create temporary runtime config: %w", err)
	}
	configPath := configFile.Name()
	removeConfig := func() error {
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove temporary runtime config: %w", err)
		}
		return nil
	}

	encoder := json.NewEncoder(configFile)
	encoder.SetEscapeHTML(false)
	writeErr := encoder.Encode(runtimeconfig.FromManifest(snapshot.Manifest, snapshot.Manifest.Assets.Root))
	closeErr := configFile.Close()
	if writeErr != nil || closeErr != nil {
		_ = removeConfig()
		return Result{}, fmt.Errorf("write temporary runtime config: %w", errors.Join(writeErr, closeErr))
	}

	exitCode, launchErr := launcher(snapshot.HostPath, configPath, stdout, stderr)
	cleanupErr := removeConfig()
	if launchErr != nil {
		return Result{ExitCode: exitCode}, errors.Join(launchErr, cleanupErr)
	}
	if cleanupErr != nil {
		return Result{ExitCode: exitCode}, cleanupErr
	}
	if exitCode != 0 {
		return Result{ExitCode: exitCode}, &HostExitError{Code: exitCode}
	}
	return Result{ExitCode: 0}, nil
}

func Launch(hostPath, configPath string, stdout, stderr io.Writer) (int, error) {
	command := exec.Command(hostPath, "--config", configPath)
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if err == nil {
		return 0, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if code := exitError.ExitCode(); code > 0 {
			return code, nil
		}
		return 6, fmt.Errorf("host terminated without a usable exit code: %w", err)
	}
	return 6, fmt.Errorf("start host process: %w", err)
}
