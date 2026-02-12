#!/bin/bash
set -e

# PortEden CLI Installation Script
# Downloads pre-built binary from GitHub releases

REPO="porteden/cli"
BINARY_NAME="porteden"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}==>${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest version from GitHub
get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_binary() {
    PLATFORM=$(detect_platform)
    VERSION=${VERSION:-$(get_latest_version)}

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Please set VERSION env var."
    fi

    VERSION_NUM=${VERSION#v}
    FILENAME="${BINARY_NAME}_${VERSION_NUM}_${PLATFORM}"

    if [ "$OS" = "windows" ]; then
        FILENAME="${FILENAME}.zip"
    else
        FILENAME="${FILENAME}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    info "Installing PortEden CLI ${VERSION} for ${PLATFORM}..."

    TMPDIR=$(mktemp -d)
    trap "rm -rf $TMPDIR" EXIT

    info "Downloading from ${DOWNLOAD_URL}..."
    if ! curl -sL "$DOWNLOAD_URL" -o "$TMPDIR/$FILENAME"; then
        error "Download failed. Check if release exists: https://github.com/${REPO}/releases"
    fi

    info "Extracting..."
    cd "$TMPDIR"
    if [ "$OS" = "windows" ]; then
        unzip -q "$FILENAME"
    else
        tar -xzf "$FILENAME"
    fi

    # Check if we can write to install dir
    if [ ! -w "$INSTALL_DIR" ]; then
        warn "Cannot write to ${INSTALL_DIR}. Using sudo..."
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        mv "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_install() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        success "Installation complete!"
        echo ""
        "$BINARY_NAME" --version
        echo ""
        echo "Get started:"
        echo "  porteden auth login --token <your-api-key>  # Direct token"
        echo "  porteden auth login                         # Browser OAuth"
        echo "  porteden calendar events --today -jc"
        echo ""
        echo "For help: porteden --help"
    else
        warn "Installation complete, but '$BINARY_NAME' not found in PATH"
        echo "Add ${INSTALL_DIR} to your PATH or use: ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

# Main
main() {
    echo ""
    echo "  PortEden CLI Installer"
    echo "  ======================"
    echo ""

    install_binary
    verify_install
}

main "$@"
