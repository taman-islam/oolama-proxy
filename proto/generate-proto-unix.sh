#!/usr/bin/env bash
set -euo pipefail
set -x

# Generate TS from pinpost protos into commonTs/generated
# Pinned protoc to avoid cross-platform drift.

PROTOC_VERSION="33.2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Paths
LB_PROTO_SRC="${SCRIPT_DIR}"
LB_TS_OUT="${SCRIPT_DIR}/../fe/src/generated"

# Cache protoc per-user (no repo pollution, no Downloads)
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
  local os
  os="$(uname -s)"
  if [ "${os}" = "Darwin" ]; then
    echo "osx"
    return
  fi
  if [ "${os}" = "Linux" ]; then
    echo "linux"
    return
  fi
  die "Unsupported OS: ${os}"
}

function get_arch() {
  local arch
  arch="$(uname -m)"
  if [ "${arch}" = "x86_64" ] || [ "${arch}" = "amd64" ]; then
    echo "x86_64"
    return
  fi
  if [ "${arch}" = "arm64" ] || [ "${arch}" = "aarch64" ]; then
    echo "aarch_64"
    return
  fi
  die "Unsupported arch: ${arch}"
}

function ensure_pinned_protoc() {
  if [ -x "${PROTOC_BIN}" ]; then
    return
  fi

  ensure_command curl
  ensure_command unzip

  local os arch zip_name url tmp_zip
  os="$(get_os)"
  arch="$(get_arch)"

  zip_name="protoc-${PROTOC_VERSION}-${os}-${arch}.zip"
  url="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${zip_name}"

  mkdir -p "${PROTOC_DIR}"
  tmp_zip="${PROTOC_DIR}/${zip_name}"

  echo "⬇️  Downloading pinned protoc ${PROTOC_VERSION} (${os}/${arch}) to cache..."
  curl -fL "${url}" -o "${tmp_zip}"

  echo "📦 Extracting protoc into ${PROTOC_DIR}..."
  unzip -o "${tmp_zip}" -d "${PROTOC_DIR}" >/dev/null
  rm -f "${tmp_zip}"

  [ -x "${PROTOC_BIN}" ] || die "Pinned protoc install failed: ${PROTOC_BIN} not found/executable"
}

ensure_pinned_protoc
echo "✅ Using pinned protoc: $(${PROTOC_BIN} --version)"

# Prepare output
rm -rf "${LB_TS_OUT}"
mkdir -p "${LB_TS_OUT}"

# Collect protos
PROTO_FILES="$(find "${LB_PROTO_SRC}" -name "*.proto")"
[ -z "${PROTO_FILES}" ] && die "No .proto files found in ${LB_PROTO_SRC}"

# Run from fe to resolve local ts-proto plugin
cd "${SCRIPT_DIR}/../fe" || die "Failed to cd into fe"

TS_PROTO_PLUGIN_PATH="$(npx --no-install which protoc-gen-ts_proto)"
[ -z "${TS_PROTO_PLUGIN_PATH}" ] && die "Could not resolve protoc-gen-ts_proto (is ts-proto installed in fe?)"

"${PROTOC_BIN}" \
  --plugin="protoc-gen-ts_proto=${TS_PROTO_PLUGIN_PATH}" \
  --ts_proto_out="${LB_TS_OUT}/" \
  --ts_proto_opt=esModuleInterop=true,outputServices=false,exportCommonSymbols=false,useDate=true \
  -I"${LB_PROTO_SRC}" \
  ${PROTO_FILES}

echo "✅ Code generation complete. Output: ${LB_TS_OUT}"
echo "ℹ️  protoc cache: ${PROTOC_DIR}"
