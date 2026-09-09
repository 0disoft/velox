//go:build windows

package evalsandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

const probeModel = "commandcode/meta-muse-spark-1.3-contributor"

func TestBrokerRejectsNonPipeHandles(t *testing.T) {
	first, err := os.CreateTemp(t.TempDir(), "first")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := os.CreateTemp(t.TempDir(), "second")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if _, _, err := launchContainedWithHandles(preparedConfig{}, nil, nil, []windows.Handle{windows.Handle(first.Fd()), windows.Handle(second.Fd())}); err == nil {
		t.Fatal("non-pipe handles were accepted")
	}
}

func TestBrokerInheritedPipeBoundary(t *testing.T) {
	for _, mode := range []string{"allowed", "denied", "timeout"} {
		t.Run(mode, func(t *testing.T) { runBrokerProbe(t, mode) })
	}
}

func runBrokerProbe(t *testing.T, mode string) {
	t.Helper()
	base, err := os.MkdirTemp(".", ".velox-eval-broker-test-")
	if err != nil {
		t.Fatal(err)
	}
	base, err = filepath.Abs(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(base); err != nil {
			t.Error(err)
		}
	})
	trial, tool := filepath.Join(base, "trial"), filepath.Join(base, "tool")
	for _, path := range []string{trial, tool} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	forbidden := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(forbidden, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(tool, "probe.exe")
	if err := os.WriteFile(executable, body, 0o700); err != nil {
		t.Fatal(err)
	}
	var extraEnvironment []string
	command := []string{executable, "-test.run=^TestBrokerPipeChild$"}
	if mode == "hermes" {
		executable, extraEnvironment = stageHermesProbe(t, tool)
		command = []string{executable, "-I", "-B", filepath.Join(tool, "hermes_pipe_probe.py")}
	}
	name, err := randomProfileName()
	if err != nil {
		t.Fatal(err)
	}
	capability, err := windows.StringToSid("S-1-15-3-1")
	if err != nil {
		t.Fatal(err)
	}
	capabilities := []windows.SIDAndAttributes{{Sid: capability, Attributes: windows.SE_GROUP_ENABLED}}
	sid, err := createProfile(name, capabilities)
	if err != nil {
		t.Fatal(err)
	}
	var applied []preparedGrant
	t.Cleanup(func() {
		for _, grant := range applied {
			if err := updatePathAccess(grant, sid, windows.REVOKE_ACCESS); err != nil {
				t.Error(err)
			}
		}
		if err := deleteProfile(name); err != nil {
			t.Error(err)
		}
		_ = windows.FreeSid(sid)
	})
	for _, grant := range []preparedGrant{{Path: trial, Rights: "read-write-execute"}, {Path: tool, Rights: "read-execute"}} {
		if err := updatePathAccess(grant, sid, windows.GRANT_ACCESS); err != nil {
			t.Fatal(err)
		}
		applied = append(applied, grant)
	}
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer requestRead.Close()
	defer requestWrite.Close()
	responseRead, responseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer responseRead.Close()
	defer responseWrite.Close()
	handles := []windows.Handle{windows.Handle(requestWrite.Fd()), windows.Handle(responseRead.Fd())}
	for _, handle := range handles {
		if err := windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			t.Fatal(err)
		}
	}
	result := make(chan error, 1)
	go func() {
		defer responseWrite.Close()
		if mode == "hermes" {
			result <- serveHermesProbe(requestRead, responseWrite, os.Getenv("VELOX_EVAL_BROKER_LIVE") == "1")
			return
		}
		request, err := bufio.NewReader(io.LimitReader(requestRead, 32)).ReadString('\n')
		if err != nil {
			result <- err
			return
		}
		response := "denied"
		if request == "probe\n" {
			response = "ok"
			if os.Getenv("VELOX_EVAL_BROKER_LIVE") == "1" {
				response, err = probeLiveModel()
				if err != nil {
					response = "failed"
				}
			}
		}
		_, writeErr := io.WriteString(responseWrite, response+"\n")
		if err != nil {
			result <- err
		} else {
			result <- writeErr
		}
	}()
	timeout := 60 * time.Second
	if mode == "timeout" {
		timeout = time.Second
	}
	environment, _, _, err := prepareEnvironment(trial, nil)
	if err != nil {
		t.Fatal(err)
	}
	environment = append(environment,
		"VELOX_BROKER_MODE="+mode, "VELOX_BROKER_FORBIDDEN="+forbidden,
		"VELOX_BROKER_RESULT_ROOT="+trial,
		fmt.Sprintf("VELOX_BROKER_WRITE=%d", handles[0]), fmt.Sprintf("VELOX_BROKER_READ=%d", handles[1]),
	)
	environment = append(environment, extraEnvironment...)
	sort.Slice(environment, func(i, j int) bool { return strings.ToUpper(environment[i]) < strings.ToUpper(environment[j]) })
	config := preparedConfig{
		Config:      Config{TrialRoot: trial, Timeout: timeout, Command: command},
		Executable:  executable,
		Environment: environment,
	}
	exitCode, timedOut, launchErr := launchContainedWithHandles(config, sid, capabilities, handles)
	_ = requestRead.Close()
	_ = requestWrite.Close()
	_ = responseRead.Close()
	_ = responseWrite.Close()
	select {
	case brokerErr := <-result:
		if mode != "timeout" && brokerErr != nil {
			t.Errorf("broker: %v", brokerErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("broker did not stop after pipe closure")
	}
	if launchErr != nil {
		t.Fatal(launchErr)
	}
	if mode == "timeout" {
		if !timedOut {
			t.Fatal("timeout did not terminate the process tree")
		}
	} else if timedOut || exitCode != 0 {
		t.Fatalf("sandbox child failed: exit=%d timeout=%t", exitCode, timedOut)
	}
	if data, err := os.ReadFile(forbidden); err != nil || string(data) != "outside" {
		t.Fatal("outside sentinel changed")
	}
	if mode == "hermes" {
		data, err := os.ReadFile(filepath.Join(trial, "hermes-probe.json"))
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			ProviderClient      string `json:"providerClient"`
			Requests            int    `json:"requests"`
			SecondRequestDenied bool   `json:"secondRequestDenied"`
			OutsideFileDenied   bool   `json:"outsideFileDenied"`
			QualifyingTrial     bool   `json:"qualifyingTrial"`
		}
		if json.Unmarshal(data, &report) != nil || report.ProviderClient != "hermes.process_bootstrap.OpenAI" || report.Requests != 1 || !report.SecondRequestDenied || !report.OutsideFileDenied || report.QualifyingTrial {
			t.Fatal("invalid Hermes provider-client diagnostic")
		}
	}
}

