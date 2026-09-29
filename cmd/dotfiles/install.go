package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// aptPackages are ensured on Debian/Ubuntu before anything else.
var aptPackages = []string{"git", "curl", "ca-certificates", "byobu", "tmux", "sudo"}

func install(dry bool) error {
	dryRun = dry
	repo, err := dotfilesRoot()
	if err != nil {
		return err
	}
	h := home()

	info("Dotfiles: %s", repo)
	fmt.Println()

	ensureAptPackages()
	ensureBashrc(repo, h)
	ensureBashrcLocal(h)
	ensureSymlink(filepath.Join(repo, "ssh", "config"), filepath.Join(h, ".ssh", "config"), 0700)
	ensureSymlink(filepath.Join(repo, "git", "config"), filepath.Join(h, ".gitconfig"), 0)
	ensureHelix(repo, h)
	ensureGo()
	ensureGopls()
	ensureOpencode(repo, h)
	ensureAnthropicKey()
	ensureByobu()

	fmt.Println()
	if dryRun {
		warn("Dry-run mode — no changes were made.")
	} else {
		ok("Done. Reload your shell: source ~/.bashrc")
	}
	return nil
}

func ensureAptPackages() {
	info("Installing prerequisites (apt)")
	if !have("apt-get") {
		warn("apt-get not found — install git, curl, byobu, tmux yourself")
		return
	}
	name, args := maybeSudo("apt-get", "update")
	if err := run(name, args...); err != nil {
		warn("apt-get update failed; continuing...")
		return
	}
	missing := []string{}
	for _, p := range aptPackages {
		if out, err := runQuiet("dpkg-query", "-W", "-f=${Status}", p); err != nil || !strings.Contains(out, "install ok installed") {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		ok("Prerequisites already installed")
		return
	}
	name, args = maybeSudo("apt-get", append([]string{"install", "-y"}, missing...)...)
	if err := run(name, args...); err != nil {
		warn("apt install failed; continuing...")
		return
	}
	ok("Installed: %s", strings.Join(missing, ", "))
}

func ensureBashrc(repo, h string) {
	info("Configuring ~/.bashrc")
	fragment := filepath.Join(repo, "bash", "bashrc")
	rc := filepath.Join(h, ".bashrc")
	line := fmt.Sprintf("[ -f %q ] && source %q", fragment, fragment)
	if fileContains(rc, fragment) {
		ok("%s already sources dotfiles fragment", rc)
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] Append source line to %s\n", rc)
		return
	}
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		warn("cannot write %s: %v", rc, err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n# Managed dotfiles config\n%s\n", line)
	ok("Appended source line to %s", rc)
}

const bashrcLocalTemplate = `# ~/.bashrc.local — local secrets, never committed
# export ANTHROPIC_API_KEY=""
# export GITHUB_TOKEN=""
`

func ensureBashrcLocal(h string) {
	info("Checking ~/.bashrc.local")
	local := filepath.Join(h, ".bashrc.local")
	if fileExists(local) {
		ok("%s already exists", local)
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] Create %s template\n", local)
		return
	}
	if err := os.WriteFile(local, []byte(bashrcLocalTemplate), 0600); err != nil {
		warn("cannot create %s: %v", local, err)
		return
	}
	ok("Created %s template (mode 0600)", local)
}

// ensureSymlink links src -> dst when dst does not exist yet.
// dirMode, when non-zero, ensures the parent dir exists with that mode.
func ensureSymlink(src, dst string, dirMode os.FileMode) {
	info("Linking %s", dst)
	if !fileExists(src) {
		warn("%s not found in dotfiles, skipping", src)
		return
	}
	if fileExists(dst) || isSymlink(dst) {
		ok("%s already exists, skipping", dst)
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] ln -sf %s %s\n", src, dst)
		return
	}
	if dirMode != 0 {
		if err := os.MkdirAll(filepath.Dir(dst), dirMode); err != nil {
			warn("cannot create %s: %v", filepath.Dir(dst), err)
			return
		}
	} else if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		warn("cannot create %s: %v", filepath.Dir(dst), err)
		return
	}
	if err := os.Symlink(src, dst); err != nil {
		warn("cannot link %s: %v", dst, err)
		return
	}
	ok("Linked %s → %s", src, dst)
}

