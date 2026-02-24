#!/usr/bin/env bash
set -euo pipefail
set -x

PROTOC_GEN_GO_VERSION="v1.36.1"
PROTOC_VERSION="33.2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

LB_PROTO_SRC="${SCRIPT_DIR}"
LB_GO_OUT="${SCRIPT_DIR}/../be"
LB_PB_GO_OUT_REL="${SCRIPT_DIR}/../be/pb"

# Cache protoc per-user
UNAME_S="$(uname -s)"
if [ "${UNAME_S}" = "Darwin" ]; then
  PROTOC_CACHE_BASE="${HOME}/Library/Caches/lb"
else
  PROTOC_CACHE_BASE="${HOME}/.cache/lb"
fi
PROTOC_DIR="${PROTOC_CACHE_BASE}/protoc/${PROTOC_VERSION}"
PROTOC_BIN="${PROTOC_DIR}/bin/protoc"

function die() {
  echo "❌ $1" >&2
  exit 1
}

function ensure_command() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

function get_os() {
  local os="$(uname -s)"
  if [ "${os}" = "Darwin" ]; then echo "osx"; return; fi
  if [ "${os}" = "Linux" ]; then echo "linux"; return; fi
  die "Unsupported OS: ${os}"
}

function get_arch() {
  local arch="$(uname -m)"
  if [ "${arch}" = "x86_64" ] || [ "${arch}" = "amd64" ]; then echo "x86_64"; return; fi
  if [ "${arch}" = "arm64" ] || [ "${arch}" = "aarch64" ]; then echo "aarch_64"; return; fi
  die "Unsupported arch: ${arch}"
}

function ensure_pinned_protoc() {
  if [ -x "${PROTOC_BIN}" ]; then return; fi
  ensure_command curl
  ensure_command unzip

  local os="$(get_os)"
  local arch="$(get_arch)"
  local zip_name="protoc-${PROTOC_VERSION}-${os}-${arch}.zip"
  local url="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${zip_name}"

  mkdir -p "${PROTOC_DIR}"
  local tmp_zip="${PROTOC_DIR}/${zip_name}"
  echo "⬇️  Downloading pinned protoc ${PROTOC_VERSION} (${os}/${arch}) to cache..."
  curl -fL "${url}" -o "${tmp_zip}"
  echo "📦 Extracting protoc into ${PROTOC_DIR}..."
  unzip -o "${tmp_zip}" -d "${PROTOC_DIR}" >/dev/null
  rm -f "${tmp_zip}"
  [ -x "${PROTOC_BIN}" ] || die "Pinned protoc install failed"
}

function ensure_protoc_gen_go() {
  ensure_command go
  local go_bin_dir="$(go env GOPATH)/bin"
  local protoc_gen_go="${go_bin_dir}/protoc-gen-go"
  if [ ! -x "${protoc_gen_go}" ]; then
    go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
  fi
  [ -x "${protoc_gen_go}" ] || die "protoc-gen-go not found at ${protoc_gen_go}"
  echo "${protoc_gen_go}"
}

ensure_pinned_protoc
PROTOC_GEN_GO_BIN="$(ensure_protoc_gen_go)"

rm -rf "${LB_PB_GO_OUT_REL}"
mkdir -p "${LB_PB_GO_OUT_REL}"

PROTO_FILES="$(find "${LB_PROTO_SRC}" -name "*.proto")"
[ -z "${PROTO_FILES}" ] && die "No .proto files found"

"${PROTOC_BIN}" \
  --plugin="protoc-gen-go=${PROTOC_GEN_GO_BIN}" \
  -I="${LB_PROTO_SRC}" \
  --go_out="${LB_GO_OUT}" \
  --go_opt=module=lb \
  ${PROTO_FILES}

echo "✅ Go code generation complete. Output root: ${LB_GO_OUT}"
