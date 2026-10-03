package ipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/0disoft/velox/internal/externalurl"
)

const (
	Version            = 1
	MaxRequestBytes    = 64 << 10
	MaxNestingDepth    = 16
	MaxInflight        = 64
	PermissionAppInfo  = "app.info"
	PermissionWindow   = "window.basic"
	PermissionExternal = "external.open"
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

type ExternalOpener interface{ Open(string) error }

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
	identity     Identity
	permissions  map[string]struct{}
	window       Window
	external     ExternalOpener
	clipboard    ClipboardWriter
	files        FileOpener
	saver        FileSaver
	folders      FolderAccess
	upload       *saveUpload
	uploadSerial uint32
	savePending  bool

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
		return failure(request.ID, rpcErr.Code, rpcErr.Message)
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
	d.upload = nil
	d.mu.Unlock()
	d.DropPreparedText()
	d.DropFolderTarget()
}

func (d *Dispatcher) SetExternalOpener(opener ExternalOpener) {
	d.mu.Lock()
	d.external = opener
	d.mu.Unlock()
}

func (d *Dispatcher) IsClosing() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closing
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
	if request.Method == "external.open" {
		return d.openExternal(request)
	}
	if request.Method == "clipboard.writeText" {
		return d.writeClipboard(request)
	}
	if permission == PermissionFileSave {
		return d.prepareSave(request)
	}
	if request.Method == "folder.release" {
		return d.releaseFolder(request)
	}
	if err := requireEmptyParams(request.Params); err != nil {
		return failure(request.ID, "INVALID_PARAMS", err.Error())
	}

	var (
		result any
		err    error
	)
	switch request.Method {
	case "file.openText":
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "File selection requires asynchronous dispatch.")
	case "folder.select", "folder.list", "folder.openText":
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "Folder selection requires asynchronous dispatch.")
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
	case "external.open":
		return PermissionExternal, true
	case "clipboard.writeText":
		return PermissionClipboardWrite, true
	case "file.openText":
		return PermissionFileOpen, true
	case "folder.select", "folder.list", "folder.release":
		return PermissionFolderRead, true
	case "folder.openText":
		return PermissionFolderReadText, true
	case "file.beginSave", "file.appendSave", "file.commitSave", "file.cancelSave", "file.commitSaveAs", "file.commitSaveTo", "file.releaseSaveTarget":
		return PermissionFileSave, true
	case "window.getState", "window.minimize", "window.maximize", "window.restore", "window.close":
		return PermissionWindow, true
	default:
		return "", false
	}
}

func (d *Dispatcher) openExternal(request Request) Response {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params) != 1 || params["url"] == nil {
		return failure(request.ID, "INVALID_PARAMS", "External-link parameters must contain only a URL string.")
	}
	var rawURL string
	if err := json.Unmarshal(params["url"], &rawURL); err != nil {
		return failure(request.ID, "INVALID_PARAMS", "External-link parameters must contain only a URL string.")
	}
	target, err := externalurl.Validate(rawURL)
	if err != nil {
		return failure(request.ID, "INVALID_PARAMS", externalurl.ErrInvalidURL.Error())
	}
	d.mu.Lock()
	opener := d.external
	d.mu.Unlock()
	if opener == nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	err = opener.Open(target)
	if errors.Is(err, externalurl.ErrBusy) {
		return failure(request.ID, "TOO_MANY_REQUESTS", externalurl.ErrBusy.Error())
	}
	if err != nil {
		return failure(request.ID, "NATIVE_OPERATION_FAILED", "The native operation failed.")
	}
	return Response{Version: Version, ID: request.ID, OK: true, Result: struct {
		Queued bool `json:"queued"`
	}{true}}
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
		return request, rpcError("UNSUPPORTED_VERSION", "The native protocol version is unsupported.")
	}
	if request.ID == 0 {
		return Request{}, rpcError("INVALID_REQUEST", "The request identifier must be positive.")
	}
	if strings.TrimSpace(request.Method) == "" {
		return request, rpcError("INVALID_REQUEST", "The native method is required.")
	}
	if len(request.Params) == 0 || request.Params[0] != '{' {
		return request, rpcError("INVALID_PARAMS", "Native method parameters must be an object.")
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
