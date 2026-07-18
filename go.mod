module github.com/0disoft/actutum

go 1.26

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
)

require github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect

replace github.com/jchv/go-webview2 => ./third_party/go-webview2
