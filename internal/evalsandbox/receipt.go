package evalsandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	ReceiptVersion = "velox.eval-sandbox-receipt/v1"
	PolicyVersion  = "velox.eval-sandbox-policy/v1"
)

type Config struct {
	TrialID              string
	SeriesID             string
	Sequence             int
	TrialRoot            string
	ToolRoots            []string
	PassEnvironment      []string
	SessionIDEnvironment string
	ReceiptPath          string
	Timeout              time.Duration
	Command              []string
}

type Receipt struct {
	SchemaVersion     string      `json:"schemaVersion"`
	TrialID           string      `json:"trialId"`
	SeriesID          string      `json:"seriesId"`
	Sequence          int         `json:"sequence"`
	Policy            Policy      `json:"policy"`
	Supervisor        Supervisor  `json:"supervisor"`
	CommandSHA256     string      `json:"commandSha256"`
	EnvironmentSHA256 string      `json:"environmentSha256"`
	SessionIDSHA256   string      `json:"sessionIdSha256"`
	StartedAtUTC      string      `json:"startedAtUtc"`
	FinishedAtUTC     string      `json:"finishedAtUtc"`
	ExitCode          uint32      `json:"exitCode"`
	TimedOut          bool        `json:"timedOut"`
	Containment       Containment `json:"containment"`
	Grants            []Grant     `json:"grants"`
}

type Policy struct {
	SchemaVersion      string `json:"schemaVersion"`
	Platform           string `json:"platform"`
	FilesystemBoundary string `json:"filesystemBoundary"`
	ProcessBoundary    string `json:"processBoundary"`
	NetworkCapability  string `json:"networkCapability"`
}

type Supervisor struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Containment struct {
	FilesystemEnforced  bool `json:"filesystemEnforced"`
	ProcessTreeEnforced bool `json:"processTreeEnforced"`
	CleanupCompleted    bool `json:"cleanupCompleted"`
}

type Grant struct {
	Role       string `json:"role"`
	PathSHA256 string `json:"pathSha256"`
	Rights     string `json:"rights"`
}

type preparedConfig struct {
	Config
	Executable             string
	Grants                 []preparedGrant
	Environment            []string
	EnvironmentSHA256      string
	SessionIDSHA256        string
	PrivateEnvironmentRoot string
}

type preparedGrant struct {
	Path   string
	Role   string
	Rights string
}

func prepare(config Config) (preparedConfig, error) {
	if runtime.GOOS != "windows" {
		return preparedConfig{}, fmt.Errorf("evaluation sandbox is supported only on Windows")
	}
	if !trialIDPattern.MatchString(config.TrialID) || !seriesIDPattern.MatchString(config.SeriesID) {
		return preparedConfig{}, fmt.Errorf("invalid trial or series identity")
	}
	if config.Sequence < 1 || config.Sequence > 3 {
		return preparedConfig{}, fmt.Errorf("sequence must be between 1 and 3")
	}
	if config.Timeout < time.Second || config.Timeout > 2*time.Hour {
		return preparedConfig{}, fmt.Errorf("timeout must be between 1s and 2h")
	}
	if len(config.Command) == 0 {
		return preparedConfig{}, fmt.Errorf("command is required")
	}
	if len(config.ToolRoots) == 0 || len(config.ToolRoots) > 15 {
		return preparedConfig{}, fmt.Errorf("between one and fifteen tool roots are required")
	}
	if !environmentNamePattern.MatchString(config.SessionIDEnvironment) {
		return preparedConfig{}, fmt.Errorf("session ID environment variable is required")
	}
	sessionID, exists := os.LookupEnv(config.SessionIDEnvironment)
	if !exists || sessionID == "" {
		return preparedConfig{}, fmt.Errorf("session ID environment variable %q is not set", config.SessionIDEnvironment)
	}
	config.PassEnvironment = append(config.PassEnvironment, config.SessionIDEnvironment)
	passEnvironment, err := validateEnvironmentNames(config.PassEnvironment)
	if err != nil {
		return preparedConfig{}, err
	}

	trialRoot, err := safeDirectory(config.TrialRoot)
	if err != nil {
		return preparedConfig{}, fmt.Errorf("trial root: %w", err)
	}
	receiptPath, err := safeOutputPath(config.ReceiptPath, trialRoot)
	if err != nil {
		return preparedConfig{}, err
	}

	toolRoots := make([]string, 0, len(config.ToolRoots))
	seen := map[string]struct{}{strings.ToLower(trialRoot): {}}
	for _, root := range config.ToolRoots {
		clean, cleanErr := safeDirectory(root)
		if cleanErr != nil {
			return preparedConfig{}, fmt.Errorf("tool root: %w", cleanErr)
		}
		key := strings.ToLower(clean)
		if _, exists := seen[key]; exists {
			return preparedConfig{}, fmt.Errorf("tool roots must be distinct from the trial root and each other")
		}
		seen[key] = struct{}{}
		toolRoots = append(toolRoots, clean)
	}
	sort.Slice(toolRoots, func(i, j int) bool { return strings.ToLower(toolRoots[i]) < strings.ToLower(toolRoots[j]) })

	executable, err := filepath.Abs(config.Command[0])
	if err != nil {
		return preparedConfig{}, fmt.Errorf("resolve command: %w", err)
	}
	executable = filepath.Clean(executable)
	if !strings.EqualFold(filepath.Ext(executable), ".exe") {
		return preparedConfig{}, fmt.Errorf("sandbox command must name an executable file")
	}
	info, err := os.Lstat(executable)
	if err != nil || !info.Mode().IsRegular() {
		return preparedConfig{}, fmt.Errorf("sandbox executable must be a regular file")
	}
	if !containedByAny(executable, toolRoots) {
		return preparedConfig{}, fmt.Errorf("sandbox executable must be contained by a tool root")
	}

	grants := []preparedGrant{{Path: trialRoot, Role: "trial-read-write-execute", Rights: "read-write-execute"}}
	for _, root := range toolRoots {
		grants = append(grants, preparedGrant{Path: root, Role: "tool-read-execute", Rights: "read-execute"})
	}
	config.TrialRoot = trialRoot
	config.ToolRoots = toolRoots
	config.ReceiptPath = receiptPath
	config.Command = append([]string{executable}, config.Command[1:]...)
	environment, environmentSHA, privateRoot, err := prepareEnvironment(trialRoot, passEnvironment)
	if err != nil {
		return preparedConfig{}, err
	}
	config.PassEnvironment = passEnvironment
	return preparedConfig{
		Config:                 config,
		Executable:             executable,
		Grants:                 grants,
		Environment:            environment,
		EnvironmentSHA256:      environmentSHA,
		SessionIDSHA256:        digest([]byte(sessionID)),
		PrivateEnvironmentRoot: privateRoot,
	}, nil
}

