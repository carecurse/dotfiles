# dotfiles

Personal configuration files managed with symlinks and Git.

## Structure

```
dotfiles/
├── bash/
│   └── .bashrc          # Symlinked → ~/.bashrc
├── config/              # XDG config stubs (add per-app subdirs)
├── git/                 # git config files
├── ssh/                 # Non-secret SSH config (config, known_hosts.example)
└── README.md
```

## Bootstrap

Clone and run the symlink setup on a new machine:

```sh
git clone git@github.com:<user>/dotfiles.git ~/dotfiles

# Shell
ln -sf ~/dotfiles/bash/.bashrc ~/.bashrc

# Git (if you add git/config)
ln -sf ~/dotfiles/git/config ~/.gitconfig

# SSH config (public/non-secret only)
mkdir -p ~/.ssh
ln -sf ~/dotfiles/ssh/config ~/.ssh/config
chmod 700 ~/.ssh
```

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
