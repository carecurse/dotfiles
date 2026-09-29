package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sandboxHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	dryRun = false
	return h
}

func TestEnsureBashrcAppendsIdempotently(t *testing.T) {
	h := sandboxHome(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "bash"), 0755); err != nil {
		t.Fatal(err)
	}
	fragment := filepath.Join(repo, "bash", "bashrc")
	if err := os.WriteFile(fragment, []byte("# fragment\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ensureBashrc(repo, h)
	ensureBashrc(repo, h) // second run must not duplicate

	content, err := os.ReadFile(filepath.Join(h, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(content), "source \""+fragment+"\""); count != 1 {
		t.Fatalf("fragment sourced %d times, want 1:\n%s", count, content)
	}
}

func TestEnsureBashrcLocalCreatesTemplateOnce(t *testing.T) {
	h := sandboxHome(t)
	ensureBashrcLocal(h)
	ensureBashrcLocal(h)

	path := filepath.Join(h, ".bashrc.local")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(content), "ANTHROPIC_API_KEY"); count != 1 {
		t.Fatalf("template written %d times, want 1:\n%s", count, content)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want 600", fi.Mode().Perm())
	}

	// Existing file (e.g. with a real key) must be left untouched.
	custom := "export ANTHROPIC_API_KEY=\"sk-test\"\n"
	if err := os.WriteFile(path, []byte(custom), 0600); err != nil {
		t.Fatal(err)
	}
	ensureBashrcLocal(h)
	after, _ := os.ReadFile(path)
	if string(after) != custom {
		t.Fatalf("existing file modified:\n%s", after)
	}
}

func TestEnsureSymlinkLinksOnce(t *testing.T) {
	h := sandboxHome(t)
	src := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(src, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(h, ".ssh", "config")

	ensureSymlink(src, dst, 0700)
	ensureSymlink(src, dst, 0700)

	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	if target != src {
		t.Fatalf("link target = %q, want %q", target, src)
	}

	// A real existing file must never be clobbered.
	os.Remove(dst)
	if err := os.WriteFile(dst, []byte("mine"), 0644); err != nil {
		t.Fatal(err)
	}
	ensureSymlink(src, dst, 0700)
	content, _ := os.ReadFile(dst)
	if string(content) != "mine" {
		t.Fatal("existing file was clobbered")
	}
}
