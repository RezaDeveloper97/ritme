#!/bin/bash
# PreToolUse hook for Bash: block dangerous/destructive commands.

INPUT=$(cat)
CMD=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('command',''))" 2>/dev/null)

[ -z "$CMD" ] && exit 0

block() { echo "BLOCKED by bash-guard: $1" >&2; exit 2; }

echo "$CMD" | grep -qE 'rm\s+-rf?\s+(/|~|\$HOME)(\s|$)' && block "recursive delete of root/home"
echo "$CMD" | grep -qE 'git\s+push\s+.*(--force|-f)\b' && block "force push — ask the user first"
echo "$CMD" | grep -qE 'git\s+reset\s+--hard' && block "git reset --hard — ask the user first"
echo "$CMD" | grep -qE 'migrate:fresh|migrate:reset|db:wipe' && block "destructive DB command (fresh/reset/wipe) — ask the user first"
echo "$CMD" | grep -qE '(^|\s)(cat|less|head|tail|grep)\s+[^|;]*\.env($|\s)' && block "reading .env directly — use .env.example instead"
echo "$CMD" | grep -qE 'docker\s+(system|volume)\s+prune' && block "docker prune — ask the user first"

exit 0
