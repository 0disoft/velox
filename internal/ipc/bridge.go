package ipc

import _ "embed"

//go:embed bridge.js
var bridgeSource string

func BridgeSource() string {
	return bridgeSource
}
