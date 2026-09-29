#!/bin/bash
# bootstrap.sh — Termux → Ubuntu → dotfiles full setup
# This script:
#   1. Installs proot-distro + Ubuntu (if not already done)
#   2. Inside Ubuntu: installs git, curl, golang
#   3. Clones the dotfiles repo (HTTPS, no SSH key needed yet)
#   4. Runs the Go installer
#
# Run this once from Termux:
#   curl -fsSL https://raw.githubusercontent.com/carecurse/dotfiles/main/bootstrap.sh | bash
# Or if you already have git:
#   git clone https://github.com/carecurse/dotfiles.git ~/dotfiles
#   bash ~/dotfiles/bootstrap.sh

set -eo pipefail

info()  { printf '  \033[34m%s\033[0m\n' "$*"; }
ok()    { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn()  { printf '  \033[33m!\033[0m %s\n' "$*"; }
err()   { printf '  \033[31m✗\033[0m %s\n' "$*"; }

# Detect context: are we in Termux, or already in proot-distro Ubuntu?
detect_context() {
    # Most reliable: check if /data/data/com.termux exists AND pkg/proot-distro work
    # If we're already in proot-distro, pkg will fail with "cannot run as root"
    if [ -d /data/data/com.termux ]; then
        # We're in the Termux environment, but check if pkg actually works
        if pkg list-installed >/dev/null 2>&1; then
            echo "termux"
            return
        else
            # In proot-distro (pkg doesn't work), but /data/data/com.termux visible
            echo "ubuntu"
            return
        fi
    fi
    # Check if we're in nested proot via cgroup (may not exist in all proot versions)
    if grep -q proot /proc/1/cgroup 2>/dev/null; then
        echo "ubuntu"
        return
    fi
    # Check for Ubuntu-specific markers
    if [ -f /etc/os-release ]; then
        if grep -q "^NAME=\"Ubuntu" /etc/os-release; then
            echo "ubuntu"
            return
        fi
    fi
    echo "unknown"
}

# Termux setup: install proot-distro, Ubuntu, then run this script inside Ubuntu
termux_setup() {
    info "Installing proot-distro + Ubuntu"
    pkg update && pkg upgrade -y || {
        err "pkg update failed"
        return 1
    }
    pkg install -y proot-distro git openssh curl || {
        err "pkg install failed"
        return 1
    }
    ok "proot-distro and prerequisites installed"

    if proot-distro list | grep -q "^ubuntu"; then
        ok "Ubuntu already installed"
    else
        info "Installing Ubuntu (this may take a minute)..."
        proot-distro install ubuntu || {
            err "proot-distro install ubuntu failed"
            return 1
        }
        ok "Ubuntu installed"
    fi

    info "Running bootstrap inside Ubuntu..."
    proot-distro login ubuntu -- bash -c "curl -fsSL https://raw.githubusercontent.com/carecurse/dotfiles/main/bootstrap.sh | bash"
}

# Ubuntu setup: clone repo + run Go installer
ubuntu_setup() {
    info "Installing Ubuntu prerequisites"
    apt update || {
        err "apt update failed"
        return 1
    }
    # golang = bootstrap toolchain, the Go installer then ensures latest
    # git curl ca-certificates = needed to clone and run
    apt install -y golang git curl ca-certificates || {
        err "apt install failed"
        return 1
    }
    ok "Prerequisites installed"

    DOTFILES="${HOME}/dotfiles"
    if [ -d "$DOTFILES" ]; then
        ok "Dotfiles already cloned at $DOTFILES"
    else
        info "Cloning dotfiles repo (HTTPS, no SSH key needed)"
        git clone https://github.com/carecurse/dotfiles.git "$DOTFILES" || {
            err "git clone failed"
            return 1
        }
        ok "Cloned to $DOTFILES"
    fi

    info "Running dotfiles installer"
    cd "$DOTFILES"
    go run ./cmd/dotfiles install || {
        err "install failed"
        return 1
    }
    ok "Installer completed successfully"

    info "Running doctor to verify"
    go run ./cmd/dotfiles doctor || {
        err "doctor found issues — review above"
        return 1
    }

    echo
    ok "Bootstrap complete!"
    info "Next steps:"
    echo "  1. Add your Anthropic API key to ~/.bashrc.local:"
    echo "       echo 'export ANTHROPIC_API_KEY=\"sk-ant-...\"' >> ~/.bashrc.local"
    echo "  2. Set up GitHub SSH (optional, for pushes to use SSH):"
    echo "       ssh-keygen -t ed25519 -C 'you@example.com' -f ~/.ssh/id_ed25519"
    echo "       eval \"\$(ssh-agent -s)\" && ssh-add ~/.ssh/id_ed25519"
    echo "       cat ~/.ssh/id_ed25519.pub   # paste into GitHub → Settings → SSH keys"
    echo "  3. Reload your shell and start vibecoding:"
    echo "       source ~/.bashrc"
    echo "       opencode ~/dotfiles   # or any other project"
}

# Main
context=$(detect_context)
case "$context" in
    termux)
        info "Detected Termux environment"
        termux_setup
        ;;
    ubuntu)
        info "Detected Ubuntu (proot-distro) environment"
        ubuntu_setup
        ;;
    *)
        err "Cannot detect environment (not Termux, not proot-distro Ubuntu)"
        echo
        echo "Run this script in one of these contexts:"
        echo "  • Termux on your phone (from F-Droid)"
        echo "  • Inside proot-distro Ubuntu (after 'proot-distro login ubuntu')"
        exit 1
        ;;
esac
