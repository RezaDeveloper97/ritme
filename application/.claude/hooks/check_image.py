#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write on .kt): enforces the hand-written image-loading rules
(CLAUDE.md §3/§5) — no Coil/Glide, decodes go through RitmeImageLoader downsampled."""
import json
import re
import sys

# Files allowed to call BitmapFactory directly.
ALLOWED = ("/adapter/outbound/image/", "CaptchaImage.kt")


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

    if re.search(r"^import\s+(coil|com\.bumptech\.glide)", src, re.MULTILINE):
        problems.append("imports Coil/Glide — banned by §3; use RitmeImageLoader/NetworkImage")

    if "BitmapFactory" in src and not any(a in path for a in ALLOWED):
        problems.append(
            "decodes bitmaps outside adapter/outbound/image — remote images must go "
            "through ImageLoader (NetworkImage composable) so caching + downsampling apply (§5)"
        )

    if "/adapter/outbound/image/" in path and "BitmapFactory.decode" in src:
        if "inSampleSize" not in src or "inJustDecodeBounds" not in src:
            problems.append(
                "decodes without a two-pass inJustDecodeBounds + inSampleSize downsample — "
                "full-resolution decodes violate the §5 low-end budget"
            )

    if problems:
        print(
            "Image-loading check (CLAUDE.md §3/§5) found issues in " + path + ":\n- "
            + "\n- ".join(problems),
            file=sys.stderr,
        )
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
