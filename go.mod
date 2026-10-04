module github.com/0disoft/velox

go 1.26.0

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	golang.org/x/sys v0.48.0
)

require (
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/jchv/go-webview2 => ./third_party/go-webview2
