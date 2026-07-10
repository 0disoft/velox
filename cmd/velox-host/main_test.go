package main

import "testing"

func TestFileURL(t *testing.T) {
	got := fileURL(`C:\apps\hello world\index.html`)
	want := "file:///C:/apps/hello%20world/index.html"
	if got != want {
		t.Fatalf("fileURL() = %q, want %q", got, want)
	}
}