func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func ensureHelix(repo, h string) {
	info("Installing Helix")
	switch {
	case have("hx"):
		if out, err := runQuiet("hx", "--version"); err == nil {
			first := strings.SplitN(out, "\n", 2)[0]
			ok("Helix already installed (%s)", first)
		} else {
			ok("Helix already installed")
		}
	case have("apt-get"):
		name, args := maybeSudo("apt-get", "install", "-y", "hx")
		if err := run(name, args...); err != nil {
			errMsg("Failed to install Helix via apt (package 'hx')")
		} else {
			ok("Helix installed via apt")
		}
	case have("brew"):
		if err := run("brew", "install", "helix"); err != nil {
			errMsg("Failed to install Helix via Homebrew")
		} else {
			ok("Helix installed via Homebrew")
		}
	default:
		warn("Cannot install Helix automatically — https://helix-editor.com")
	}

	info("Configuring Helix")
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(h, ".config")
	}
	dst := filepath.Join(xdg, "helix")
	src := filepath.Join(repo, "config", "helix")
	if dryRun {
		fmt.Printf("  [dry-run] Link %s → %s\n", src, dst)
	} else {
		if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink == 0 {
			backup := dst + ".bak." + time.Now().Format("20060102150405")
			warn("Backing up existing Helix config dir → %s", backup)
			if err := os.Rename(dst, backup); err != nil {
				warn("backup failed: %v", err)
			}
		} else if err == nil {
			warn("Removing existing Helix config symlink")
			os.Remove(dst)
		}
		os.MkdirAll(filepath.Dir(dst), 0755)
		if err := os.Symlink(src, dst); err != nil && !isSymlink(dst) {
			warn("cannot link Helix config: %v", err)
		} else {
			ok("Linked config/helix → %s", dst)
		}
	}

	info("Building Helix tree-sitter grammars")
	if dryRun {
		fmt.Println("  [dry-run] hx --grammar build go")
		return
	}
	if !have("hx") {
		warn("helix not found — skipping grammar build")
		return
	}
	if err := runTimeout(5*time.Minute, "hx", "--grammar", "build", "go"); err != nil {
		warn("Go grammar build failed; run manually: hx --grammar build go")
		return
	}
	ok("Go tree-sitter grammar built")
}

var goVersionRe = regexp.MustCompile(`go\d+\.\d+(?:\.\d+)?`)

func ensureGo() {
	info("Installing Go")
	if dryRun {
		fmt.Println("  [dry-run] Install latest Go from go.dev")
		return
	}
	current, _ := runQuiet("go", "version")
	currentVer := goVersionRe.FindString(current)
	latestVer := latestGoVersion()
	if latestVer == "" {
		warn("Could not determine latest Go version from go.dev")
		return
	}
	if currentVer == latestVer {
		ok("Go %s already installed", latestVer)
		return
	}
	arch := map[string]string{"amd64": "amd64", "arm64": "arm64", "386": "386"}[runtime.GOARCH]
	if arch == "" {
		errMsg("Unsupported architecture: %s", runtime.GOARCH)
		return
	}
	tarball := fmt.Sprintf("%s.linux-%s.tar.gz", latestVer, arch)
	url := "https://go.dev/dl/" + tarball
	info("Downloading %s from go.dev...", tarball)
	tmp, err := os.CreateTemp("", "go-*.tar.gz")
	if err != nil {
		warn("cannot create temp file: %v", err)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := downloadFile(url, tmp); err != nil {
		tmp.Close()
		warn("Failed to download Go: %v", err)
		return
	}
	tmp.Close()
	info("Extracting to /usr/local...")
	name, args := maybeSudo("tar", "-C", "/usr/local", "-xzf", tmpName)
	if err := run(name, args...); err != nil {
		errMsg("Failed to extract Go tarball")
		return
	}
	ok("Go %s installed successfully", latestVer)
}

func latestGoVersion() string {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://go.dev/dl/")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}
	return goVersionRe.FindString(string(body))
}

func downloadFile(url string, w io.Writer) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func ensureGopls() {
	info("Installing gopls (Go LSP)")
	goBin, err := goBinary()
	if err != nil {
		warn("go not found — skipping gopls")
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] %s install golang.org/x/tools/gopls@latest\n", goBin)
		return
	}
	cmd := exec.Command(goBin, "install", "golang.org/x/tools/gopls@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Hide failures only as much as the old script did: report, don't stop.
	if err := cmd.Run(); err != nil {
		warn("gopls installation failed; continuing...")
		return
	}
	ok("gopls installed")
}

func goBinary() (string, error) {
	if p, err := exec.LookPath("go"); err == nil {
		return p, nil
	}
	if fileExists("/usr/local/go/bin/go") {
		return "/usr/local/go/bin/go", nil
	}
	return "", fmt.Errorf("go not found")
}

func ensureOpencode(repo, h string) {
	info("Installing opencode")
	if have("opencode") {
		if out, err := runQuiet("opencode", "--version"); err == nil {
			ok("opencode already installed (%s)", strings.SplitN(out, "\n", 2)[0])
		} else {
			ok("opencode already installed")
		}
	} else {
		if !have("curl") {
			warn("curl missing — cannot run opencode installer")
		} else if err := run("sh", "-c", "curl -fsSL https://opencode.ai/install | bash"); err != nil {
			warn("opencode install failed; retry manually: curl -fsSL https://opencode.ai/install | bash")
		} else {
			ok("opencode installed")
		}
	}

	info("Configuring opencode")
	dstDir := filepath.Join(h, ".config", "opencode")
	dst := filepath.Join(dstDir, "opencode.json")
	ensureSymlink(filepath.Join(repo, "config", "opencode", "opencode.json"), dst, 0755)
}

func ensureAnthropicKey() {
	info("Checking Anthropic credentials")
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		ok("ANTHROPIC_API_KEY is set")
		return
	}
	warn("ANTHROPIC_API_KEY is not set — add it to ~/.bashrc.local, then run: opencode")
}

func ensureByobu() {
	info("Configuring byobu")
	if dryRun {
		fmt.Println("  [dry-run] byobu-enable")
		return
	}
	if !have("byobu-enable") {
		warn("byobu not found — it should have been installed with prerequisites")
		return
	}
	if err := run("byobu-enable"); err != nil {
		warn("byobu-enable failed; continuing...")
		return
	}
	ok("byobu enabled for login sessions")
}
