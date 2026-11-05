#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# Axiom — Install Script
# Ecosystem, Frameworks, Workflows for AI coding agents.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.sh | bash
#
# Or download and run:
#   curl -fsSLO https://raw.githubusercontent.com/IGutierrezZ/axiom/main/scripts/install.sh
#   chmod +x install.sh
#   ./install.sh
# ============================================================================

GITHUB_OWNER="IGutierrezZ"
GITHUB_REPO="axiom"
BINARY_NAME="axiom"

# ============================================================================
# Color support
# ============================================================================

setup_colors() {
    if [ -t 1 ] && [ "${TERM:-}" != "dumb" ]; then
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        YELLOW='\033[1;33m'
        BLUE='\033[0;34m'
        CYAN='\033[0;36m'
        BOLD='\033[1m'
        DIM='\033[2m'
        NC='\033[0m'
    else
        RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' DIM='' NC=''
    fi
}

# ============================================================================
# Logging helpers
# ============================================================================

info()    { echo -e "${BLUE}[info]${NC}    $*"; }
success() { echo -e "${GREEN}[ok]${NC}      $*"; }
warn()    { echo -e "${YELLOW}[warn]${NC}    $*"; }
error()   { echo -e "${RED}[error]${NC}   $*" >&2; }
fatal()   { error "$@"; exit 1; }
step()    { echo -e "\n${CYAN}${BOLD}==>${NC} ${BOLD}$*${NC}"; }

# ============================================================================
# Help
# ============================================================================

show_help() {
    cat <<EOF
${BOLD}Axiom installer${NC}

Usage: install.sh [OPTIONS]

Options:
  --method METHOD   Force install method: go, binary (default: binary)
  --channel CHANNEL Axiom channel: stable (default), beta, or nightly (env: AXIOM_CHANNEL, fallback: GENTLE_AI_CHANNEL)
  --dir DIR         Custom install directory for binary method
  --insecure        Skip checksum verification (not recommended)
  -h, --help        Show this help

Install methods:
  1. binary  — Pre-built binary from GitHub Releases (default)
  2. go      — go install from source (needs Go)

Examples:
  curl -fsSL https://raw.githubusercontent.com/${GITHUB_OWNER}/${GITHUB_REPO}/main/scripts/install.sh | bash
  ./install.sh --method binary
  ./install.sh --channel beta
  ./install.sh --method binary --dir \$HOME/.local/bin
  ./install.sh --method binary --insecure   # skip checksum (not recommended)

EOF
}

# ============================================================================
# Platform detection
# ============================================================================

detect_platform() {
    local uname_os uname_arch

    uname_os="$(uname -s)"
    uname_arch="$(uname -m)"

    case "$uname_os" in
        Darwin) OS="darwin"; OS_LABEL="macOS"; GORELEASER_OS="darwin" ;;
        Linux)  OS="linux";  OS_LABEL="Linux"; GORELEASER_OS="linux" ;;
        *)      fatal "Unsupported OS: $uname_os. Only macOS and Linux are supported." ;;
    esac

    case "$uname_arch" in
        x86_64|amd64)   ARCH="amd64" ;;
        arm64|aarch64)  ARCH="arm64" ;;
        *)              fatal "Unsupported architecture: $uname_arch. Only amd64 and arm64 are supported." ;;
    esac

    success "Platform: ${OS_LABEL} (${OS}/${ARCH})"
}

# ============================================================================
# GoReleaser archive naming
#
# From .goreleaser.yaml:
#   name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
#
# GoReleaser v2 {{ .Os }} produces GOOS values (lowercase: darwin, linux)
# GoReleaser {{ .Arch }} produces GOARCH values (amd64, arm64)
# Examples:
#   axiom_1.0.0_darwin_arm64.tar.gz
#   axiom_1.0.0_linux_amd64.tar.gz
# ============================================================================

get_archive_name() {
    local version="$1"
    echo "${BINARY_NAME}_${version}_${GORELEASER_OS}_${ARCH}.tar.gz"
}

# ============================================================================
# Prerequisites
# ============================================================================

check_prerequisites() {
    step "Checking prerequisites"

    local missing=()

    if ! command -v curl &>/dev/null; then
        missing+=("curl")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        fatal "Missing required tools: ${missing[*]}. Please install them and try again."
    fi

    success "curl is available"
}

# ============================================================================
# Install method detection
# ============================================================================

