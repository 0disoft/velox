//go:build windows

package webview2

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
)

// DeferredResult starts on the UI message queue after the WebView callback
// returns. Complete must be called on the UI thread; only its first call wins.
type DeferredResult struct {
	Start func(complete func(any, error))
}

func (w *webview) startDeferredBinding(d rpcMessage, result *DeferredResult) {
	session, err := hex.DecodeString(d.Session)
	if err != nil || len(session) != 16 || result == nil || result.Start == nil {
		w.completeBinding(d, nil, errors.New("invalid deferred binding"))
		return
	}
	var once sync.Once
	w.Dispatch(func() {
		w.m.Lock()
		closing := w.closing
		w.m.Unlock()
		if !closing {
			result.Start(func(value any, err error) {
				once.Do(func() { w.completeBinding(d, value, err) })
			})
		}
	})
}

func (w *webview) completeBinding(d rpcMessage, value any, err error) {
	method, body := "resolve", "null"
	if err == nil {
		var encoded []byte
		encoded, err = json.Marshal(value)
		body = string(encoded)
	}
	if err != nil {
		method, body = "reject", jsString(err.Error())
	}
	id := strconv.Itoa(d.ID)
	guard := "window._rpc && window._rpc[" + id + "]"
	if d.Session != "" {
		guard += " && window._rpc.session === " + jsString(d.Session)
	}
	w.dispatchBindingResponse("if (" + guard + ") { try { window._rpc[" + id + "]." + method + "(" + body + ") } finally { delete window._rpc[" + id + "] } }")
}
