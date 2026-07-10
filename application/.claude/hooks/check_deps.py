#!/usr/bin/env python3
"""PreToolUse hook (Edit|Write): blocks banned third-party libraries (CLAUDE.md §3)."""
import json
import re
import sys

BANNED = [
    r"okhttp", r"retrofit", r"com\.google\.gson|com\.google\.code\.gson",
    r"moshi", r"kotlinx[.-]serialization", r"androidx\.room|[\"']room",
    r"dagger", r"hilt", r"koin", r"coil", r"glide", r"lottie",
    r"timber", r"io\.ktor", r"sentry", r"crashlytics", r"firebase",
]

def main():
    data = json.load(sys.stdin)
    tool_input = data.get("tool_input", {})
    path = tool_input.get("file_path", "")
    if "gradle" not in path.lower() and not path.endswith(".toml"):
        return 0
    text = tool_input.get("content", "") or tool_input.get("new_string", "")
    hits = sorted({p for p in BANNED if re.search(p, text, re.IGNORECASE)})
    if hits:
        print(
            "BLOCKED by zero-deps policy (CLAUDE.md §3): this edit adds banned "
            f"third-party dependencies matching: {', '.join(hits)}. "
            "Hand-write the capability instead (see the §3 table), or stop and "
            "ask the user for explicit approval before adding any library.",
            file=sys.stderr,
        )
        return 2
    return 0

if __name__ == "__main__":
    sys.exit(main())
