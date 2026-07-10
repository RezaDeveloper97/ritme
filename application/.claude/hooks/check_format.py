#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt): non-blocking style warnings (CLAUDE.md §4, §5c)."""
import json
import re
import sys

def main():
    data = json.load(sys.stdin)
    path = data.get("tool_input", {}).get("file_path", "")
    if not path.endswith(".kt"):
        return 0
    try:
        with open(path, encoding="utf-8") as f:
            lines = f.read().splitlines()
    except OSError:
        return 0

    warnings = []
    src = "\n".join(lines)
    if any("\t" in l for l in lines):
        warnings.append("tab characters found — use 4-space indentation")
    if any(l != l.rstrip() for l in lines):
        warnings.append("trailing whitespace present")
    if re.search(r"^import\s+[\w.]+\.\*", src, re.MULTILINE):
        warnings.append("wildcard import — use explicit imports")
    if ("/ui/" in path or path.endswith("Screen.kt")) and re.search(r"Color\(0x[0-9A-Fa-f]{6,8}\)", src):
        warnings.append("inline Color(0x...) in UI code — use RitmeTheme tokens (§5c, no magic colors)")
    if re.search(r"\bColumn\s*\{[^}]*\.forEach", src, re.DOTALL):
        warnings.append("Column + forEach over a collection — use LazyColumn with stable keys (§5)")
    long_lines = sum(1 for l in lines if len(l) > 140)
    if long_lines:
        warnings.append(f"{long_lines} line(s) exceed 140 chars")

    if warnings:
        out = {
            "hookSpecificOutput": {
                "hookEventName": "PostToolUse",
                "additionalContext": "Style warnings for " + path + ": " + "; ".join(warnings),
            }
        }
        print(json.dumps(out))
    return 0

if __name__ == "__main__":
    sys.exit(main())
