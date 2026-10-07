#!/usr/bin/env bash
# Fail when a NEW unreachable function appears.
#
# Why a ratchet and not a clean gate: the repository already carries 230
# functions that nothing reaches from main, and most are legitimate (exported
# API exercised only by tests, platform-specific variants, mocks). Demanding
# that list reach zero before this guard can exist would mean the guard never
# exists. Freezing it and refusing growth is the part that pays for itself.
#
# What this catches: a function nothing calls. That is exactly how
# InstallCommandsForDep survived while the installation docs described the
# dependency remediation it would have performed, and nothing performed.
#
# What this does NOT catch, stated plainly so nobody trusts it further than it
# goes: an unreachable BRANCH inside a live function; a struct field that
# nothing ever assigns; a function that is called but whose effect is dead
# because its input is never configured. This is a call-graph tool. Those three
# shapes are found by executing the product, not by analyzing it.
#
# Regenerate the baseline after deliberately removing dead code:
#   scripts/deadcode-ratchet.sh --update
set -euo pipefail

# comm(1) requires both inputs sorted under the SAME collation, and silently
# produces nonsense when they are not. A baseline sorted under en_US.UTF-8 and
# compared under C (the usual CI locale) reports every baselined entry as new.
# Pinning the collation here and regenerating the baseline under it is what
# makes this reproducible off the author's machine.
export LC_ALL=C

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
baseline="${repo_root}/.deadcode-baseline.txt"
# The canonical binary, not ./cmd/gentle-ai. That one is a deprecation shim
# whose main() only forwards to app.RunArgs, so reachability measured from it
# is reachability of the shim's forwarding path -- every fork-owned command
# wired into cmd/axiom looks dead. Measured: 350 dead functions from the shim
# against 277 from the canonical binary, and the 73 difference is pure
# measurement artifact. The inherited baseline was generated upstream, where
# ./cmd/gentle-ai WAS the product; nobody regenerated it here because this step
# sat behind a benchmark step that sat behind a failing `go test`.
target="${DEADCODE_TARGET:-./cmd/axiom}"
tool="golang.org/x/tools/cmd/deadcode@v0.30.0"

cd "${repo_root}"

# The baseline is a Linux call graph (it is generated and checked on Linux CI).
# deadcode analyzes the build configuration of the host Go, so on Windows or
# macOS the platform-specific files change which functions look unreachable
# (for example *_windows.go helpers appear as new and Linux-only ones vanish).
# Pin the analyzed platform. The pin applies only to the analysis, never to
# building the tool below: a GOOS=linux binary could not run on a Windows host.
analysis_goos="linux"
analysis_goarch="amd64"

# Build the pinned tool for the host first, in its own step, so a failure here
# (no network for the module download, a toolchain problem) is reported as
# exactly that instead of looking like "no dead code" or like a baseline drift.
tool_dir="$(mktemp -d)"
trap 'rm -rf "${tool_dir}"' EXIT
if ! GOBIN="${tool_dir}" go install "${tool}" 2>"${tool_dir}/install.log"; then
  printf 'deadcode-ratchet: could not install %s:\n' "${tool}" >&2
  sed 's/^/  /' "${tool_dir}/install.log" >&2
  exit 1
fi
deadcode_bin="$(find "${tool_dir}" -maxdepth 1 -type f \( -name deadcode -o -name deadcode.exe \) | head -n 1)"
if [[ -z "${deadcode_bin}" ]]; then
  printf 'deadcode-ratchet: go install %s produced no deadcode binary in %s\n' "${tool}" "${tool_dir}" >&2
  exit 1
fi

# Keep the tool's own stderr: with 2>/dev/null and pipefail a failing analysis
# (bad package pattern, load error) used to surface as a baseline mismatch or as
# an empty result. Fail loudly with the tool's message instead.
raw="$(GOOS="${analysis_goos}" GOARCH="${analysis_goarch}" "${deadcode_bin}" "${target}" 2>"${tool_dir}/run.log")" || {
  printf 'deadcode-ratchet: deadcode failed for %s (GOOS=%s GOARCH=%s):\n' "${target}" "${analysis_goos}" "${analysis_goarch}" >&2
  sed 's/^/  /' "${tool_dir}/run.log" >&2
  exit 1
}

# Normalize to "<file>\t<symbol>". Line and column are deliberately dropped:
# pinning them would turn every unrelated edit above a dead function into a
# spurious failure, and a guard that cries wolf gets disabled.
current="$(printf '%s\n' "${raw}" \
  | sed -E 's/^(.+):[0-9]+:[0-9]+: unreachable func: (.+)$/\1\t\2/' \
  | tr '\\' '/' \
  | sort -u)"

if [[ "${1:-}" == "--update" ]]; then
  printf '%s\n' "${current}" > "${baseline}"
  printf 'baseline updated: %s entries\n' "$(printf '%s\n' "${current}" | grep -c '^' || true)"
  exit 0
fi

if [[ ! -f "${baseline}" ]]; then
  printf 'missing %s — run: scripts/deadcode-ratchet.sh --update\n' "${baseline}" >&2
  exit 1
fi

added="$(comm -13 "${baseline}" <(printf '%s\n' "${current}") || true)"
removed="$(comm -23 "${baseline}" <(printf '%s\n' "${current}") || true)"

if [[ -n "${removed}" ]]; then
  printf 'note: %s baselined entries are now reachable or gone.\n' "$(printf '%s\n' "${removed}" | grep -c '^' || true)"
  printf '      Tighten the baseline with: scripts/deadcode-ratchet.sh --update\n\n'
fi

if [[ -n "${added}" ]]; then
  printf 'NEW unreachable functions (nothing calls these):\n\n' >&2
  printf '%s\n' "${added}" | sed 's/^/  /' >&2
  printf '\nEither wire it up, delete it, or if it is genuinely reachable in a way\n' >&2
  printf 'the call graph cannot see, run --update and say why in the commit.\n' >&2
  exit 1
fi

printf 'no new unreachable functions\n'
