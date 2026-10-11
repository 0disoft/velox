package webview2

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNativeManagedProfileStorage(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_STORAGE_PROBE") != "1" {
		t.Skip("opt-in real WebView2 storage through the profile-link manager")
	}
	_, profile, root := profileLinkFixture(t)
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer link.Close()
	for _, mode := range []string{"write", "read"} {
		result, exited := runStorageProbeChild(t, profile, link.path, mode)
		if !exited || result.Status != "verified" {
			t.Fatalf("managed storage %s: %+v exited=%v", mode, result, exited)
		}
		t.Logf("managed native %s: profile-length=%d browser-path-length=%d", mode, result.ProfileLength, result.BrowserPathLength)
	}
}

func profileLinkFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root, err := os.MkdirTemp("", "vpl-")
	if err != nil {
		t.Fatal(err)
	}
	// All links in this fixture point to sibling private fixture directories.
	t.Cleanup(func() {
		entries, _ := os.ReadDir(filepath.Join(root, "links"))
		for _, entry := range entries {
			path := filepath.Join(root, "links", entry.Name())
			name, _ := windows.UTF16PtrFromString(browserDataPath(path))
			h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
				nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
			if err != nil {
				t.Errorf("open fixture link for cleanup: %v", err)
				return
			}
			var info windows.ByHandleFileInformation
			err = windows.GetFileInformationByHandle(h, &info)
			windows.CloseHandle(h)
			if err != nil {
				t.Error(err)
				return
			}
			if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
				if err := os.Remove(path); err != nil {
					t.Error(err)
					return
				}
			}
		}
		if filepath.Dir(root) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(root), "vpl-") {
			t.Error("unsafe fixture cleanup path")
			return
		}
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	profile := filepath.Join(root, strings.Repeat("long-profile-", 12))
	if err := os.Mkdir(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "marker.txt"), []byte("original fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, profile, filepath.Join(root, "links")
}

