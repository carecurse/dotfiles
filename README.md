# dotfiles

Personal configuration files managed with Git.

## Structure

```
dotfiles/
├── bash/
│   └── bashrc           # Sourced fragment — custom config only
├── config/              # XDG config stubs (add per-app subdirs)
├── git/                 # gitconfig (symlinked → ~/.gitconfig)
├── ssh/                 # Non-secret SSH config (config, known_hosts.example)
├── install.sh           # Bootstrap script
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

## Bootstrap

```sh
git clone git@github.com:<user>/dotfiles.git ~/dotfiles
bash ~/dotfiles/install.sh
```

`install.sh` will:
- Append the source line to `~/.bashrc` (idempotent)
- Create `~/.bashrc.local` template if missing
- Symlink `ssh/config` → `~/.ssh/config` if present
- Symlink `git/config` → `~/.gitconfig` if present

Run with `--dry-run` to preview changes without applying them.

---

## SSH Token Authentication — GitHub / GitLab

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
