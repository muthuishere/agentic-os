#!/bin/sh
# Install aos, installing Go first if it is missing.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/muthuishere/aos/main/install.sh | sh
#
# Or, if you would rather read it first — which is the right instinct for
# anything you pipe into a shell:
#   curl -fsSL https://raw.githubusercontent.com/muthuishere/aos/main/install.sh -o install.sh
#   less install.sh && sh install.sh
#
# It installs into your home directory only. Nothing here needs root, and the
# script will not ask for it.

set -eu

MODULE="github.com/muthuishere/aos/cmd/aos@latest"
RELEASE_BASE="https://github.com/muthuishere/aos/releases/latest/download"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
MIN_GO_MAJOR=1
MIN_GO_MINOR=24
GO_INSTALL_DIR="${GO_INSTALL_DIR:-$HOME/.local/go}"

say()  { printf '%s\n' "$*"; }
step() { printf '\n==> %s\n' "$*"; }
die()  { printf 'install: %s\n' "$*" >&2; exit 1; }

# go_is_new_enough reports whether the go on PATH can build this module. An old
# toolchain fails deep inside the build with a confusing message, so check first.
go_is_new_enough() {
    command -v go >/dev/null 2>&1 || return 1
    version=$(go env GOVERSION 2>/dev/null || echo "")
    version=${version#go}
    major=${version%%.*}
    rest=${version#*.}
    minor=${rest%%.*}
    [ -n "${major:-}" ] && [ -n "${minor:-}" ] || return 1
    [ "$major" -gt "$MIN_GO_MAJOR" ] && return 0
    [ "$major" -eq "$MIN_GO_MAJOR" ] && [ "$minor" -ge "$MIN_GO_MINOR" ]
}

detect_platform() {
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        darwin|linux) ;;
        *) die "unsupported OS $os. On Windows use install.ps1." ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch=amd64 ;;
        arm64|aarch64) arch=arm64 ;;
        *) die "unsupported architecture $(uname -m)" ;;
    esac
}

# install_go fetches the official toolchain into $GO_INSTALL_DIR and verifies it
# against the checksum go.dev publishes beside the archive.
install_go() {
    detect_platform
    step "Installing Go (none found, or too old for this module)"

    version=$(curl -fsSL https://go.dev/VERSION?m=text | head -n1)
    [ -n "$version" ] || die "could not determine the current Go version"
    archive="${version}.${os}-${arch}.tar.gz"
    url="https://go.dev/dl/${archive}"
    say "    $version for ${os}/${arch}"

    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT
    curl -fsSL "$url" -o "$tmp/$archive" || die "download failed: $url"

    # Verify before unpacking: an unverified toolchain is a very bad thing to put
    # on someone's PATH.
    #
    # The checksum comes from the JSON index, not from "<url>.sha256" — that
    # path returns an HTML page with a 200, which compared against a real hash
    # fails every time and would break the install for everyone who needs it.
    expected=$(curl -fsSL 'https://go.dev/dl/?mode=json' 2>/dev/null | awk -v want="\"$archive\"" '
        index($0, want) { found = 1 }
        found && /"sha256"/ { gsub(/[^0-9a-f]/, "", $2); print $2; exit }
    ')
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
    else
        actual=""
    fi

    if [ -n "$expected" ] && [ -n "$actual" ]; then
        [ "$expected" = "$actual" ] || die "checksum mismatch for $archive — refusing to install"
        say "    checksum verified"
    else
        say "    WARNING: could not verify the checksum for $archive"
    fi

    rm -rf "$GO_INSTALL_DIR"
    mkdir -p "$(dirname "$GO_INSTALL_DIR")"
    tar -C "$tmp" -xzf "$tmp/$archive"
    mv "$tmp/go" "$GO_INSTALL_DIR"
    PATH="$GO_INSTALL_DIR/bin:$PATH"
    export PATH
    say "    installed to $GO_INSTALL_DIR"
}

# install_release fetches the prebuilt binary for this platform. Most people do
# not need a Go toolchain to run a Go program, and asking them to install one is
# a big ask for a first try.
install_release() {
    detect_platform
    asset="aos-${os}-${arch}"
    step "Downloading aos for ${os}/${arch}"

    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT
    if ! curl -fsSL "${RELEASE_BASE}/${asset}" -o "$tmp/aos" 2>/dev/null; then
        say "    no prebuilt binary for this platform"
        return 1
    fi

    # Verify against the checksums published beside the binaries. A failure to
    # fetch them is a warning; a mismatch is fatal.
    if curl -fsSL "${RELEASE_BASE}/checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
        expected=$(grep " ${asset}\$" "$tmp/checksums.txt" | cut -d' ' -f1)
        if command -v sha256sum >/dev/null 2>&1; then
            actual=$(sha256sum "$tmp/aos" | cut -d' ' -f1)
        elif command -v shasum >/dev/null 2>&1; then
            actual=$(shasum -a 256 "$tmp/aos" | cut -d' ' -f1)
        else
            actual=""
        fi
        if [ -n "$expected" ] && [ -n "$actual" ]; then
            [ "$expected" = "$actual" ] || die "checksum mismatch for $asset — refusing to install"
            say "    checksum verified"
        fi
    fi

    mkdir -p "$BIN_DIR"
    mv "$tmp/aos" "$BIN_DIR/aos"
    chmod +x "$BIN_DIR/aos"
    binary="$BIN_DIR/aos"
    say "    $binary"
    return 0
}

# build_from_source is the fallback: a platform with no prebuilt binary, or a
# release that cannot be reached.
build_from_source() {
    if go_is_new_enough; then
        step "Go $(go env GOVERSION) is already installed"
    else
        install_go
    fi

    step "Building aos"
    go install "$MODULE"

    gobin=$(go env GOBIN)
    [ -n "$gobin" ] || gobin="$(go env GOPATH)/bin"
    binary="$gobin/aos"
    [ -x "$binary" ] || die "expected a binary at $binary"
    BIN_DIR="$gobin"
    say "    $binary"
}

main() {
    if [ "${AOS_FROM_SOURCE:-}" = "1" ] || ! install_release; then
        build_from_source
    fi

    step "Installing the agent skill"
    "$binary" install --skills

    step "Checking this machine"
    "$binary" doctor || true

    if command -v aos >/dev/null 2>&1; then
        printf '\nReady. Try: aos --help\n'
    else
        # Say exactly what to add rather than editing someone's shell profile
        # without asking.
        printf '\nReady, but %s is not on your PATH yet.\n' "$BIN_DIR"
        printf 'Add this to your shell profile:\n\n    export PATH="%s:$PATH"\n' "$BIN_DIR"
        printf '\nOr run it directly: %s --help\n' "$binary"
    fi
}

main "$@"