func TestProfileLinkPreservesIdentityAndPinsMutation(t *testing.T) {
	_, profile, root := profileLinkFixture(t)
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer link.Close()
	original, _ := os.Stat(profile)
	alias, err := os.Stat(link.path)
	if err != nil || !os.SameFile(original, alias) {
		t.Fatalf("different profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(link.path, "written.txt"), []byte("same data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(profile, "written.txt")); err != nil || string(data) != "same data" {
		t.Fatalf("profile write did not reach original: %v", err)
	}
	if err := os.Rename(profile, profile+"-renamed"); err == nil {
		t.Fatal("live original profile was renamed")
	}
	if err := os.Remove(link.path); err == nil {
		t.Fatal("live profile link was removed")
	}
	if err := os.Rename(root, root+"-renamed"); err == nil {
		t.Fatal("live link root was renamed")
	}
	name, _ := windows.UTF16PtrFromString(link.path)
	h, err := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err == nil {
		windows.CloseHandle(h)
		t.Fatal("live junction allowed write/reparse mutation")
	}
	rootName, _ := windows.UTF16PtrFromString(root)
	rootWriter, err := windows.CreateFile(rootName, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	mutateErr := setProfileJunctionTarget(rootWriter, browserDataPath(profile))
	windows.CloseHandle(rootWriter)
	if mutateErr == nil {
		t.Fatal("nonempty pinned link root became a reparse point")
	}
	if err := link.Close(); err != nil {
		t.Fatal(err)
	}
	if err := link.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(link.path); err != nil {
		t.Fatal("closing lease removed persistent path", err)
	}
	if err := os.Remove(link.path); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(profile, "marker.txt")); err != nil || string(data) != "original fixture" {
		t.Fatalf("link removal changed profile: %v", err)
	}
}

func TestProfileLinkReusesStableConcurrentIdentity(t *testing.T) {
	_, profile, root := profileLinkFixture(t)
	const count = 6
	links := make([]*profileLink, count)
	errors := make([]error, count)
	var wait sync.WaitGroup
	for i := range links {
		wait.Add(1)
		go func(i int) { defer wait.Done(); links[i], errors[i] = acquireProfileLink(profile, root) }(i)
	}
	wait.Wait()
	for i, link := range links {
		if link != nil {
			defer link.Close()
		}
		if errors[i] != nil {
			t.Errorf("acquire %d: %v", i, errors[i])
		}
	}
	if t.Failed() {
		return
	}
	for _, link := range links {
		if link.path != links[0].path {
			t.Fatal("concurrent instances selected different browser paths")
		}
	}
	if err := links[0].Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(links[1].path); err == nil {
		t.Fatal("closing one lease unpinned another")
	}
	for _, link := range links {
		if err := link.Close(); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || strings.HasPrefix(entries[0].Name(), "tmp-") {
		t.Fatalf("publication left unexpected entries: %v %v", entries, err)
	}
	reopened, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.path != links[0].path {
		t.Fatal("relaunch changed browser path")
	}
}

func TestProfileLinkAllowsOtherProfilesWithoutRelaxingTheFirstLease(t *testing.T) {
	fixture, profile, root := profileLinkFixture(t)
	first, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	secondProfile := filepath.Join(fixture, "second-profile")
	if err := os.Mkdir(secondProfile, 0o700); err != nil {
		t.Fatal(err)
	}
	second, err := acquireProfileLink(secondProfile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.path == second.path {
		t.Fatal("different profiles shared an alias")
	}
	if err := os.Remove(first.path); err == nil {
		t.Fatal("publishing another profile unpinned the first")
	}
	if err := os.WriteFile(filepath.Join(second.path, "second.txt"), []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(profile, "second.txt")); !os.IsNotExist(err) {
		t.Fatal("second-profile write crossed identities")
	}
}

func TestProfileLinkAcrossProcesses(t *testing.T) {
	_, profile, root := profileLinkFixture(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestProfileLinkChild$", "-test.v")
	cmd.Env = append(os.Environ(), "VELOX_PROFILE_LINK_CHILD_PROFILE="+profile, "VELOX_PROFILE_LINK_CHILD_ROOT="+root)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		input.Close()
		if !waited {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	ready := false
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		if scanner.Text() == "VELOX_PROFILE_LINK_READY" {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatalf("child lease not ready: %v", scanner.Err())
	}
	parent, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	path := parent.path
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err == nil {
		t.Fatal("parent release removed child's protection")
	}
	if err := os.Rename(profile, profile+"-moved"); err == nil {
		t.Fatal("child lease did not pin source identity")
	}
	if _, err := input.Write([]byte("release\n")); err != nil {
		t.Fatal(err)
	}
	input.Close()
	for scanner.Scan() {
	}
	err = cmd.Wait()
	waited = true
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal("process exit did not release link handle", err)
	}
	if data, err := os.ReadFile(filepath.Join(profile, "marker.txt")); err != nil || string(data) != "original fixture" {
		t.Fatal("child changed profile data", err)
	}
}

func TestProfileLinkChild(t *testing.T) {
	profile := os.Getenv("VELOX_PROFILE_LINK_CHILD_PROFILE")
	if profile == "" {
		t.Skip("private cross-process link lease helper")
	}
	root := os.Getenv("VELOX_PROFILE_LINK_CHILD_ROOT")
	fixture := filepath.Dir(root)
	if filepath.Dir(profile) != fixture || filepath.Dir(fixture) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(fixture), "vpl-") {
		t.Fatal("invalid private fixture")
	}
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer link.Close()
	fmt.Println("VELOX_PROFILE_LINK_READY")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() || scanner.Text() != "release" {
		t.Fatal("private child did not receive release")
	}
}

func TestProfileLinkFailedPublicationRemovesOnlyItsOwnStage(t *testing.T) {
	_, profile, root := profileLinkFixture(t)
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	defer link.Close()
	rootName, _ := windows.UTF16PtrFromString(root)
	h, err := windows.CreateFile(rootName, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	if err := publishProfileLink(h, "failed", `\\?\C:\`+strings.Repeat("x", 9000)); err == nil {
		t.Fatal("oversized target was published")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(link.path) {
		t.Fatalf("failed staging affected entries: %v %v", entries, err)
	}
	if data, err := os.ReadFile(filepath.Join(link.path, "marker.txt")); err != nil || string(data) != "original fixture" {
		t.Fatal("failed publication touched existing profile", err)
	}
}

func TestProfileLinkRejectsForeignRootAndOccupiedLink(t *testing.T) {
	_, profile, root := profileLinkFixture(t)
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "foreign.txt")
	if err := os.WriteFile(marker, []byte("foreign fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if link, err := acquireProfileLink(profile, root); err == nil {
		link.Close()
		t.Fatal("adopted a non-private existing root")
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "foreign fixture" {
		t.Fatal("foreign root changed", err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	alias := link.path
	if err := link.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(alias, 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(alias, "foreign.txt")
	if err := os.WriteFile(foreign, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if link, err := acquireProfileLink(profile, root); err == nil {
		link.Close()
		t.Fatal("overwrote occupied plain directory")
	}
	if data, err := os.ReadFile(foreign); err != nil || string(data) != "keep" {
		t.Fatal("occupied path changed", err)
	}
}

func TestProfileLinkRejectsRedirectedLinkAndLinkedAncestor(t *testing.T) {
	fixture, profile, root := profileLinkFixture(t)
	link, err := acquireProfileLink(profile, root)
	if err != nil {
		t.Fatal(err)
	}
	alias := link.path
	if err := link.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(fixture, "other")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	createStorageProbeJunction(t, strings.TrimPrefix(alias, `\\?\`), other)
	if link, err := acquireProfileLink(profile, root); err == nil {
		link.Close()
		t.Fatal("accepted redirected alias")
	}
	observed, _ := os.Stat(alias)
	foreign, _ := os.Stat(other)
	if !os.SameFile(observed, foreign) {
		t.Fatal("redirected link was rewritten")
	}
	parentAlias := filepath.Join(fixture, "parent-alias")
	createStorageProbeJunction(t, parentAlias, fixture)
	defer os.Remove(parentAlias)
	if link, err := acquireProfileLink(filepath.Join(parentAlias, filepath.Base(profile)), root); err == nil {
		link.Close()
		t.Fatal("accepted linked profile ancestor")
	}
}

func TestProfileLinkRejectsUnsupportedPathsWithoutCreation(t *testing.T) {
	fixture, profile, root := profileLinkFixture(t)
	for _, pair := range [][2]string{
		{`relative\profile`, root}, {`\\server\share\profile`, root}, {profile, `\\.\C:\links`},
		{profile, filepath.Join(profile, "links")},
		{profile, filepath.Join(fixture, strings.Repeat("r", 110))},
	} {
		if link, err := acquireProfileLink(pair[0], pair[1]); err == nil {
			link.Close()
			t.Fatalf("accepted unsupported path pair: %v", pair)
		}
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("rejected paths created the link root")
	}
}
