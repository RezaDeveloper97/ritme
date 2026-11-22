#!/usr/bin/env bash
# secret-scan.sh — fail when a tracked file carries a secret (T-M12-07, findings S-12/S-13).
#
#   bash scripts/secret-scan.sh              scan the tracked files of this repository (git ls-files / git grep)
#   bash scripts/secret-scan.sh --self-test  prove every rule fires and the allow-listed fixtures pass
#
# A hit prints only "<file>:<line>  <rule>" — never the matched text — so CI logs stay clean.
# Exit 0 = clean, 1 = at least one hit, 2 = usage / environment error.
#
# Rules
#   app_key_in_example  APP_KEY=base64:<32+ literal key chars> in a *.example file (leave it empty). A real key is 44
#                       chars; 32 (not 40) also catches the shorter real-looking value S-12 found. $(…) does not match.
#   private_key         a PEM "BEGIN … PRIVATE KEY" header outside the allow-listed test-fixture dirs below
#   aws_access_key      AKIA… / ASIA… access key id
#   github_token        ghp_/gho_/ghu_/ghs_/ghr_ tokens and github_pat_ fine-grained tokens
#   google_api_key      AIza… API keys (Gemini, Maps, Firebase …)
#   slack_token         xoxa-/xoxb-/xoxp-/xoxo-/xoxr-/xoxs- tokens
#
# Allow-listed test-fixture dirs for private_key (public, test-only keys; production refuses them, see
# backend-go/internal/auth/passport/fixture.go). Add a dir here only for a key that is committed on purpose.
PRIVATE_KEY_FIXTURE_DIRS=(
  "backend-go/contract/fixtures/keys"
)

set -uo pipefail

# Patterns are POSIX ERE (git grep -E), no \b — BSD and GNU regex both accept them. Token rules require a
# non-alphanumeric neighbour so a long base64 / hash blob does not match by accident.
B='(^|[^A-Za-z0-9_])'
E='([^A-Za-z0-9_]|$)'
RULE_NAMES=(app_key_in_example private_key aws_access_key github_token github_token google_api_key slack_token)
RULE_PATTERNS=(
  'APP_KEY=["'"'"']?base64:[A-Za-z0-9+/_=-]{32,}'
  '-----BEGIN ([A-Z0-9]+ )*PRIVATE KEY-----'
  "${B}(AKIA|ASIA)[0-9A-Z]{16}${E}"
  "${B}gh[pousr]_[A-Za-z0-9]{36,}${E}"
  "${B}github_pat_[A-Za-z0-9_]{22,}"
  "${B}AIza[0-9A-Za-z_-]{35}${E}"
  "${B}xox[aboprs]-[A-Za-z0-9-]{10,}"
)

# scan <root> → prints hits, returns 0 clean / 1 hits.
scan() {
  local root=$1 hits=0 i name pattern out
  local -a spec
  for i in "${!RULE_NAMES[@]}"; do
    name=${RULE_NAMES[$i]}
    pattern=${RULE_PATTERNS[$i]}
    case $name in
      app_key_in_example) spec=('*.example') ;;
      private_key)
        spec=('.')
        local d
        for d in "${PRIVATE_KEY_FIXTURE_DIRS[@]}"; do spec+=(":(exclude)$d/**"); done
        ;;
      *) spec=('.') ;;
    esac
    # -I skips binaries, -n line numbers; the content after file:line is cut away before anything is printed.
    out=$(git -C "$root" grep -nIE --no-color -e "$pattern" -- "${spec[@]}" 2>/dev/null | cut -d: -f1,2) || true
    if [[ -n $out ]]; then
      hits=1
      while IFS= read -r loc; do printf '%s  %s\n' "$loc" "$name"; done <<<"$out"
    fi
  done
  return "$hits"
}

