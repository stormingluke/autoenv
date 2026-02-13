#!/usr/bin/env bash
set -euo pipefail

# --- Colors and formatting ---

if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BOLD=''
    RESET=''
fi

info()    { printf "${YELLOW}[*]${RESET} %s\n" "$*"; }
success() { printf "${GREEN}[+]${RESET} %s\n" "$*"; }
error()   { printf "${RED}[!]${RESET} %s\n" "$*" >&2; }

# --- Cleanup trap ---

TMPDIR_INSTALL=""

cleanup() {
    if [ -n "$TMPDIR_INSTALL" ] && [ -d "$TMPDIR_INSTALL" ]; then
        rm -rf "$TMPDIR_INSTALL"
    fi
}

trap cleanup EXIT INT TERM

# --- Header ---

printf "\n"
printf "${BOLD}  autoenv installer${RESET}\n"
printf "  Automatic environment variable loading for your shell\n"
printf "\n"

# --- Check dependencies ---

info "Checking dependencies..."

missing=()
for cmd in curl tar; do
    if ! command -v "$cmd" &>/dev/null; then
        missing+=("$cmd")
    fi
done

if [ ${#missing[@]} -gt 0 ]; then
    error "Missing required dependencies: ${missing[*]}"
    error "Please install them and try again."
    exit 1
fi

success "All dependencies found (curl, tar)."

# --- Detect platform ---

info "Detecting platform..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        error "Unsupported architecture: $ARCH"
        error "autoenv supports: x86_64 (amd64), aarch64/arm64"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        if [ "$ARCH" != "amd64" ]; then
            error "Unsupported platform: linux/$ARCH"
            error "autoenv supports: linux/amd64, darwin/arm64"
            exit 1
        fi
        ;;
    darwin)
        if [ "$ARCH" != "arm64" ]; then
            error "Unsupported platform: darwin/$ARCH"
            error "autoenv supports: linux/amd64, darwin/arm64"
            exit 1
        fi
        ;;
    *)
        error "Unsupported operating system: $OS"
        error "autoenv supports: linux, darwin (macOS)"
        exit 1
        ;;
esac

success "Detected platform: ${OS}/${ARCH}"

# --- Fetch latest release tag ---

info "Fetching latest release from GitHub..."

REPO="stormingluke/autoenv"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

HTTP_RESPONSE=$(curl -fsSL -w "\n%{http_code}" "$API_URL" 2>/dev/null) || {
    error "Failed to fetch release information from GitHub."
    error "Please check your internet connection and try again."
    exit 1
}

HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n1)
BODY=$(echo "$HTTP_RESPONSE" | sed '$d')

if [ "$HTTP_STATUS" != "200" ]; then
    error "GitHub API returned HTTP $HTTP_STATUS."
    error "Could not determine latest release."
    exit 1
fi

TAG=$(echo "$BODY" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

if [ -z "$TAG" ]; then
    error "Could not parse release tag from GitHub API response."
    exit 1
fi

VERSION="${TAG#v}"

success "Latest release: ${TAG}"

# --- Download and install ---

ARCHIVE="autoenv_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"
INSTALL_DIR="${HOME}/.local/bin"

info "Downloading ${ARCHIVE}..."

TMPDIR_INSTALL="$(mktemp -d)"

curl -fsSL -o "${TMPDIR_INSTALL}/${ARCHIVE}" "$DOWNLOAD_URL" || {
    error "Failed to download: ${DOWNLOAD_URL}"
    error "The release archive may not exist for your platform."
    exit 1
}

success "Downloaded autoenv ${TAG}."

info "Extracting archive..."

tar -xzf "${TMPDIR_INSTALL}/${ARCHIVE}" -C "$TMPDIR_INSTALL" || {
    error "Failed to extract archive."
    exit 1
}

info "Installing to ${INSTALL_DIR}..."

mkdir -p "$INSTALL_DIR"

if [ -f "${TMPDIR_INSTALL}/autoenv" ]; then
    cp "${TMPDIR_INSTALL}/autoenv" "${INSTALL_DIR}/autoenv"
    chmod +x "${INSTALL_DIR}/autoenv"
else
    error "Binary not found in archive. Contents:"
    ls -la "$TMPDIR_INSTALL" >&2
    exit 1
fi

success "Installed autoenv to ${INSTALL_DIR}/autoenv"

# --- Shell setup instructions ---

printf "\n"
printf "${BOLD}  Setup${RESET}\n"
printf "\n"

CURRENT_SHELL="$(basename "${SHELL:-unknown}")"

if echo "$PATH" | tr ':' '\n' | grep -qx "${HOME}/.local/bin"; then
    success "~/.local/bin is already in your PATH."
else
    printf "${YELLOW}[*]${RESET} ~/.local/bin is not in your PATH. Add it to your shell config.\n"
fi

printf "\n"

ZSH_MARKER=""
BASH_MARKER=""

if [ "$CURRENT_SHELL" = "zsh" ]; then
    ZSH_MARKER=" ${GREEN}<-- your current shell${RESET}"
elif [ "$CURRENT_SHELL" = "bash" ]; then
    BASH_MARKER=" ${GREEN}<-- your current shell${RESET}"
fi

printf "  ${BOLD}# For zsh${RESET} (add to ~/.zshrc):${ZSH_MARKER}\n"
printf "  export PATH=\"\$HOME/.local/bin:\$PATH\"\n"
printf "  eval \"\$(autoenv hook zsh)\"\n"
printf "\n"
printf "  ${BOLD}# For bash${RESET} (add to ~/.bashrc):${BASH_MARKER}\n"
printf "  export PATH=\"\$HOME/.local/bin:\$PATH\"\n"
printf "  eval \"\$(autoenv hook bash)\"\n"
printf "\n"

info "Then restart your shell or run: source ~/.${CURRENT_SHELL}rc"
printf "\n"
success "Installation complete!"
printf "\n"
