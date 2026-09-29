package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// doctor checks machine state and reports gaps. It never changes anything.
// Returns true when everything is fine.
func doctor() bool {
	repo, err := dotfilesRoot()
	if err != nil {
		errMsg(err.Error())
		return false
	}
	h := home()
	fragment := filepath.Join(repo, "bash", "bashrc")
	failures := 0
	check := func(name string, good bool, detail string) {
		if good {
			fmt.Printf("  \x1b[32m✓\x1b[0m %-28s %s\n", name, detail)
		} else {
			fmt.Printf("  \x1b[31m✗\x1b[0m %-28s %s\n", name, detail)
			failures++
		}
	}

	info("Tools")
	for _, t := range []struct {
		bin  string
		args []string
	}{
		{"git", []string{"--version"}},
		{"curl", []string{"--version"}},
		{"hx", []string{"--version"}},
		{"go", []string{"version"}},
		{"gopls", []string{"version"}},
		{"cargo", []string{"--version"}},
		{"opencode", []string{"--version"}},
	} {
		if !have(t.bin) {
			check(t.bin, false, "not installed")
			continue
		}
		out, err := runQuiet(t.bin, t.args...)
		ver := ""
		if err == nil && out != "" {
			ver = strings.SplitN(out, "\n", 2)[0]
			if len(ver) > 60 {
				ver = ver[:60]
			}
		}
		check(t.bin, err == nil, ver)
	}
	
	// rust-analyzer is in ~/.cargo/bin which may not be in PATH
	raPath := filepath.Join(h, ".cargo", "bin", "rust-analyzer")
	raExists := fileExists(raPath)
	if raExists {
		out, err := runQuiet(raPath, "--version")
		ver := ""
		if err == nil && out != "" {
			ver = strings.SplitN(out, "\n", 2)[0]
			if len(ver) > 60 {
				ver = ver[:60]
			}
		}
		check("rust-analyzer", err == nil, ver)
	} else {
		check("rust-analyzer", false, "not installed")
	}

	info("Dotfiles wiring")
	check("~/.bashrc sources fragment", fileContains(filepath.Join(h, ".bashrc"), fragment), fragment)
	local := filepath.Join(h, ".bashrc.local")
	check("~/.bashrc.local exists", fileExists(local), local)
	check("~/.gitconfig symlink", isSymlink(filepath.Join(h, ".gitconfig")), "→ git/config")
	check("~/.ssh/config symlink", isSymlink(filepath.Join(h, ".ssh", "config")), "→ ssh/config")
	helixDst := filepath.Join(xdgConfig(h), "helix")
	check("helix config symlink", isSymlink(helixDst), "→ config/helix")
	check("opencode.json symlink", isSymlink(filepath.Join(xdgConfig(h), "opencode", "opencode.json")), "→ config/opencode/opencode.json")

	info("Sessions & credentials")
	// Never print the key itself — presence only.
	check("ANTHROPIC_API_KEY set", os.Getenv("ANTHROPIC_API_KEY") != "", "env (value never shown)")

	fmt.Println()
	if failures > 0 {
		errMsg("doctor found %d problem(s) — run: go run ./cmd/dotfiles install", failures)
		return false
	}
	ok("doctor: everything looks good")
	return true
}

func xdgConfig(h string) string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg
	}
	return filepath.Join(h, ".config")
}
