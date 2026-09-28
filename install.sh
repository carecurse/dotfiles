#!/usr/bin/env bash
# install.sh — wire up dotfiles on a new machine
# Usage: bash install.sh [--dry-run]

set -euo pipefail

DOTFILES="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

info()  { printf '  \033[34m%s\033[0m\n' "$*"; }
ok()    { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn()  { printf '  \033[33m!\033[0m %s\n' "$*"; }

run() {
    if $DRY_RUN; then
        printf '  [dry-run] %s\n' "$*"
    else
        eval "$@"
    fi
}

backup_and_replace() {
    local target="$1"
    local source="$2"

    if [[ -e "$target" && ! -L "$target" ]]; then
        local backup="${target}.bak.$(date +%Y%m%d%H%M%S)"
        warn "Backing up $target → $backup"
        run mv "$target" "$backup"
    elif [[ -L "$target" ]]; then
        warn "Removing existing symlink $target"
        run rm "$target"
    fi

    run "grep -qF \"$source\" \"$target\" 2>/dev/null || echo 'source \"$source\"' >> \"$target\""
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

# Byobu — enable for login sessions
info "Configuring byobu"
if command -v byobu-enable >/dev/null 2>&1; then
    if $DRY_RUN; then
        printf '  [dry-run] byobu-enable\n'
    else
        byobu-enable
        ok "byobu enabled for login sessions"
    fi
else
    warn "byobu not found — skipping (install with: apt install byobu)"
fi

echo
ok "Done. Reload your shell: source ~/.bashrc"
$DRY_RUN && warn "Dry-run mode — no changes were made."