detect_install_method() {
    if [ "${CHANNEL}" = "beta" ]; then
        if [ -n "${FORCE_METHOD:-}" ] && [ "${FORCE_METHOD}" != "go" ]; then
            fatal "--channel beta installs Axiom from main and only supports --method go"
        fi
        INSTALL_METHOD="go"
        info "Using beta channel — will install ${BINARY_NAME} from main via go install"
        return
    fi

    if [ -n "${FORCE_METHOD:-}" ]; then
        case "$FORCE_METHOD" in
            go|binary) INSTALL_METHOD="$FORCE_METHOD" ;;
            *) fatal "Unknown install method: $FORCE_METHOD. Use: go or binary" ;;
        esac
        info "Using forced method: $INSTALL_METHOD"
        return
    fi

    step "Detecting best install method"

    # Default: pre-built release archive from GitHub Releases, always up to
    # date. go install stays opt-in (--method go) because the Go module proxy
    # can lag behind new tags for up to 30 minutes, installing a stale version.
    INSTALL_METHOD="binary"
    info "Will download pre-built binary from GitHub Releases"
}

# ============================================================================
# Install via go install
# ============================================================================

install_go() {
    step "Installing via go install"

    if ! command -v go &>/dev/null; then
        fatal "Go 1.25.10 or newer is required for --method go. Install it from https://go.dev/dl/ or use the default binary method."
    fi

    local version="latest"
    if [ "${CHANNEL}" = "beta" ]; then
        version="main"
    fi
    # /v3 is part of the module path, not decoration: Go refuses to resolve a
    # module whose tags are v3.x unless the import path carries the major
    # version suffix. Keep the owner's exact case (IGutierrezZ): Go compares the
    # module path case-sensitively, so a lowercased owner makes go install fail.
    local go_package="github.com/${GITHUB_OWNER}/${GITHUB_REPO}/v3/cmd/${BINARY_NAME}@${version}"

    info "Running: go install ${go_package}"
    if [ "${CHANNEL}" = "beta" ]; then
        prepend_go_env_pattern GONOSUMDB github.com/IGutierrezZ/axiom/v3
        prepend_go_env_pattern GOPRIVATE github.com/IGutierrezZ/axiom/v3
        prepend_go_env_pattern GONOPROXY github.com/IGutierrezZ/axiom/v3
        export GONOSUMDB GOPRIVATE GONOPROXY

        if ! go install "$go_package"; then
            fatal "Failed to install via go install. Make sure Go is properly configured."
        fi
    elif ! go install "$go_package"; then
        fatal "Failed to install via go install. Make sure Go is properly configured."
    fi

    # Verify GOBIN / GOPATH/bin is in PATH
    local gobin
    gobin="$(go env GOBIN)"
    if [ -z "$gobin" ]; then
        gobin="$(go env GOPATH)/bin"
    fi

    ensure_dir_in_path "$gobin"

    success "Installed ${BINARY_NAME} via go install"
}

# ============================================================================
# PATH handling
# ============================================================================

PATH_MARKER="# axiom: PATH"
INSTALL_TMPDIR=""
INSTALL_STAGING=""

cleanup_install() {
    if [ -n "${INSTALL_STAGING:-}" ]; then
        rm -f -- "$INSTALL_STAGING" 2>/dev/null || true
    fi
    if [ -n "${INSTALL_TMPDIR:-}" ]; then
        rm -rf -- "$INSTALL_TMPDIR" 2>/dev/null || true
    fi
}

# path_rc_file SHELL_NAME OS -> prints the rc file of a recognized shell.
path_rc_file() {
    case "$1" in
        bash)
            if [ "$2" = "Darwin" ]; then
                printf '%s\n' "${HOME}/.bash_profile"
            else
                printf '%s\n' "${HOME}/.bashrc"
            fi
            ;;
        zsh)  printf '%s\n' "${HOME}/.zshrc" ;;
        fish) printf '%s\n' "${HOME}/.config/fish/config.fish" ;;
        *)    return 1 ;;
    esac
}

# path_rc_line SHELL_NAME DIR -> prints the marked line that adds DIR to PATH.
# DIR is quoted for the target shell so it cannot inject commands.
path_rc_line() {
    local shell_name="$1" dir="$2" quoted
    if [ "$shell_name" = "fish" ]; then
        quoted="${dir//\\/\\\\}"
        quoted="${quoted//\'/\\\'}"
        printf "fish_add_path '%s' %s\n" "$quoted" "$PATH_MARKER"
    else
        printf -v quoted '%q' "$dir"
        printf 'export PATH="$PATH":%s %s\n' "$quoted" "$PATH_MARKER"
    fi
}

