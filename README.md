# dotfiles

Personal configuration files managed with Git. All scripting is Go
(standard library only) — there is no bash installer.

## Structure

```
dotfiles/
├── bash/
│   └── bashrc           # Sourced fragment — custom config only (stays bash: it *is* shell config)
├── cmd/
│   └── dotfiles/        # Go CLI: `install` and `doctor` (stdlib only, `go run`)
├── config/
│   ├── helix/           # Helix config (symlinked → ~/.config/helix)
│   └── opencode/
│       └── opencode.json  # Minimal opencode config, no model pinned (symlinked if missing)
├── git/                 # gitconfig (symlinked → ~/.gitconfig)
├── ssh/                 # Non-secret SSH config (config, known_hosts.example)
├── go.mod               # Go module (no external dependencies)
└── README.md
```

### Model

`~/.bashrc` is a **real file** that stays on disk and is not tracked. It sources
the dotfiles fragment at the end:

```sh
# bottom of ~/.bashrc
[ -f "$HOME/dotfiles/bash/bashrc" ] && source "$HOME/dotfiles/bash/bashrc"
```

Secrets live in `~/.bashrc.local`, sourced from the fragment — never committed:

```sh
# ~/.bashrc.local
export ANTHROPIC_API_KEY=""
export GITHUB_TOKEN=""
```
## Bootstrap: Termux → Ubuntu → vibecode

Target: fresh phone → one command → ready to vibecode with opencode + Anthropic.

### Quick start

**From Termux (on your phone):**

```sh
curl -fsSL https://raw.githubusercontent.com/carecurse/dotfiles/main/bootstrap.sh | bash
```

That's it. The script will:

1. Install `proot-distro` and Ubuntu (if not already done)
2. Enter Ubuntu and install the bootstrap toolchain (`golang`, `git`, `curl`)
3. Clone the dotfiles repo over HTTPS
4. Run the Go installer to wire everything up
5. Run the doctor to verify everything
6. Print the next steps (add Anthropic key, optional SSH setup)

If you already have git and proot-distro Ubuntu installed, you can skip the first part:

```sh
proot-distro login ubuntu
curl -fsSL https://raw.githubusercontent.com/carecurse/dotfiles/main/bootstrap.sh | bash
```

Or clone the repo first and run locally:

```sh
git clone https://github.com/carecurse/dotfiles.git ~/dotfiles
bash ~/dotfiles/bootstrap.sh
```

---

## Deep dive: what bootstrap.sh does

### 1. Termux (on the phone)

