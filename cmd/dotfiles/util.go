package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var dryRun bool

func info(format string, args ...any) {
	fmt.Printf("  \x1b[34m%s\x1b[0m\n", fmt.Sprintf(format, args...))
}

func ok(format string, args ...any) {
	fmt.Printf("  \x1b[32m✓\x1b[0m %s\n", fmt.Sprintf(format, args...))
}

func warn(format string, args ...any) {
	fmt.Printf("  \x1b[33m!\x1b[0m %s\n", fmt.Sprintf(format, args...))
}

func errMsg(format string, args ...any) {
	fmt.Printf("  \x1b[31m✗\x1b[0m %s\n", fmt.Sprintf(format, args...))
}

// dotfilesRoot locates the repo checkout: $DOTFILES, else the source tree
// this binary was built from (works under `go run`), else the working dir
// if it looks like the repo.
func dotfilesRoot() (string, error) {
	if d := os.Getenv("DOTFILES"); d != "" {
		return d, nil
	}
	_, file, _, ok := runtime.Caller(0)
	if ok {
		// file = <repo>/cmd/dotfiles/util.go
		candidate := filepath.Dir(filepath.Dir(filepath.Dir(file)))
		if looksLikeRepo(candidate) {
			return candidate, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil && looksLikeRepo(cwd) {
		return cwd, nil
	}
	return "", fmt.Errorf("cannot locate dotfiles repo (set $DOTFILES)")
}

func looksLikeRepo(dir string) bool {
	for _, p := range []string{"bash/bashrc", "cmd/dotfiles/main.go"} {
		if st, err := os.Stat(filepath.Join(dir, p)); err != nil || st.IsDir() {
			return false
		}
	}
	return true
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return h
}

func have(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// run executes name with args, streaming output. In dry-run it only prints.
func run(name string, args ...string) error {
	if dryRun {
		fmt.Printf("  [dry-run] %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runQuiet runs and returns combined output, hiding it unless it fails.
func runQuiet(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

// runTimeout runs with a timeout, streaming output.
func runTimeout(d time.Duration, name string, args ...string) error {
	if dryRun {
		fmt.Printf("  [dry-run] %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// maybeSudo prepends sudo when not running as root.
func maybeSudo(name string, args ...string) (string, []string) {
	if os.Geteuid() != 0 && have("sudo") {
		return "sudo", append([]string{name}, args...)
	}
	return name, args
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileContains(path, needle string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(b, []byte(needle))
}
