package externalurl

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateHTTPSURLs(t *testing.T) {
	for _, value := range []string{
		"https://example.com", "HTTPS://Example.COM/docs?q=1#part", "https://localhost:8443/a%20b",
		"https://127.0.0.1/", "https://[::1]:443/", "https://xn--3e0b707e.example/%ED%95%9C%EA%B8%80",
	} {
		got, err := Validate(value)
		if err != nil || !strings.HasPrefix(got, "https://") {
			t.Fatalf("Validate(%q)=%q, %v", value, got, err)
		}
	}
	for _, value := range []string{
		"", "http://example.com", "file:///C:/test", "javascript:alert(1)", "mailto:test@example.com",
		"cmd.exe /c start https://example.com", "https:example.com", "//example.com", " https://example.com",
		"https://example.com ", "https://user:secret@example.com", `https://example.com\file`,
		"https://example.com/%5cfile", "https://example.com/%00", "https://example.com/%0d%0a",
		"https://example.com/?bad=%xx", "https://example.com\n", `https://example.com/"arg`,
		"https://example.com:0", "https://example.com:65536", "https://example.com:", "https://example.com:bad",
		"https://-bad.example/", "https://bad..example/", "https://[invalid]/", "https://[fe80::1%25zone]/",
		"https://example.com/" + strings.Repeat("a", MaxURLBytes), "https://example.com/한글", "https://한글.example/",
	} {
		if _, err := Validate(value); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("accepted %q: %v", value, err)
		}
	}
}

func TestOpenerRequiresApprovalAndClearsBusy(t *testing.T) {
	confirmations, launches := 0, 0
	approved := false
	opener := &Opener{Confirm: func(target string) (bool, error) { confirmations++; return approved, nil }, Launch: func(target string) error { launches++; return nil }}
	if _, err := opener.Open("file:///C:/test"); !errors.Is(err, ErrInvalidURL) || confirmations != 0 {
		t.Fatal("invalid URI reached confirmation")
	}
	if opened, err := opener.Open("https://example.com"); opened || err != nil || launches != 0 {
		t.Fatal("cancel opened browser")
	}
	approved = true
	if opened, err := opener.Open("https://example.com"); !opened || err != nil || launches != 1 {
		t.Fatalf("approved=%t, %v", opened, err)
	}
	opener.Confirm = func(string) (bool, error) {
		if _, err := opener.Open("https://example.com"); !errors.Is(err, ErrBusy) {
			t.Fatal("nested confirmation was not rejected")
		}
		return true, nil
	}
	opener.Launch = func(string) error { return errors.New("private native details") }
	if opened, err := opener.Open("https://example.com"); opened || err == nil {
		t.Fatal("native failure reported opened")
	}
	opener.IsClosing = func() bool { return true }
	opener.Launch = func(string) error { t.Fatal("closing during approval launched browser"); return nil }
	if opened, err := opener.Open("https://example.com"); opened || err != nil {
		t.Fatal("close during approval was not cancelled")
	}
}

func TestSchedulerDefersBoundsAndCancelsConfirmation(t *testing.T) {
	var queued func()
	closed := false
	confirmations, launches, failures := 0, 0, 0
	approved := false
	opener := &Opener{Confirm: func(string) (bool, error) { confirmations++; return approved, nil }, Launch: func(string) error { launches++; return nil }}
	scheduler := &Scheduler{Dispatch: func(fn func()) { queued = fn }, IsClosing: func() bool { return closed }, Opener: opener, NotifyFailure: func() { failures++ }}
	if err := scheduler.Open("https://example.com"); err != nil || confirmations != 0 {
		t.Fatal("confirmation ran synchronously")
	}
	if err := scheduler.Open("https://example.com"); !errors.Is(err, ErrBusy) {
		t.Fatal("queued duplicate accepted")
	}
	queued()
	if confirmations != 1 || launches != 0 {
		t.Fatal("cancel launched")
	}
	approved = true
	if err := scheduler.Open("https://example.com"); err != nil {
		t.Fatal(err)
	}
	closed = true
	queued()
	if confirmations != 1 || launches != 0 {
		t.Fatal("shutdown reached confirmation")
	}
	closed = false
	opener.Launch = func(string) error { return errors.New("failed") }
	if err := scheduler.Open("https://example.com"); err != nil {
		t.Fatal(err)
	}
	queued()
	if failures != 1 {
		t.Fatal("deferred failure was not reported")
	}
	if err := scheduler.Open("https://example.com"); err != nil {
		t.Fatal("failed launch retained slot")
	}
	queued()
}
