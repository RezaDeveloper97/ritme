#!/usr/bin/env python3
"""PreToolUse hook (Edit|Write): blocks hardcoded secrets/tokens in source files."""
import json
import re
import sys

PATTERNS = [
    (r"(?i)(api[_-]?key|client[_-]?secret|password|passwd|bearer)\s*[:=]\s*[\"'][^\"']{8,}[\"']",
     "hardcoded credential assignment"),
    (r"eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}", "embedded JWT token"),
    (r"-----BEGIN (RSA |EC )?PRIVATE KEY-----", "private key material"),
    (r"AKIA[0-9A-Z]{16}", "AWS access key"),
]
ALLOWLIST_HINTS = ("example", "placeholder", "your_", "xxx", "<", "sample", "fake", "test_")

def main():
    data = json.load(sys.stdin)
    tool_input = data.get("tool_input", {})
    path = tool_input.get("file_path", "")
    if not path.endswith((".kt", ".kts", ".java", ".xml", ".json", ".properties")):
        return 0
    if "/test" in path.lower() or "check_secrets" in path:
        return 0
    text = tool_input.get("content", "") or tool_input.get("new_string", "")
    findings = []
    for pattern, label in PATTERNS:
        for m in re.finditer(pattern, text):
            snippet = m.group(0)
            if any(h in snippet.lower() for h in ALLOWLIST_HINTS):
                continue
            findings.append(f"{label}: …{snippet[:60]}…")
    if findings:
        print(
            "BLOCKED — possible hardcoded secret in "
            + path + ":\n- " + "\n- ".join(findings)
            + "\nLoad secrets from local.properties/BuildConfig or Android Keystore, "
            "never commit them in source. If this is a false positive, tell the user why.",
            file=sys.stderr,
        )
        return 2
    return 0

if __name__ == "__main__":
    sys.exit(main())
