#!/usr/bin/env bash
# Render a canvas-v1 board to PNG with headless Chrome (reference image for fidelity checks).
# Usage: roadmap/bin/shot-board.sh <board file, e.g. nbl_Meno_Home.dc.html> [out.png]
# Default out: docs/qa/canvas/boards/<board>.png. Size comes from docs/design/canvas-v1/canvas.json.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
b="${1:?board file}"; D="$ROOT/docs/design/canvas-v1"
[ -f "$D/boards/$b" ] || { echo "no board $b" >&2; exit 1; }
out="${2:-$ROOT/docs/qa/canvas/boards/${b%.dc.html}.png}"; mkdir -p "$(dirname "$out")"
read -r w h < <(python3 -c "import json,sys;b=json.load(open('$D/canvas.json'))['boards']['$b'];print(b['w'],b['h'])")
CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
# LTR wrapper page with an exact-size iframe: RTL boards otherwise anchor to the right of headless Chrome's min width.
wrap="$(mktemp -t board).html"; trap 'rm -f "$wrap"' EXIT
printf '<!doctype html><html dir="ltr"><body style="margin:0"><iframe src="file://%s" width="%s" height="%s" style="border:0;display:block"></iframe></body></html>' "$D/boards/$b" "$w" "$h" > "$wrap"
"$CHROME" --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 --allow-file-access-from-files \
  --window-size="$w,$h" --virtual-time-budget=4000 --screenshot="$out" "file://$wrap" >/dev/null 2>&1
echo "$out (${w}x${h})"
