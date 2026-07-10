#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt): flags hexagonal-architecture violations (CLAUDE.md §2)."""
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
            src = f.read()
    except OSError:
        return 0

    problems = []
    imports = re.findall(r"^import\s+([\w.]+)", src, re.MULTILINE)

    if "/domain/" in path:
        for imp in imports:
            if imp.startswith(("android.", "androidx.", "java.io", "java.net",
                               "java.sql", "org.json")):
                problems.append(f"domain layer imports forbidden '{imp}' — domain must be pure Kotlin")
            if imp.startswith("kotlinx.") and not imp.startswith("kotlinx.coroutines"):
                problems.append(f"domain layer imports '{imp}' — only kotlinx.coroutines is allowed")
            if ".adapter." in imp or ".application." in imp:
                problems.append(f"domain imports outward layer '{imp}' — dependencies must point inward")

    if "/application/" in path:
        for imp in imports:
            if ".adapter." in imp or imp.startswith(("android.", "androidx.")):
                problems.append(f"application layer imports '{imp}' — must depend on domain only")

    if re.search(r"class\s+\w*ViewModel", src):
        for bad, why in [
            (r"HttpURLConnection|Socket\(", "networking"),
            (r"SQLiteOpenHelper|SQLiteDatabase|rawQuery", "SQL"),
            (r"JSONObject|JSONArray", "JSON parsing"),
        ]:
            if re.search(bad, src):
                problems.append(f"ViewModel does {why} directly — move it behind a port/adapter (§4 SRP)")

    if problems:
        print(
            "Architecture check (CLAUDE.md) found issues in " + path + ":\n- "
            + "\n- ".join(problems)
            + "\nFix these before finishing (see /hexagonal-check).",
            file=sys.stderr,
        )
        return 2
    return 0

if __name__ == "__main__":
    sys.exit(main())
