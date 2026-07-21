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