Install Termux from F-Droid (not Play Store — it's stale).

The bootstrap script detects you're in Termux and:

- Updates and upgrades `pkg`
- Installs `proot-distro`, `git`, `openssh`, `curl`
- Checks if Ubuntu is installed; if not, installs it (`proot-distro install ubuntu`)
- Re-invokes itself inside Ubuntu via `proot-distro login ubuntu -- bash -c "curl ... | bash"`

### 2. Inside Ubuntu

The bootstrap script detects you're in proot-distro Ubuntu and:

- Runs `apt update && apt upgrade`
- Installs the bootstrap toolchain: `golang`, `git`, `curl`, `ca-certificates`
  (golang is *just* for `go run`; the Go installer ensures the latest version)
- Clones the dotfiles repo over HTTPS to `~/dotfiles` (no SSH key needed yet)
- Runs `go run ./cmd/dotfiles install` (the Go installer, see below)
- Runs `go run ./cmd/dotfiles doctor` to verify
- Prints next steps

### 3. Go installer + doctor

Once the bootstrap script gets you into Ubuntu with git + curl + golang:

```sh
# Preview changes without applying them
go run ./cmd/dotfiles install --dry-run

# Actually set up dotfiles
go run ./cmd/dotfiles install

# Verify everything
go run ./cmd/dotfiles doctor
```

`install` wires up:

- Bash config: appends source line to `~/.bashrc`, creates `~/.bashrc.local` template
- Symlinks: `ssh/config` → `~/.ssh/config`, `git/config` → `~/.gitconfig`,
  `config/helix` → `~/.config/helix`, `config/opencode/opencode.json` → `~/.config/opencode/`
- Tools: Helix (editor), Go (latest from go.dev), gopls (Go LSP), opencode (via official installer)
- Sessions: byobu autolaunch for login shells

`doctor` reports the status of all the above and exits with a fix hint if anything is missing.

---

## Post-bootstrap: Anthropic + optional SSH

### 1. Add your Anthropic API key

The key lives in `~/.bashrc.local` (mode 0600, never committed):

```sh
cat >> ~/.bashrc.local <<'EOF'
export ANTHROPIC_API_KEY="sk-ant-..."
EOF
chmod 600 ~/.bashrc.local
source ~/.bashrc
```

### 2. Start vibecoding

```sh
opencode ~/dotfiles
# or any project directory
```

### 3. Optional: GitHub SSH (for pushing)

If you want pushes to use SSH instead of HTTPS:

```sh
ssh-keygen -t ed25519 -C "you@example.com" -f ~/.ssh/id_ed25519
eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519
cat ~/.ssh/id_ed25519.pub   # paste into GitHub → Settings → SSH keys
ssh -T git@github.com       # verify: "Hi <username>! ..."
cd ~/dotfiles && git remote set-url origin git@github.com:carecurse/dotfiles.git
```

---

## Troubleshooting

**bootstrap.sh failed partway through:**

If the bootstrap script exits with an error, you're left with a partially-configured
machine but the repo is cloned. Manually pick up where it left off:

```sh
cd ~/dotfiles
go run ./cmd/dotfiles install --dry-run   # see what would be done
go run ./cmd/dotfiles install             # apply changes
go run ./cmd/dotfiles doctor              # verify
```

**doctor says something is missing:**

Run `go run ./cmd/dotfiles install` again — it's idempotent and will fix gaps.

**Still having issues?**

Check that you're inside proot-distro Ubuntu:

```sh
uname -a    # should show "Linux"
which apt   # should find /usr/bin/apt
```

If you're in Termux, enter Ubuntu first: `proot-distro login ubuntu`

---

## SSH Token Authentication — GitHub / GitLab (detailed reference)

This section documents the manual SSH setup in detail. The quick version is in
"Post-bootstrap: optional SSH" above.

### 1. Generate an Ed25519 key pair

```sh
ssh-keygen -t ed25519 -C "you@example.com" -f ~/.ssh/id_ed25519
```

- `-t ed25519` — modern elliptic-curve algorithm; prefer over RSA.
- `-C` — comment embedded in the public key (use your email or a label).
- `-f` — explicit output path; avoids overwriting existing keys.

Enter a strong passphrase when prompted. Do **not** leave it empty.

---

### 2. Start ssh-agent and load the private key

```sh
# Start the agent and export the socket into the current shell
eval "$(ssh-agent -s)"

# Add the private key (will prompt for passphrase)
ssh-add ~/.ssh/id_ed25519
```

To avoid repeating this on every login, add both lines to `~/.bashrc` (or a
dedicated `~/.ssh/agent.env` pattern). The agent PID is exported automatically
by `eval`.

---

### 3. Copy the public key

```sh
# Print to terminal — paste directly into the provider's UI
cat ~/.ssh/id_ed25519.pub
```

On systems with `xclip` or `xsel` available:

```sh
xclip -selection clipboard < ~/.ssh/id_ed25519.pub   # X11
xsel --clipboard --input < ~/.ssh/id_ed25519.pub      # alternative
```

On macOS:

```sh
pbcopy < ~/.ssh/id_ed25519.pub
```

#### Where to paste

| Provider | Path |
|----------|------|
| GitHub   | Settings → SSH and GPG keys → New SSH key |
| GitLab   | Preferences → SSH Keys → Add new key |

Set **Key type** to *Authentication Key* and give it a descriptive title.

---

### 4. Verify the connection

```sh
# GitHub
ssh -T git@github.com

# GitLab
ssh -T git@gitlab.com
```

Expected output (GitHub):

```
Hi <username>! You've successfully authenticated, but GitHub does not provide shell access.
```

Expected output (GitLab):

```
Welcome to GitLab, @<username>!
```

Exit code `1` with the welcome message is normal for both providers — it
indicates authentication succeeded even though interactive shell access is
denied.

---

### Hardening checklist

- [ ] Private key (`id_ed25519`) has a passphrase.
- [ ] Private key is chmod `600`; `~/.ssh` is chmod `700`.
- [ ] Public key (`.pub`) is the **only** file committed to this repo.
- [ ] Rotate keys if a machine is compromised; revoke the old key in the provider UI immediately.

```sh
# Verify permissions
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
```