self_test() {
  local tmp fail=0
  tmp=$(mktemp -d "${TMPDIR:-/tmp}/secret-scan-selftest.XXXXXX") || exit 2
  trap 'rm -rf "$tmp"' RETURN
  git -C "$tmp" init -q || exit 2

  # Fakes are assembled from pieces so this script never matches its own rules.
  local b64 key_hdr
  b64=$(printf 'A%.0s' {1..43})=
  key_hdr="-----BEGIN"" PRIVATE KEY-----"
  mkdir -p "$tmp/leak" "$tmp/backend-go/contract/fixtures/keys" "$tmp/docs"
  # must fire
  printf 'APP_ENV=local\nAPP_KEY=base64:%s\n' "$b64" >"$tmp/.env.example"
  printf 'APP_KEY="base64:%s"\n' "$b64" >"$tmp/leak/env.example"
  printf 'APP_KEY=base64:%s\n' "$(printf 'k_-%.0s' {1..12})" >"$tmp/leak/short.env.example"
  printf '%s\nMIIB\n' "$key_hdr" >"$tmp/leak/id_rsa"
  printf '%s\nMIIB\n' "-----BEGIN"" RSA PRIVATE KEY-----" >"$tmp/leak/legacy.pem"
  printf '%s\nMIIB\n' "-----BEGIN"" OPENSSH PRIVATE KEY-----" >"$tmp/leak/ssh_key"
  printf 'aws_key = "%s"\n' "AK""IAABCDEFGHIJKLMNOP" >"$tmp/leak/aws.ini"
  printf 'token: %s\n' "gh""p_$(printf 'a%.0s' {1..36})" >"$tmp/leak/gh.yml"
  printf 'token: %s\n' "github""_pat_$(printf 'B%.0s' {1..30})" >"$tmp/leak/ghpat.yml"
  printf 'GEMINI=%s\n' "AI""za$(printf 'c%.0s' {1..35})" >"$tmp/leak/google.env"
  printf 'SLACK=%s\n' "xo""xb-1234567890-abcdef" >"$tmp/leak/slack.env"
  # must pass
  printf '%s\nMIIB\n' "$key_hdr" >"$tmp/backend-go/contract/fixtures/keys/oauth-private.key"
  # shellcheck disable=SC2016 # the literal $(…) is the point: a generator command must not be flagged
  printf '# key:generate --show\nAPP_KEY=\n#   echo "APP_KEY=base64:$(openssl rand -base64 32)"\n' >"$tmp/clean.example"
  printf 'APP_KEY=base64:%s\n' "$b64" >"$tmp/docs/not-an-example.md"
  printf 'see AKIA… and AIza… prefixes; sha512-%s\n' "$(printf 'Z%.0s' {1..80})" >"$tmp/docs/notes.md"
  git -C "$tmp" add -A || exit 2

  local out
  out=$(scan "$tmp")
  local rc=$?
  expect() { # expect <file> <rule>
    if ! grep -qE "^$1:[0-9]+  $2\$" <<<"$out"; then echo "FAIL: $2 did not fire on $1"; fail=1; fi
  }
  refuse() { # refuse <file>
    if grep -qE "^$1:" <<<"$out"; then echo "FAIL: $1 must pass but was flagged"; fail=1; fi
  }
  expect '\.env\.example' app_key_in_example
  expect 'leak/env\.example' app_key_in_example
  expect 'leak/short\.env\.example' app_key_in_example
  expect 'leak/id_rsa' private_key
  expect 'leak/legacy\.pem' private_key
  expect 'leak/ssh_key' private_key
  expect 'leak/aws\.ini' aws_access_key
  expect 'leak/gh\.yml' github_token
  expect 'leak/ghpat\.yml' github_token
  expect 'leak/google\.env' google_api_key
  expect 'leak/slack\.env' slack_token
  refuse 'backend-go/contract/fixtures/keys/oauth-private\.key'
  refuse 'clean\.example'
  refuse 'docs/not-an-example\.md'
  refuse 'docs/notes\.md'
  [[ $rc -eq 1 ]] || { echo "FAIL: scan exit $rc with hits, want 1"; fail=1; }

  # A tree with only the allow-listed fixture and clean files exits 0.
  git -C "$tmp" rm -rq --cached leak .env.example || exit 2
  out=$(scan "$tmp")
  rc=$?
  [[ $rc -eq 0 && -z $out ]] || { echo "FAIL: clean tree flagged (exit $rc): $out"; fail=1; }

  if [[ $fail -eq 0 ]]; then echo "secret-scan self-test: OK (${#RULE_NAMES[@]} patterns fire, fixtures pass)"; fi
  return "$fail"
}

case "${1:-}" in
  --self-test) self_test; exit $? ;;
  "") ;;
  *) echo "usage: $0 [--self-test]" >&2; exit 2 ;;
esac

root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel 2>/dev/null) || { echo "secret-scan: not a git checkout" >&2; exit 2; }
if scan "$root"; then
  echo "secret-scan: OK ($(git -C "$root" ls-files | wc -l | tr -d ' ') tracked files, ${#RULE_NAMES[@]} patterns)"
  exit 0
fi
echo "secret-scan: secrets found (file:line  rule above). Remove them, rotate anything real, never print the value." >&2
exit 1
