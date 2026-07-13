package ipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

const (
	Version           = 1
	MaxRequestBytes   = 64 << 10
	MaxNestingDepth   = 16
	MaxInflight       = 64
	PermissionAppInfo = "app.info"
	PermissionWindow  = "window.basic"
)

type Identity struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

type Window interface {
	State() (string, error)
	Minimize() error
	Maximize() error
	Restore() error
	Close() error
}

type Request struct {
	Version uint32          `json:"v"`
	ID      uint32          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	Version uint32    `json:"v"`
	ID      uint32    `json:"id"`
	OK      bool      `json:"ok"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Dispatcher struct {
	identity    Identity
	permissions map[string]struct{}
	window      Window

	mu       sync.Mutex
	closing  bool
	inflight map[uint32]struct{}
}

func NewDispatcher(identity Identity, permissions []string, window Window) *Dispatcher {
	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission] = struct{}{}
	}
	return &Dispatcher{
		identity:    identity,
		permissions: granted,
		window:      window,
		inflight:    make(map[uint32]struct{}),
	}
}

func (d *Dispatcher) Dispatch(raw json.RawMessage) Response {
	request, rpcErr := decodeRequest(raw)
	if rpcErr != nil {
		return failure(0, rpcErr.Code, rpcErr.Message)
	}
	if rpcErr := d.begin(request.ID); rpcErr != nil {
		return failure(request.ID, rpcErr.Code, rpcErr.Message)
	}
	defer d.finish(request.ID)

	return d.dispatch(request)
}

func (d *Dispatcher) Close() {
	d.mu.Lock()
	d.closing = true
	d.mu.Unlock()
}

func (d *Dispatcher) begin(id uint32) *RPCError {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closing {
		return rpcError("SHUTTING_DOWN", "The native host is shutting down.")
	}
	if _, exists := d.inflight[id]; exists {
		return rpcError("DUPLICATE_REQUEST_ID", "The request identifier is already in flight.")
	}
	if len(d.inflight) >= MaxInflight {
		return rpcError("TOO_MANY_REQUESTS", "The native request limit has been reached.")
	}
	d.inflight[id] = struct{}{}
	return nil
}

func (d *Dispatcher) finish(id uint32) {
	d.mu.Lock()
	delete(d.inflight, id)
	d.mu.Unlock()
}

func (d *Dispatcher) dispatch(request Request) Response {
	permission, known := methodPermission(request.Method)
	if !known {
		return failure(request.ID, "METHOD_NOT_FOUND", "The native method is not available.")
	}
	if _, granted := d.permissions[permission]; !granted {
		return failure(request.ID, "PERMISSION_DENIED", "The native method permission is not granted.")
	}
	if err := requireEmptyParams(request.Params); err != nil {
		return failure(request.ID, "INVALID_PARAMS", err.Error())
	}

	var (
		result any
		err    error
	)
	switch request.Method {
	case "app.getInfo":
		result = d.identity
	case "window.getState":
		result, err = d.window.State()
	case "window.minimize":
		err = d.window.Minimize()
	case "window.maximize":
		err = d.window.Maximize()
	case "window.restore":
		err = d.window.Restore()
	case "window.close":
		err = d.window.Close()
	}
	if err != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	if result == nil {
		result = json.RawMessage("null")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: result}
}

func methodPermission(method string) (string, bool) {
	switch method {
	case "app.getInfo":
		return PermissionAppInfo, true
	case "window.getState", "window.minimize", "window.maximize", "window.restore", "window.close":
		return PermissionWindow, true
	default:
		return "", false
	}
}

func decodeRequest(raw json.RawMessage) (Request, *RPCError) {
	if len(raw) == 0 || len(raw) > MaxRequestBytes {
		return Request{}, rpcError("PAYLOAD_TOO_LARGE", "The native request payload is outside the allowed size.")
	}
	if err := validateJSONShape(raw); err != nil {
		return Request{}, rpcError("INVALID_REQUEST", "The native request is malformed.")
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Request{}, rpcError("INVALID_REQUEST", "The native request is malformed.")
	}
	if request.Version != Version {
		return Request{}, rpcError("UNSUPPORTED_VERSION", "The native protocol version is unsupported.")
	}
	if request.ID == 0 {
		return Request{}, rpcError("INVALID_REQUEST", "The request identifier must be positive.")
	}
	if strings.TrimSpace(request.Method) == "" {
		return Request{}, rpcError("INVALID_REQUEST", "The native method is required.")
	}
	if len(request.Params) == 0 || request.Params[0] != '{' {
		return Request{}, rpcError("INVALID_PARAMS", "Native method parameters must be an object.")
	}
	return request, nil
}

func validateJSONShape(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	if depth >= MaxNestingDepth {
		return errors.New("JSON nesting limit exceeded")
	}

	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			keys[key] = struct{}{}
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}

func requireEmptyParams(raw json.RawMessage) error {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		return errors.New("Native method parameters are malformed.")
	}
	if len(params) != 0 {
		return errors.New("This native method does not accept parameters.")
	}
	return nil
}

func failure(id uint32, code, message string) Response {
	return Response{Version: Version, ID: id, OK: false, Error: rpcError(code, message)}
}

func rpcError(code, message string) *RPCError {
	return &RPCError{Code: code, Message: message}
}
