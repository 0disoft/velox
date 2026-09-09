//go:build windows

package evalsandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const hermesRequestLimit = 8192
const hermesResponseLimit = 65536

func TestBrokerHermesProviderClient(t *testing.T) {
	if os.Getenv("VELOX_HERMES_PROBE") != "1" {
		t.Skip("opt-in installed Hermes provider-client diagnostic")
	}
	runBrokerProbe(t, "hermes")
}

func stageHermesProbe(t *testing.T, tool string) (string, []string) {
	t.Helper()
	root := filepath.Join(os.Getenv("LOCALAPPDATA"), "hermes", "hermes-agent")
	runtime := filepath.Join(os.Getenv("APPDATA"), "uv", "python", "cpython-3.11-windows-x86_64-none")
	packages := filepath.Join(root, "venv", "Lib", "site-packages")
	stagedRuntime, stagedPackages := filepath.Join(tool, "python"), filepath.Join(tool, "sdk")
	var copiedBytes int64
	copyTree := func(source, target string) {
		t.Helper()
		err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("linked runtime input denied")
			}
			if entry.IsDir() && (entry.Name() == "__pycache__" || entry.Name() == "test" || entry.Name() == "tests" || entry.Name() == "site-packages") {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			destination := filepath.Join(target, rel)
			if entry.IsDir() {
				return os.MkdirAll(destination, 0700)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("nonregular runtime input denied")
			}
			copiedBytes += info.Size()
			if copiedBytes > 200*1024*1024 {
				return fmt.Errorf("runtime staging budget exceeded")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"python.exe", "python3.dll", "python311.dll", "vcruntime140.dll", "vcruntime140_1.dll", "DLLs", "Lib"} {
		copyTree(filepath.Join(runtime, name), filepath.Join(stagedRuntime, name))
	}
	for _, name := range []string{"openai", "httpx", "httpcore", "anyio", "certifi", "idna", "sniffio", "h11", "pydantic", "pydantic_core", "typing_extensions.py", "typing_inspection", "annotated_types", "distro", "jiter", "tqdm", "yaml"} {
		copyTree(filepath.Join(packages, name), filepath.Join(stagedPackages, name))
	}
	executable := filepath.Join(stagedRuntime, "python.exe")
	// Copy only public helper source, never the Hermes home, config, memories or DB.
	for source, target := range map[string]string{
		filepath.Join(root, "agent", "process_bootstrap.py"): filepath.Join(tool, "agent", "process_bootstrap.py"),
		filepath.Join(root, "utils.py"):                      filepath.Join(tool, "utils.py"),
		filepath.Join("testdata", "hermes_pipe_probe.py"):    filepath.Join(tool, "hermes_pipe_probe.py"),
	} {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return executable, []string{"VELOX_HERMES_SITE_PACKAGES=" + stagedPackages}
}

func validateHermesProbe(frame []byte) ([]byte, error) {
	var envelope struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Body   struct {
			Model     string `json:"model"`
			Stream    *bool  `json:"stream"`
			MaxTokens int    `json:"max_tokens"`
			Messages  []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"body"`
	}
	decoder := json.NewDecoder(bytes.NewReader(frame))
	decoder.DisallowUnknownFields()
	if len(frame) > hermesRequestLimit || decoder.Decode(&envelope) != nil {
		return nil, fmt.Errorf("invalid request frame")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return nil, fmt.Errorf("trailing request data")
	}
	b := envelope.Body
	if envelope.Method != "POST" || envelope.Path != "/v1/chat/completions" || b.Model != probeModel || b.Stream == nil || *b.Stream || b.MaxTokens != 1024 || len(b.Messages) != 1 || b.Messages[0].Role != "user" || b.Messages[0].Content != "Reply exactly OK. Do not use tools." {
		return nil, fmt.Errorf("probe request denied")
	}
	// Forward a new canonical body, not child-controlled headers or raw JSON.
	return json.Marshal(b)
}

func serveHermesProbe(reader io.Reader, writer io.Writer, live bool) error {
	frame, err := bufio.NewReader(io.LimitReader(reader, hermesRequestLimit+1)).ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read probe frame: %w", err)
	}
	body, err := validateHermesProbe(frame)
	if err != nil {
		_, _ = io.WriteString(writer, "{\"status\":403,\"body\":{}}\n")
		return err
	}
	payload := []byte(`{"id":"probe","object":"chat.completion","created":0,"model":"commandcode/meta-muse-spark-1.3-contributor","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}]}`)
	if live {
		client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		defer client.CloseIdleConnections()
		request, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:10100/v1/chat/completions", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("local provider request failed")
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return fmt.Errorf("local provider HTTP %d", response.StatusCode)
		}
		payload, err = io.ReadAll(io.LimitReader(response.Body, hermesResponseLimit+1))
		if err != nil || len(payload) > hermesResponseLimit || !json.Valid(payload) {
			return fmt.Errorf("invalid or oversized provider response")
		}
	}
	response, err := json.Marshal(struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}{200, payload})
	if err != nil || len(response)+1 > hermesResponseLimit {
		return fmt.Errorf("response frame too large")
	}
	_, err = writer.Write(append(response, '\n'))
	return err
}

func TestBrokerHermesRequestPolicy(t *testing.T) {
	valid := `{"method":"POST","path":"/v1/chat/completions","body":{"model":"` + probeModel + `","stream":false,"max_tokens":1024,"messages":[{"role":"user","content":"Reply exactly OK. Do not use tools."}]}}` + "\n"
	if _, err := validateHermesProbe([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for name, frame := range map[string]string{
		"method": strings.Replace(valid, "POST", "GET", 1), "path": strings.Replace(valid, "/v1/chat/completions", "/api/config", 1),
		"model": strings.Replace(valid, probeModel, "xai/grok-4.6", 1), "stream": strings.Replace(valid, "false", "true", 1),
		"tokens": strings.Replace(valid, "1024", "1025", 1), "prompt": strings.Replace(valid, "Reply exactly OK.", "Something else.", 1),
		"headers": strings.Replace(valid, `"body":`, `"headers":{"Authorization":"blocked"},"body":`, 1),
		"tools":   strings.Replace(valid, `"model":`, `"tools":[],"model":`, 1), "trailing": valid + "{}", "size": strings.Repeat("x", hermesRequestLimit+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateHermesProbe([]byte(frame)); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
	var output bytes.Buffer
	if err := serveHermesProbe(strings.NewReader(valid), &output, false); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(bytes.TrimSpace(output.Bytes())) {
		t.Fatal("invalid response frame")
	}
}
