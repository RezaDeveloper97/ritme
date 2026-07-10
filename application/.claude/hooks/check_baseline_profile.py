#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt): advisory-only reminder to refresh the Baseline
Profile when a startup / navigation / primary-list code path changes (CLAUDE.md §5).

A stale Baseline Profile AOT-compiles the wrong code, so any change to the cold-start
wiring or a main scrollable list should trigger regeneration. This hook never blocks —
it only surfaces a reminder as additionalContext."""
import json
import os
import re
import sys

# Filename signals for cold-start / navigation wiring.
STARTUP_NAME_HINTS = ("Application.kt", "MainActivity.kt")
NAV_NAME_HINTS = ("NavHost", "Navigation", "NavGraph", "AppNav")

# Content signals for a primary scrollable list (the other profiled path).
LIST_CONTENT_RE = re.compile(r"\b(LazyColumn|LazyRow|LazyVerticalGrid|LazyHorizontalGrid)\b")
NAV_CONTENT_RE = re.compile(r"\b(NavHost|rememberNavController|composable\s*\()")
STARTUP_CONTENT_RE = re.compile(r"class\s+\w+\s*:\s*Application\b|onCreate\s*\(")


def main():
    data = json.load(sys.stdin)
    path = data.get("tool_input", {}).get("file_path", "")
    if not path.endswith(".kt"):
        return 0
    base = os.path.basename(path)
    norm = path.replace(os.sep, "/")

    try:
        with open(path, encoding="utf-8") as f:
            src = f.read()
    except OSError:
        return 0

    reason = None
    if base in STARTUP_NAME_HINTS or STARTUP_CONTENT_RE.search(src):
        reason = "cold-start wiring (Application/entry Activity)"
    elif any(h in base for h in NAV_NAME_HINTS) or NAV_CONTENT_RE.search(src):
        reason = "navigation graph"
    elif LIST_CONTENT_RE.search(src) and "/ui/" in norm:
        reason = "a primary scrollable list"

    if reason:
        print(json.dumps({
            "hookSpecificOutput": {
                "hookEventName": "PostToolUse",
                "additionalContext": (
                    f"Baseline Profile reminder (CLAUDE.md §5): this change touches {reason}. "
                    "If it affects the cold-start-to-first-screen path or the main list scroll, "
                    "regenerate the Baseline Profile before finishing — a stale profile compiles "
                    "the wrong code and hurts startup/scroll on low-end devices."
                ),
            }
        }))
    return 0


if __name__ == "__main__":
    sys.exit(main())
