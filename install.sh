#!/bin/sh
# casa one-line bootstrap.
#   curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-username>
# Installs Homebrew (if missing), installs casa, and—if a username/repo is given—
# sets up this machine from your dotfiles.
set -eu

if ! command -v brew >/dev/null 2>&1; then
  # Homebrew's Linux prerequisites (https://docs.brew.sh/Homebrew-on-Linux)
  if [ "$(uname -s)" = "Linux" ]; then
    SUDO=""
    [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"
    if command -v apt-get >/dev/null 2>&1; then
      $SUDO apt-get update -qq
      $SUDO apt-get install -y build-essential procps curl file git
    elif command -v dnf >/dev/null 2>&1; then
      $SUDO dnf group install -y development-tools
      $SUDO dnf install -y procps-ng curl file git
    elif command -v pacman >/dev/null 2>&1; then
      $SUDO pacman -S --noconfirm --needed base-devel procps-ng curl file git
    fi
  fi
  echo "==> installing Homebrew"
  NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  for p in /opt/homebrew/bin/brew /home/linuxbrew/.linuxbrew/bin/brew /usr/local/bin/brew; do
    [ -x "$p" ] && eval "$("$p" shellenv)" && break
  done
fi

echo "==> installing casa"
brew install carrots-sh/tap/casa

if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  echo "==> setting up this machine from: $1"
  casa machine setup "$1"
else
  echo "==> done. run 'casa' to get started."
fi
