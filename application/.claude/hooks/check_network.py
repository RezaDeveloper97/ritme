#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt): enforces the hand-written network layer rules
(CLAUDE.md §3/§5) — raw connections only inside the network/image adapters, no manual
Accept-Encoding (breaks transparent gzip), timeouts always configured."""
import json
import re
import sys

# The only packages allowed to open raw connections.
ALLOWED_DIRS = ("/adapter/outbound/network/", "/adapter/outbound/image/")


def main():
    data = json.load(sys.stdin)
    path = data.get("tool_input", {}).get("file_path", "")
    if not path.endswith(".kt") or "/test/" in path or "/androidTest/" in path:
        return 0
    try:
        with open(path, encoding="utf-8") as f:
            src = f.read()
    except OSError:
        return 0

    problems = []
    uses_raw_net = re.search(r"HttpURLConnection|\.openConnection\(|\bSocket\(|SSLSocket", src)

    if uses_raw_net and not any(d in path for d in ALLOWED_DIRS):
        problems.append(
            "raw connection code outside adapter/outbound/network|image — "
            "route it through HttpClient / ImageLoader (§2/§3)"
        )

    if uses_raw_net and any(d in path for d in ALLOWED_DIRS):
        if "connectTimeout" not in src or "readTimeout" not in src:
            problems.append(
                "opens a connection without setting connectTimeout/readTimeout (§5 sane timeouts)"
            )

    if re.search(r"""setRequestProperty\(\s*["']Accept-Encoding""", src):
        problems.append(
            "manually sets Accept-Encoding — this disables HttpURLConnection's "
            "transparent gzip decoding (§5); remove it"
        )

    if re.search(r"^import\s+(okhttp3|retrofit2)\.", src, re.MULTILINE):
        problems.append("imports OkHttp/Retrofit — banned by §3, use the hand-written HttpClient")

    if problems:
        print(
            "Network-layer check (CLAUDE.md §3/§5) found issues in " + path + ":\n- "
            + "\n- ".join(problems),
            file=sys.stderr,
        )
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
