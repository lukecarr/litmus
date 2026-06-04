#!/usr/bin/env bash
#
# Entry point for the Litmus GitHub Action. Downloads the release binary that
# matches the runner, maps the action inputs (passed as INPUT_* environment
# variables) to litmus flags, and runs it.
set -euo pipefail

# LITMUS_REPO and LITMUS_BIN_DIR can be overridden for local testing.
repo="${LITMUS_REPO:-lukecarr/litmus}"

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

# Map the runner platform to the names GoReleaser uses in release archives.
case "${RUNNER_OS:-}" in
  Linux) os=linux ;;
  macOS) os=darwin ;;
  Windows) os=windows ;;
  *) echo "::error::Unsupported runner OS: ${RUNNER_OS:-unset}" >&2; exit 1 ;;
esac

case "${RUNNER_ARCH:-}" in
  X64) arch=amd64 ;;
  ARM64) arch=arm64 ;;
  X86) arch=386 ;;
  *) echo "::error::Unsupported runner architecture: ${RUNNER_ARCH:-unset}" >&2; exit 1 ;;
esac

bin=litmus
[ "$os" = windows ] && bin=litmus.exe

# Resolve which release to download. An explicit version input wins. Otherwise
# follow the pinned action ref when it is an exact release tag (vX.Y.Z), and fall
# back to the latest release for branches, SHAs, or moving major tags like v1.
version="${LITMUS_VERSION:-}"
if [ -z "$version" ]; then
  if printf '%s' "${LITMUS_ACTION_REF:-}" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
    version="$LITMUS_ACTION_REF"
  else
    version=latest
  fi
fi

# Download and unpack the matching archive. The version is wildcarded in the
# pattern, so it keeps matching as releases change.
workdir="$(mktemp -d)"
echo "Downloading litmus ($version) for $os/$arch from $repo..."
if [ "$version" = latest ]; then
  gh release download --repo "$repo" --pattern "litmus_*_${os}_${arch}.tar.gz" --dir "$workdir"
else
  gh release download "$version" --repo "$repo" --pattern "litmus_*_${os}_${arch}.tar.gz" --dir "$workdir"
fi

archive="$(find "$workdir" -name "litmus_*_${os}_${arch}.tar.gz" -print -quit)"
if [ -z "$archive" ]; then
  echo "::error::No litmus archive found for $os/$arch" >&2
  exit 1
fi
tar -xzf "$archive" -C "$workdir"
chmod +x "$workdir/$bin"

# Credentials go through the environment variables litmus already reads, so they
# never appear on the command line. Only export when an input was provided, so an
# empty input does not clobber a value already set in the job environment.
if [ -n "${INPUT_API_KEY:-}" ]; then
  case "${INPUT_PROVIDER:-openrouter}" in
    cloudflare) export CLOUDFLARE_API_KEY="$INPUT_API_KEY" ;;
    *) export OPENROUTER_API_KEY="$INPUT_API_KEY" ;;
  esac
fi
[ -n "${INPUT_CF_ACCOUNT_ID:-}" ] && export CLOUDFLARE_ACCOUNT_ID="$INPUT_CF_ACCOUNT_ID"
[ -n "${INPUT_CF_GATEWAY:-}" ] && export CLOUDFLARE_GATEWAY_ID="$INPUT_CF_GATEWAY"
[ -n "${INPUT_CF_TOKEN:-}" ] && export CF_AIG_TOKEN="$INPUT_CF_TOKEN"

# Assemble the litmus run arguments.
args=(run
  --tests "${INPUT_TESTS:?tests input is required}"
  --schema "${INPUT_SCHEMA:?schema input is required}"
  --parallel "${INPUT_PARALLEL:-1}"
  --output "${INPUT_OUTPUT:-github}"
  --provider "${INPUT_PROVIDER:-openrouter}")

[ -n "${INPUT_PROMPT:-}" ] && args+=(--prompt "$INPUT_PROMPT")
[ -n "${INPUT_PROMPT_FILE:-}" ] && args+=(--prompt-file "$INPUT_PROMPT_FILE")

# Expand the model input (newline- or comma-separated) into repeated --model flags.
while IFS= read -r line; do
  model="$(trim "$line")"
  [ -n "$model" ] && args+=(--model "$model")
done < <(printf '%s\n' "${INPUT_MODEL:-}" | tr ',' '\n')

exec "$workdir/$bin" "${args[@]}"