func validateEnvironmentNames(names []string) ([]string, error) {
	reserved := map[string]struct{}{
		"SYSTEMROOT": {}, "WINDIR": {}, "TEMP": {}, "TMP": {}, "HOME": {}, "USERPROFILE": {},
		"APPDATA": {}, "LOCALAPPDATA": {},
	}
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		upper := strings.ToUpper(name)
		if !environmentNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid environment variable name %q", name)
		}
		if _, blocked := reserved[upper]; blocked {
			return nil, fmt.Errorf("environment variable %q is managed by the sandbox", name)
		}
		if _, duplicate := seen[upper]; duplicate {
			return nil, fmt.Errorf("duplicate environment variable %q", name)
		}
		if _, exists := os.LookupEnv(name); !exists {
			return nil, fmt.Errorf("environment variable %q is not set", name)
		}
		seen[upper] = struct{}{}
		result = append(result, name)
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToUpper(result[i]) < strings.ToUpper(result[j]) })
	return result, nil
}

func prepareEnvironment(trialRoot string, passNames []string) ([]string, string, string, error) {
	privateRoot := filepath.Join(trialRoot, ".velox-sandbox")
	home := filepath.Join(privateRoot, "home")
	temp := filepath.Join(privateRoot, "tmp")
	appData := filepath.Join(home, "AppData", "Roaming")
	localAppData := filepath.Join(home, "AppData", "Local")
	for _, directory := range []string{temp, appData, localAppData} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			_ = os.RemoveAll(privateRoot)
			return nil, "", "", fmt.Errorf("create private sandbox environment: %w", err)
		}
	}
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		_ = os.RemoveAll(privateRoot)
		return nil, "", "", fmt.Errorf("SystemRoot is required")
	}
	values := map[string]string{
		"SystemRoot":   systemRoot,
		"WINDIR":       systemRoot,
		"TEMP":         temp,
		"TMP":          temp,
		"HOME":         home,
		"USERPROFILE":  home,
		"APPDATA":      appData,
		"LOCALAPPDATA": localAppData,
	}
	for _, name := range passNames {
		values[name] = os.Getenv(name)
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return strings.ToUpper(names[i]) < strings.ToUpper(names[j]) })
	environment := make([]string, 0, len(names))
	projection := make([]map[string]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+values[name])
		projection = append(projection, map[string]string{"name": strings.ToUpper(name), "valueSha256": digest([]byte(values[name]))})
	}
	body, err := json.Marshal(projection)
	if err != nil {
		_ = os.RemoveAll(privateRoot)
		return nil, "", "", err
	}
	return environment, digest(body), privateRoot, nil
}

func safeDirectory(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute")
	}
	clean := filepath.Clean(path)
	volumeRoot := filepath.Clean(filepath.VolumeName(clean) + string(filepath.Separator))
	if strings.EqualFold(clean, volumeRoot) {
		return "", fmt.Errorf("volume root is not an allowed grant")
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("path must be a non-symlink directory")
	}
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(clean, real) {
		return "", fmt.Errorf("reparse-point roots are not allowed")
	}
	return clean, nil
}

func safeOutputPath(path, trialRoot string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("receipt path must be absolute")
	}
	clean := filepath.Clean(path)
	if contained(clean, trialRoot) {
		return "", fmt.Errorf("receipt path must be outside the agent-controlled trial root")
	}
	if _, err := os.Lstat(clean); err == nil {
		return "", fmt.Errorf("receipt path already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent, err := safeDirectory(filepath.Dir(clean))
	if err != nil {
		return "", fmt.Errorf("receipt parent: %w", err)
	}
	return filepath.Join(parent, filepath.Base(clean)), nil
}

func containedByAny(path string, roots []string) bool {
	for _, root := range roots {
		if contained(path, root) {
			return true
		}
	}
	return false
}

func contained(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func commandDigest(command []string) (string, error) {
	body, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	return digest(body), nil
}

func fileDigest(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digest(body), nil
}

func pathDigest(path string) string {
	return digest([]byte(strings.ToLower(filepath.Clean(path))))
}

func receiptGrants(grants []preparedGrant) []Grant {
	result := make([]Grant, 0, len(grants))
	for _, grant := range grants {
		result = append(result, Grant{Role: grant.Role, PathSHA256: pathDigest(grant.Path), Rights: grant.Rights})
	}
	return result
}

func WriteReceiptExclusive(path string, receipt Receipt) error {
	body, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
