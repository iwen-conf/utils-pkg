#!/usr/bin/env bash
set -euo pipefail

COVERAGE_MIN_TOTAL="${COVERAGE_MIN_TOTAL:-75.0}"
COVERAGE_MIN_ERRORS="${COVERAGE_MIN_ERRORS:-65.0}"
COVERAGE_MIN_SLICE="${COVERAGE_MIN_SLICE:-80.0}"
COVERAGE_MIN_USERAGENT="${COVERAGE_MIN_USERAGENT:-80.0}"
COVERAGE_MIN_JWT="${COVERAGE_MIN_JWT:-80.0}"
COVERAGE_MIN_CRYPTO="${COVERAGE_MIN_CRYPTO:-80.0}"
COVERAGE_MIN_URL="${COVERAGE_MIN_URL:-80.0}"
COVERAGE_MIN_CAPTCHA="${COVERAGE_MIN_CAPTCHA:-90.0}"
COVERAGE_MIN_PAGINATION="${COVERAGE_MIN_PAGINATION:-90.0}"
COVERAGE_MIN_PTR="${COVERAGE_MIN_PTR:-95.0}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

cover_total() {
  local profile="$1"
  local line
  line="$(go tool cover -func="$profile" | awk '/^total:/ {print $3; found=1} END {if (!found) exit 1}')"
  if [[ -z "$line" ]]; then
    echo "unable to parse coverage from $profile" >&2
    exit 1
  fi
  printf '%s' "${line%%%}"
}

fail_if_below() {
  local name="$1"
  local got="$2"
  local min="$3"
  awk -v got="$got" -v min="$min" -v name="$name" 'BEGIN {
    if (got+0 != got || min+0 != min) { print name ": unparsable coverage"; exit 1 }
    if (got+0 < min+0) { printf "%s coverage %.1f%% < %.1f%%\n", name, got, min; exit 1 }
    printf "%s coverage %.1f%% (min %.1f%%)\n", name, got, min
  }'
}

go test -count=1 -covermode=atomic -coverprofile="$workdir/all.out" ./...
fail_if_below "aggregate" "$(cover_total "$workdir/all.out")" "$COVERAGE_MIN_TOTAL"

check_pkg() {
  local pkg="$1"
  local min="$2"
  local name="$3"
  local profile="$workdir/${name}.out"
  go test -count=1 -covermode=atomic -coverprofile="$profile" "./$pkg"
  fail_if_below "$pkg" "$(cover_total "$profile")" "$min"
}

check_pkg errors "$COVERAGE_MIN_ERRORS" errors
check_pkg slice "$COVERAGE_MIN_SLICE" slice
check_pkg useragent "$COVERAGE_MIN_USERAGENT" useragent
check_pkg jwt "$COVERAGE_MIN_JWT" jwt
check_pkg crypto "$COVERAGE_MIN_CRYPTO" crypto
check_pkg url "$COVERAGE_MIN_URL" url
check_pkg captcha "$COVERAGE_MIN_CAPTCHA" captcha
check_pkg pagination "$COVERAGE_MIN_PAGINATION" pagination
check_pkg ptr "$COVERAGE_MIN_PTR" ptr
