#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt/.xml): enforces Vazirmatn Regular as the only
font — everything must flow through RitmeFontFamily/RitmeTypography (CLAUDE.md §5c)."""
import json
import os
import re
import sys

TYPOGRAPHY_FILE = "RitmeTypography.kt"
FONT_RES = "vazirmatn_regular"
# The bundled Vazirmatn weights. Regular is the XML/default weight; the rest are the
# real weight files RitmeTypography maps to via FontWeight (CLAUDE.md §5c).
ALLOWED_FONT_RES = {
    "vazirmatn_regular",
    "vazirmatn_medium",
    "vazirmatn_semibold",
    "vazirmatn_bold",
}

def main():
    data = json.load(sys.stdin)
    path = data.get("tool_input", {}).get("file_path", "")
    if not path.endswith((".kt", ".xml")):
        return 0
    try:
        with open(path, encoding="utf-8") as f:
            src = f.read()
    except OSError:
        return 0

    problems = []
    is_typography = path.endswith(TYPOGRAPHY_FILE)

    if path.endswith(".kt"):
        if not is_typography:
            # Platform/system font families are banned outside the theme file.
            for m in re.finditer(r"FontFamily\.(Default|SansSerif|Serif|Monospace|Cursive)", src):
                problems.append(f"uses system font '{m.group(0)}' — use RitmeFontFamily (Vazirmatn Regular) from the theme")
            # Ad-hoc font family construction outside the theme file.
            for m in re.finditer(r"FontFamily\s*\(\s*Font\s*\(\s*R\.font\.(\w+)", src):
                problems.append(f"builds its own FontFamily from R.font.{m.group(1)} — only {TYPOGRAPHY_FILE} may define the font family")
            # fontFamily = something other than the theme family / a TextStyle-derived value.
            for m in re.finditer(r"fontFamily\s*=\s*([\w.]+)", src):
                ref = m.group(1)
                if "RitmeFontFamily" not in ref and not ref.startswith(("MaterialTheme", "LocalTextStyle")):
                    problems.append(f"fontFamily = {ref} — must be RitmeFontFamily (Vazirmatn Regular)")
        else:
            # Inside the theme file: only bundled Vazirmatn weights are allowed.
            for m in re.finditer(r"R\.font\.(\w+)", src):
                if m.group(1) not in ALLOWED_FONT_RES:
                    problems.append(
                        f"references R.font.{m.group(1)} — allowed Vazirmatn files are "
                        + ", ".join(sorted(ALLOWED_FONT_RES))
                    )

    if path.endswith(".xml"):
        for m in re.finditer(r'(?:android|app):fontFamily\s*=\s*"([^"]+)"', src):
            if m.group(1) != f"@font/{FONT_RES}":
                problems.append(f'XML fontFamily="{m.group(1)}" — must be "@font/{FONT_RES}"')
        if "/res/font/" in path.replace(os.sep, "/") and FONT_RES not in os.path.basename(path) and "<font" in src:
            problems.append(f"font resource other than {FONT_RES} — Vazirmatn Regular is the only bundled font")

    if problems:
        print(
            "Font check (Vazirmatn Regular, CLAUDE.md §5c) failed for " + path + ":\n- "
            + "\n- ".join(problems),
            file=sys.stderr,
        )
        return 2

    # Gentle reminder while the font file is still missing from the repo.
    if is_typography and "FontFamily.Default" in src:
        print(json.dumps({
            "hookSpecificOutput": {
                "hookEventName": "PostToolUse",
                "additionalContext": (
                    f"Reminder: res/font/{FONT_RES}.ttf is not bundled yet; RitmeFontFamily "
                    "still falls back to FontFamily.Default. Ship the Vazirmatn Regular file "
                    "and switch to FontFamily(Font(R.font.vazirmatn_regular))."
                ),
            }
        }))
    return 0

if __name__ == "__main__":
    sys.exit(main())
