#!/bin/bash
# PreToolUse hook: block edits/writes to sensitive files (.env, keys, certs, lock files).
# Exit code 2 = block the tool call and show the message to Claude.

INPUT=$(cat)
FILE=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null)

[ -z "$FILE" ] && exit 0

BASENAME=$(basename "$FILE")

case "$BASENAME" in
  .env|.env.production|.env.prod)
    echo "BLOCKED: editing $BASENAME is not allowed via hooks policy. Ask the user to change it manually." >&2
    exit 2
    ;;
  *.pem|*.key|*.crt|oauth-private.key|oauth-public.key)
    echo "BLOCKED: $BASENAME looks like a secret/key file. Do not modify it." >&2
    exit 2
    ;;
  composer.lock|package-lock.json)
    echo "BLOCKED: lock files must only change via composer/npm commands, not direct edits." >&2
    exit 2
    ;;
esac

exit 0