# add_dir_to_shell_rc DIR SHELL_NAME OS: appends the marked line to the rc file
# unless a marked line for DIR already exists. Prints the rc file path only when
# it writes. Returns 1 when the shell is not recognized or the file cannot be
# written.
add_dir_to_shell_rc() {
    local dir="$1" shell_name="$2" os="$3" rc line
    case "$dir" in
        *$'\n'*) return 1 ;;
    esac
    rc="$(path_rc_file "$shell_name" "$os")" || return 1
    if [ -f "$rc" ] && grep -F -- "$PATH_MARKER" "$rc" | grep -qF -- "$dir"; then
        return 0
    fi
    line="$(path_rc_line "$shell_name" "$dir")"
    mkdir -p -- "$(dirname -- "$rc")" 2>/dev/null || return 1
    printf '\n%s\n' "$line" >> "$rc" 2>/dev/null || return 1
    printf '%s\n' "$rc"
}

# ensure_dir_in_path DIR: does nothing when DIR is already in PATH or
# AXIOM_NO_PERSISTENT_PATH=1. Otherwise it appends one marked line to the rc
# file of the detected shell, and falls back to the manual instruction when the
# shell is not recognized or the file cannot be written.
ensure_dir_in_path() {
    local dir="$1" shell_name rc
    if [[ ":$PATH:" == *":${dir}:"* ]]; then
        return 0
    fi
    shell_name="$(basename -- "${SHELL:-}")"
    if [ "${AXIOM_NO_PERSISTENT_PATH:-}" != "1" ] \
        && rc="$(add_dir_to_shell_rc "$dir" "$shell_name" "$(uname -s 2>/dev/null || true)")"; then
        if [ -n "$rc" ]; then
            success "Added ${dir} to PATH in ${rc}"
        fi
        warn "Open a new terminal (or reload your shell profile) to use '${BINARY_NAME}'"
        return 0
    fi
    warn "${dir} is not in your PATH"
    echo ""
    warn "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo -e "  ${DIM}export PATH=\"\$PATH:${dir}\"${NC}"
    echo ""
}

prepend_go_env_pattern() {
    local name="$1"
    local pattern="$2"
    local current="${!name:-}"

    if [ -z "$current" ]; then
        printf -v "$name" '%s' "$pattern"
        return
    fi

    case ",$current," in
        *",$pattern,"*) return ;;
        *) printf -v "$name" '%s,%s' "$pattern" "$current" ;;
    esac
}

# ============================================================================
# Install via binary download
# ============================================================================

get_latest_version() {
    local url="https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest"

    info "Fetching latest release from GitHub..."

    local response
    response="$(curl -fsSL -w "\n%{http_code}" "$url")" || fatal "Failed to fetch latest release"

    local http_code body
    http_code="$(echo "$response" | tail -n1)"
    body="$(echo "$response" | sed '$d')"

    if [ "$http_code" != "200" ]; then
        fatal "GitHub API returned HTTP $http_code. Rate limited? Try again later or use --method go"
    fi

    # Extract tag_name — works without jq
    LATEST_VERSION="$(echo "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"

    if [ -z "$LATEST_VERSION" ]; then
        fatal "Could not determine latest version from GitHub API response"
    fi

    # Strip leading 'v' for archive naming (goreleaser uses version without v prefix)
    VERSION_NUMBER="${LATEST_VERSION#v}"

    success "Latest version: ${LATEST_VERSION}"
}