func TestBrokerPipeChild(t *testing.T) {
	mode := os.Getenv("VELOX_BROKER_MODE")
	if mode == "" {
		return
	}
	if _, err := os.ReadFile(os.Getenv("VELOX_BROKER_FORBIDDEN")); err == nil {
		t.Fatal("outside file was readable")
	}
	if mode == "timeout" {
		time.Sleep(time.Minute)
		return
	}
	open := func(name string) *os.File {
		handle, err := strconv.ParseUint(os.Getenv(name), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		file := os.NewFile(uintptr(handle), name)
		if file == nil {
			t.Fatal("invalid inherited pipe")
		}
		return file
	}
	writer, reader := open("VELOX_BROKER_WRITE"), open("VELOX_BROKER_READ")
	defer writer.Close()
	defer reader.Close()
	request, expected := "probe\n", "ok\n"
	if mode == "denied" {
		request, expected = "GET /api/config\n", "denied\n"
	}
	if _, err := io.WriteString(writer, request); err != nil {
		t.Fatal(err)
	}
	response, err := bufio.NewReader(io.LimitReader(reader, 64)).ReadString('\n')
	if err != nil || response != expected {
		t.Fatalf("unexpected broker response: %q %v", response, err)
	}
}

func probeLiveModel() (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": probeModel, "stream": false, "max_tokens": 128,
		"messages": []map[string]string{{"role": "user", "content": "Reply exactly OK. Do not use tools."}},
	})
	request, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:10100/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("local provider request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local provider returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&payload); err != nil {
		return "", fmt.Errorf("invalid provider response")
	}
	if !strings.Contains(payload.Model, "muse-spark-1.3") || len(payload.Choices) != 1 || strings.TrimSpace(payload.Choices[0].Message.Content) != "OK" {
		return "", fmt.Errorf("provider model or response did not match the fixed probe")
	}
	return "ok", nil
}
