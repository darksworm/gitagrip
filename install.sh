#!/bin/sh

# GitaGrip Installation Script
# This script downloads and installs the appropriate gitagrip binary for your system

set -e

# Default installation directory
INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
REPO="darksworm/gitagrip"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
error() {
    printf "${RED}Error: %s${NC}\n" "$1" >&2
    exit 1
}

success() {
    printf "${GREEN}%s${NC}\n" "$1"
}

info() {
    printf "${YELLOW}%s${NC}\n" "$1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
    
    case "$OS" in
        linux|darwin)
            ;;
        mingw*|msys*|cygwin*|windows*)
            OS="windows"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    
    echo "${OS}-${ARCH}"
}

# Get the latest release version
get_latest_version() {
    VERSION=${1:-"latest"}
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v?([^"]+)".*/\1/')
        if [ -z "$VERSION" ]; then
            error "Failed to fetch the latest version"
        fi
    else
        # Remove 'v' prefix if present
        VERSION=${VERSION#v}
    fi
    echo "$VERSION"
}

# Download and install gitagrip
install_gitagrip() {
    PLATFORM=$(detect_platform)
    VERSION=$(get_latest_version "$1")
    
    info "Installing gitagrip v${VERSION} for ${PLATFORM}..."
    
    # Construct download URL
    if [ "$OS" = "windows" ]; then
        FILENAME="gitagrip-${VERSION}-${PLATFORM}.zip"
        BINARY_NAME="gitagrip.exe"
    else
        FILENAME="gitagrip-${VERSION}-${PLATFORM}.tar.gz"
        BINARY_NAME="gitagrip"
    fi
    
    URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"
    
    # Create temp directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # Download
    info "Downloading from $URL..."
    if command -v curl >/dev/null; then
        curl -sL -o "$FILENAME" "$URL" || error "Failed to download gitagrip"
    elif command -v wget >/dev/null; then
        wget -q -O "$FILENAME" "$URL" || error "Failed to download gitagrip"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
    
    # Extract
    info "Extracting..."
    if [ "$OS" = "windows" ]; then
        unzip -q "$FILENAME" || error "Failed to extract archive"
    else
        tar -xzf "$FILENAME" || error "Failed to extract archive"
    fi
    
    # Install binary
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/" || error "Failed to move binary to $INSTALL_DIR"
    else
        info "Root permissions required to install to $INSTALL_DIR"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/" || error "Failed to move binary to $INSTALL_DIR"
    fi
    
    # Make executable (not needed on Windows)
    if [ "$OS" != "windows" ]; then
        if [ -w "$INSTALL_DIR/$BINARY_NAME" ]; then
            chmod +x "$INSTALL_DIR/$BINARY_NAME"
        else
            sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
        fi
    fi
    
    # Cleanup
    cd - >/dev/null
    rm -rf "$TMP_DIR"
    
    # Verify installation
    if command -v gitagrip >/dev/null; then
        success "gitagrip v${VERSION} has been installed successfully!"
        info "Run 'gitagrip --help' to get started"
    else
        error "Installation completed but gitagrip not found in PATH. You may need to add $INSTALL_DIR to your PATH."
    fi
}

# Main
main() {
    # Check for help
    case "$1" in
        -h|--help|help)
            echo "GitaGrip Installation Script"
            echo ""
            echo "Usage: $0 [VERSION]"
            echo ""
            echo "Arguments:"
            echo "  VERSION    Specific version to install (e.g., '0.1.0' or 'v0.1.0')"
            echo "             If not specified, installs the latest version"
            echo ""
            echo "Environment variables:"
            echo "  INSTALL_DIR    Installation directory (default: /usr/local/bin)"
            echo ""
            echo "Examples:"
            echo "  $0              # Install latest version"
            echo "  $0 0.1.0        # Install specific version"
            echo "  INSTALL_DIR=~/bin $0  # Install to custom directory"
            exit 0
            ;;
    esac
    
    install_gitagrip "$1"
}

main "$@"