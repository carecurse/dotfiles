#!/usr/bin/env bash
# install.sh — wire up dotfiles on a new machine
# Usage: bash install.sh [--dry-run]

set -uo pipefail

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

info()  { printf '  \033[34m%s\033[0m\n' "$*"; }
ok()    { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn()  { printf '  \033[33m!\033[0m %s\n' "$*"; }
err()   { printf '  \033[31m✗\033[0m %s\n' "$*"; }

run() {
    if $DRY_RUN; then
        printf '  [dry-run] %s\n' "$*"
    else
        "$@"
    fi
}

info "Dotfiles: $DOTFILES"
echo

# ~/.bashrc — real file; append source line if not already present
info "Configuring ~/.bashrc"
BASHRC_FRAGMENT="$DOTFILES/bash/bashrc"
BASHRC="$HOME/.bashrc"

if $DRY_RUN; then
    printf '  [dry-run] Ensure %s sources %s\n' "$BASHRC" "$BASHRC_FRAGMENT"
else
    if ! grep -qF "$BASHRC_FRAGMENT" "$BASHRC" 2>/dev/null; then
        printf '\n# Managed dotfiles config\n[ -f "%s" ] && source "%s"\n' \
            "$BASHRC_FRAGMENT" "$BASHRC_FRAGMENT" >> "$BASHRC"
        ok "Appended source line to $BASHRC"
    else
        ok "$BASHRC already sources dotfiles fragment"
    fi
fi

# ~/.bashrc.local — create template if missing
info "Checking ~/.bashrc.local"
LOCAL="$HOME/.bashrc.local"
if [[ ! -f "$LOCAL" ]]; then
    run "cat > '$LOCAL' <<'EOF'
# ~/.bashrc.local — local secrets, never committed
# export ANTHROPIC_API_KEY=\"\"
# export GITHUB_TOKEN=\"\"
EOF"
    ok "Created $LOCAL template"
else
    ok "$LOCAL already exists"
fi

# SSH config
info "Configuring ~/.ssh"
run mkdir -p "$HOME/.ssh"
run chmod 700 "$HOME/.ssh"
SSH_CONFIG_SRC="$DOTFILES/ssh/config"
SSH_CONFIG_DST="$HOME/.ssh/config"
if [[ -f "$SSH_CONFIG_SRC" ]]; then
    if [[ ! -e "$SSH_CONFIG_DST" ]]; then
        run ln -sf "$SSH_CONFIG_SRC" "$SSH_CONFIG_DST"
        ok "Linked ssh/config → ~/.ssh/config"
    else
        ok "~/.ssh/config already exists, skipping"
    fi
else
    warn "ssh/config not found in dotfiles, skipping"
fi

# Git config
info "Configuring git"
GIT_CONFIG_SRC="$DOTFILES/git/config"
GIT_CONFIG_DST="$HOME/.gitconfig"
if [[ -f "$GIT_CONFIG_SRC" ]]; then
    if [[ ! -e "$GIT_CONFIG_DST" ]]; then
        run ln -sf "$GIT_CONFIG_SRC" "$GIT_CONFIG_DST"
        ok "Linked git/config → ~/.gitconfig"
    else
        ok "~/.gitconfig already exists, skipping"
    fi
else
    warn "git/config not found in dotfiles, skipping"
fi

# Helix editor
info "Installing Helix"
install_helix() {
    # Prefer the distro package; fall back to the official AppImage/tarball if needed.
    if command -v hx >/dev/null 2>&1; then
        ok "Helix already installed ($(hx --version 2>&1 | head -1))"
    elif command -v apt-get >/dev/null 2>&1; then
        if run sudo apt-get install -y hx 2>/dev/null; then
            ok "Helix installed via apt"
        else
            err "Failed to install helix via apt (package 'hx')"
        fi
    elif command -v brew >/dev/null 2>&1; then
        if run brew install helix; then
            ok "Helix installed via Homebrew"
        else
            err "Failed to install helix via Homebrew"
        fi
    else
        warn "Cannot install Helix automatically — please install it manually: https://helix-editor.com"
        return
    fi
}

if $DRY_RUN; then
    printf '  [dry-run] Install helix editor\n'
else
    install_helix || warn "Helix installation failed; continuing with other setup..."
fi

# Helix config
info "Configuring Helix"
HELIX_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/helix"
HELIX_SRC_DIR="$DOTFILES/config/helix"

if $DRY_RUN; then
    printf '  [dry-run] Link %s → %s\n' "$HELIX_SRC_DIR" "$HELIX_CONFIG_DIR"
else
    if [[ -e "$HELIX_CONFIG_DIR" && ! -L "$HELIX_CONFIG_DIR" ]]; then
        backup="${HELIX_CONFIG_DIR}.bak.$(date +%Y%m%d%H%M%S)"
        warn "Backing up existing Helix config dir → $backup"
        mv "$HELIX_CONFIG_DIR" "$backup"
    elif [[ -L "$HELIX_CONFIG_DIR" ]]; then
        warn "Removing existing Helix config symlink"
        rm "$HELIX_CONFIG_DIR"
    fi
    mkdir -p "$(dirname "$HELIX_CONFIG_DIR")"
    ln -sf "$HELIX_SRC_DIR" "$HELIX_CONFIG_DIR"
    ok "Linked config/helix → $HELIX_CONFIG_DIR"
fi

# Go LSP — gopls
info "Installing gopls (Go LSP)"
if $DRY_RUN; then
    printf '  [dry-run] go install golang.org/x/tools/gopls@latest\n'
else
    if command -v go >/dev/null 2>&1; then
        if go install golang.org/x/tools/gopls@latest 2>/dev/null; then
            ok "gopls installed"
        else
            warn "gopls installation failed; continuing..."
        fi
    else
        warn "go not found — skipping gopls (install Go first, then run: go install golang.org/x/tools/gopls@latest)"
    fi
fi

# Byobu — enable for login sessions
info "Configuring byobu"
if command -v byobu-enable >/dev/null 2>&1; then
    if $DRY_RUN; then
        printf '  [dry-run] byobu-enable\n'
    else
        if byobu-enable 2>/dev/null; then
            ok "byobu enabled for login sessions"
        else
            warn "byobu-enable failed; continuing..."
        fi
    fi
else
    warn "byobu not found — skipping (install with: apt install byobu)"
fi

echo
ok "Done. Reload your shell: source ~/.bashrc"
$DRY_RUN && warn "Dry-run mode — no changes were made."