install_binary() {
    step "Installing pre-built binary"

    get_latest_version

    local archive_name
    archive_name="$(get_archive_name "$VERSION_NUMBER")"
    local download_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${archive_name}"
    local checksums_url="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/checksums.txt"

    # Create temp directory — clean up on exit
    local tmpdir
    tmpdir="$(mktemp -d)"
    INSTALL_TMPDIR="$tmpdir"
    # TERM/INT exit explicitly so the EXIT trap always runs the cleanup.
    trap cleanup_install EXIT
    trap 'exit 143' TERM
    trap 'exit 130' INT

    # Download archive
    info "Downloading ${archive_name}..."
    if ! curl -sfL -o "${tmpdir}/${archive_name}" "$download_url"; then
        fatal "Failed to download ${download_url}"
    fi

    # Verify file was actually downloaded (not a 404 HTML page)
    local file_size
    file_size="$(wc -c < "${tmpdir}/${archive_name}" | tr -d '[:space:]')"
    if [ "$file_size" -lt 1000 ]; then
        fatal "Downloaded file is suspiciously small (${file_size} bytes). Archive may not exist for this platform."
    fi

    success "Downloaded ${archive_name} (${file_size} bytes)"

    # Download and verify checksum — fail closed unless --insecure is set
    info "Verifying checksum..."
    if curl -fsSL -o "${tmpdir}/checksums.txt" "$checksums_url"; then
        local expected_checksum
        expected_checksum="$(grep "${archive_name}" "${tmpdir}/checksums.txt" 2>/dev/null | awk '{print $1}' || true)"

        if [ -n "$expected_checksum" ]; then
            local actual_checksum
            if command -v sha256sum &>/dev/null; then
                actual_checksum="$(sha256sum "${tmpdir}/${archive_name}" | awk '{print $1}')"
            elif command -v shasum &>/dev/null; then
                actual_checksum="$(shasum -a 256 "${tmpdir}/${archive_name}" | awk '{print $1}')"
            else
                if [ "$INSECURE" = "true" ]; then
                    warn "No sha256sum or shasum found — checksum verification skipped (--insecure)"
                    actual_checksum="$expected_checksum"
                else
                    fatal "No sha256sum or shasum tool found. Cannot verify checksum.\nInstall coreutils (sha256sum) or use --insecure to skip (not recommended)."
                fi
            fi

            if [ "$actual_checksum" != "$expected_checksum" ]; then
                fatal "Checksum mismatch!\n  Expected: ${expected_checksum}\n  Got:      ${actual_checksum}"
            fi
            success "Checksum verified"
        else
            if [ "$INSECURE" = "true" ]; then
                warn "Archive '${archive_name}' not found in checksums.txt — checksum verification skipped (--insecure)"
            else
                fatal "Archive '${archive_name}' not found in checksums.txt. Refusing to install unverified binary.\nUse --insecure to skip (not recommended)."
            fi
        fi
    else
        if [ "$INSECURE" = "true" ]; then
            warn "Could not download checksums.txt — checksum verification skipped (--insecure)"
        else
            fatal "Could not download checksums.txt from:\n  ${checksums_url}\nRefusing to install without integrity verification.\nUse --insecure to skip (not recommended)."
        fi
    fi

    # Extract binary
    info "Extracting ${BINARY_NAME}..."
    if ! tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"; then
        fatal "Failed to extract archive"
    fi

    if [ ! -f "${tmpdir}/${BINARY_NAME}" ]; then
        fatal "Binary '${BINARY_NAME}' not found in archive"
    fi

    # Determine install directory
    local install_dir="${INSTALL_DIR:-}"

    if [ -z "$install_dir" ]; then
        if [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
            install_dir="/usr/local/bin"
        elif [ "$(id -u)" = "0" ]; then
            install_dir="/usr/local/bin"
        else
            install_dir="${HOME}/.local/bin"
        fi
    fi

    # Create install dir if needed
    mkdir -p "$install_dir"

    # Atomic install: stage next to the destination and rename in place. A
    # rename within one filesystem is atomic, so the previous binary either
    # survives intact or is fully replaced, never half-written. `install -m 0755`
    # fixes the mode regardless of the caller's umask. Paths are always passed as
    # arguments (also under sudo), never interpolated into a shell string.
    local final="${install_dir}/${BINARY_NAME}"
    local staging="${install_dir}/${BINARY_NAME}.staging.$$"
    INSTALL_STAGING="$staging"

    info "Installing to ${final}..."
    if install -m 0755 -- "${tmpdir}/${BINARY_NAME}" "$staging" 2>/dev/null \
        && mv -f -- "$staging" "$final"; then
        :
    elif command -v sudo &>/dev/null; then
        warn "Permission denied. Trying with sudo..."
        if ! sudo -- install -m 0755 -- "${tmpdir}/${BINARY_NAME}" "$staging" \
            || ! sudo -- mv -f -- "$staging" "$final"; then
            sudo -- rm -f -- "$staging" 2>/dev/null || true
            fatal "Could not install ${final} with sudo."
        fi
    else
        fatal "Cannot write to ${install_dir}. Run with sudo or use --dir to specify a writable directory."
    fi
    INSTALL_STAGING=""

    success "Installed ${BINARY_NAME} to ${final}"

    ensure_dir_in_path "$install_dir"
}

# ============================================================================
# Verify installation
# ============================================================================

verify_installation() {
    step "Verifying installation"

    # Allow PATH changes to take effect
    hash -r 2>/dev/null || true

    if command -v "$BINARY_NAME" &>/dev/null; then
        local version_output
        version_output="$("$BINARY_NAME" version 2>&1 || true)"
        success "${BINARY_NAME} is installed: ${version_output}"
        return 0
    fi

    # Check common locations even if not in PATH
    local locations=(
        "/usr/local/bin/${BINARY_NAME}"
        "${HOME}/.local/bin/${BINARY_NAME}"
        "$(go env GOPATH 2>/dev/null || echo "")/bin/${BINARY_NAME}"
    )

    for loc in "${locations[@]}"; do
        if [ -n "$loc" ] && [ -x "$loc" ]; then
            local version_output
            version_output="$("$loc" version 2>&1 || true)"
            success "Found ${BINARY_NAME} at ${loc}: ${version_output}"
            warn "Binary location is not in your PATH. Add it to use '${BINARY_NAME}' directly."
            return 0
        fi
    done

    warn "Could not verify installation. You may need to restart your shell."
    return 0
}

# ============================================================================
# Print next steps
# ============================================================================

print_banner() {
    echo ""
    echo -e "${CYAN}${BOLD}"
    echo "    _          _                 "
    echo "   / \   __  _(_) ___  _ __ ___  "
    echo "  / _ \  \ \/ / |/ _ \| '_ \` _ \ "
    echo " / ___ \  >  <| | (_) | | | | | |"
    echo "/_/   \_\/_/\_\_|\___/|_| |_| |_|"
    echo -e "${NC}"
    echo -e "  ${DIM}Axiom — Deterministic, Spec-Driven Software Engineering${NC}"
    echo ""
}

print_next_steps() {
    echo ""
    echo -e "${GREEN}${BOLD}Installation complete!${NC}"
    echo ""
    echo -e "${BOLD}Next steps:${NC}"
    if [ "${CHANNEL}" = "beta" ]; then
        echo -e "  ${CYAN}1.${NC} Run ${BOLD}AXIOM_CHANNEL=beta ${BINARY_NAME} install${NC} to keep using the beta channel"
    else
        echo -e "  ${CYAN}1.${NC} Run ${BOLD}${BINARY_NAME}${NC} to start the TUI installer"
    fi
    echo -e "  ${CYAN}2.${NC} Select your AI agent(s) and tools to configure"
    echo -e "  ${CYAN}3.${NC} Follow the interactive prompts"
    echo ""
    echo -e "${DIM}For help: ${BINARY_NAME} --help${NC}"
    echo -e "${DIM}Docs:     https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}${NC}"
    echo ""
}

# ============================================================================
# Main
# ============================================================================

main() {
    setup_colors

    # Parse arguments
    FORCE_METHOD=""
    INSTALL_DIR=""
    INSECURE="false"
    CHANNEL="${AXIOM_CHANNEL:-${GENTLE_AI_CHANNEL:-stable}}"

    while [ $# -gt 0 ]; do
        case "$1" in
            --method)
                [ $# -lt 2 ] && fatal "--method requires an argument"
                FORCE_METHOD="$2"; shift 2
                ;;
            --channel)
                [ $# -lt 2 ] && fatal "--channel requires an argument"
                CHANNEL="$2"; shift 2
                ;;
            --dir)
                [ $# -lt 2 ] && fatal "--dir requires an argument"
                INSTALL_DIR="$2"; shift 2
                ;;
            --insecure)
                INSECURE="true"; shift
                ;;
            -h|--help)
                setup_colors
                show_help
                exit 0
                ;;
            *)
                fatal "Unknown option: $1. Use --help for usage."
                ;;
        esac
    done

    case "${CHANNEL}" in
        stable|beta|nightly) ;;
        *) fatal "Unknown channel: ${CHANNEL}. Use: stable, beta, or nightly" ;;
    esac
    if [ "${CHANNEL}" = "nightly" ]; then
        CHANNEL="beta"
    fi

    print_banner

    step "Detecting platform"
    detect_platform

    check_prerequisites
    detect_install_method

    case "$INSTALL_METHOD" in
        go)     install_go ;;
        binary) install_binary ;;
    esac

    verify_installation
    print_next_steps
}

main "$@"
