#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Ritme site — cPanel release package (L10-01). Guide: docs/DEPLOY-CPANEL.md
#
#   bash deploy/build-cpanel.sh                       # layout docroot  → dist/ritme-site-<build-id>.zip
#   bash deploy/build-cpanel.sh --layout=public_html  # layout public_html → dist/ritme-site-<build-id>-public_html.zip
#   bash deploy/build-cpanel.sh --layout=all          # both packages from one build
#   bash deploy/build-cpanel.sh --dry-run             # check tools + show what would be packaged, build nothing
#
# Options:
#   --ref=<git ref>      commit to package from a clean `git archive` (default HEAD)
#   --worktree           package the working tree instead (tracked + untracked, .gitignore respected) — local testing
#   --layout=…           docroot (default) | public_html | all
#   --app-dir=<name>     app folder name of the public_html layout (default ritme)
#   --doc-root=<name>    document-root folder name of the public_html layout (default public_html)
#   --allow-no-critical  do not fail when critical CSS cannot be generated (no Chrome); pages then use the blocking
#                        stylesheet — never for a production release
#   --keep               keep the staging folder (printed at the end)
#
# Steps: export source → composer install --no-dev --optimize-autoloader --classmap-authoritative (platform PHP 8.2
# from composer.json) → temporary SQLite + .env so the build can render pages → npm ci → CRITICAL_STRICT=1
# npm run build (Vite → public/build incl. build-id.json, service worker public/sw.js, critical CSS
# public/build/critical/) → Filament assets → strip dev files and build state → zip + sha256.
# Nothing is uploaded anywhere: the result stays in dist/ (gitignored). Needs: git, php ≥ 8.2, composer, node + npm,
# zip, shasum, Google Chrome (critical CSS; CHROME_PATH to override).
# Compatible with macOS' bash 3.2.
# -----------------------------------------------------------------------------
set -euo pipefail

SITE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATES="$SITE_DIR/deploy/cpanel"
DIST_DIR="$SITE_DIR/dist"

REF="HEAD"
SOURCE="ref"
LAYOUT="docroot"
APP_DIR="ritme"
DOC_ROOT="public_html"
DRY_RUN=0
STRICT=1
KEEP=0

for arg in "$@"; do
    case "$arg" in
        --ref=*) REF="${arg#*=}"; SOURCE="ref" ;;
        --worktree) SOURCE="worktree" ;;
        --layout=*) LAYOUT="${arg#*=}" ;;
        --app-dir=*) APP_DIR="${arg#*=}" ;;
        --doc-root=*) DOC_ROOT="${arg#*=}" ;;
        --allow-no-critical) STRICT=0 ;;
        --dry-run) DRY_RUN=1 ;;
        --keep) KEEP=1 ;;
        -h|--help) sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "Unknown option: $arg (see --help)" >&2; exit 2 ;;
    esac
done

case "$LAYOUT" in docroot|public_html|all) ;; *) echo "--layout must be docroot, public_html or all" >&2; exit 2 ;; esac
case "$APP_DIR$DOC_ROOT" in *[!A-Za-z0-9._-]*) echo "--app-dir / --doc-root: letters, digits, . _ - only" >&2; exit 2 ;; esac

step() { printf '\n\033[1;35m==> %s\033[0m\n' "$*"; }
info() { printf '    %s\n' "$*"; }
die() { printf '\033[1;31m✘ %s\033[0m\n' "$*" >&2; exit 1; }

# Paths (relative to the site root) that never ship: dev tooling, sources compiled into public/build, docs, tests,
# local state. Runtime needs resources/views + resources/svg + lang; resources/{css,js,fonts} only feed Vite.
EXCLUDES="tests design tasks docs tools deploy node_modules dist .git .github .idea .vscode .fleet .zed .claude
.playwright-mcp .editorconfig .gitattributes .gitignore .env .env.example .env.backup .env.production .phpunit.cache
.phpunit.result.cache .phpactor.json phpunit.xml phpstan.neon phpstan.neon.dist pint.json vite.config.js package.json
package-lock.json CLAUDE.md README.md auth.json Homestead.json Homestead.yaml resources/css resources/js resources/fonts
public/hot public/storage storage/pail storage/framework/testing"

# ---------------------------------------------------------------------------------------------------------------
step "Preflight"
REPO_ROOT="$(git -C "$SITE_DIR" rev-parse --show-toplevel)"
PREFIX="$(git -C "$SITE_DIR" rev-parse --show-prefix)"   # "laravel-site/" inside the monorepo, "" standalone
PREFIX="${PREFIX%/}"
missing=0
for tool in git php composer node npm zip shasum; do
    if command -v "$tool" >/dev/null 2>&1; then info "ok   $tool"; else info "MISS $tool"; missing=1; fi
done
php -r 'exit(version_compare(PHP_VERSION, "8.2.0", ">=") ? 0 : 1);' || die "php ≥ 8.2 required on the build machine"
CHROME="${CHROME_PATH:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
if [ -x "$CHROME" ]; then info "ok   chrome ($CHROME)"; else
    info "MISS chrome ($CHROME) — critical CSS needs it (set CHROME_PATH)"; [ "$STRICT" = 1 ] && missing=1
fi
for f in root.htaccess index.public_html.php public-path.php; do
    [ -f "$TEMPLATES/$f" ] || die "missing template deploy/cpanel/$f"
done
grep -q '"php": "8.2' "$SITE_DIR/composer.json" || die "composer.json must pin config.platform.php to 8.2 (cPanel)"
COMMIT="$(git -C "$SITE_DIR" rev-parse --short=8 "$REF^{commit}")" || die "unknown ref $REF"
DIRTY=""
if [ "$SOURCE" = "worktree" ] && [ -n "$(git -C "$SITE_DIR" status --porcelain -- .)" ]; then DIRTY="-dirty"; fi
info "source: $([ "$SOURCE" = ref ] && echo "git archive $REF ($COMMIT)" || echo "working tree ($COMMIT$DIRTY)")"
info "layout: $LAYOUT$([ "$LAYOUT" != docroot ] && echo " (app → $APP_DIR/, public → $DOC_ROOT/)")"

if [ "$DRY_RUN" = 1 ]; then
    step "Dry run — files that would be packaged (before vendor/ and the build)"
    if [ "$SOURCE" = ref ]; then
        list="$(git -C "$REPO_ROOT" ls-tree -r --name-only "$REF" -- "${PREFIX:-.}")"
    else
        list="$(cd "$SITE_DIR" && git ls-files -co --exclude-standard | sed "s#^#${PREFIX:+$PREFIX/}#")"
    fi
    total=0; kept=0
    while IFS= read -r path; do
        [ -n "$path" ] || continue
        rel="${path#"${PREFIX:+$PREFIX/}"}"
        total=$((total + 1))
        skip=0
        for ex in $EXCLUDES; do
            case "$rel" in "$ex"|"$ex"/*) skip=1; break ;; esac
        done
        case "$rel" in database/*.sqlite*|*.log) skip=1 ;; esac
        [ "$skip" = 0 ] && kept=$((kept + 1))
    done <<EOF_LIST
$list
EOF_LIST
    info "$kept of $total source files kept; top level:"
    echo "$list" | sed "s#^${PREFIX:+$PREFIX/}##" | cut -d/ -f1 | sort -u | while IFS= read -r top; do
        mark="+"; for ex in $EXCLUDES; do [ "$top" = "$ex" ] && mark="-"; done; info "  $mark $top"
    done
    info "+ vendor/ (composer --no-dev), public/build/ (Vite + critical/ + build-id.json), public/sw.js, public/{css,js,fonts}/filament"
    info "output: $DIST_DIR/ritme-site-<build-id>[-public_html].zip + .sha256"
    [ "$missing" = 0 ] || die "missing tools (see above)"
    printf '\n\033[1;32m✔ dry run ok\033[0m\n'
    exit 0
fi
[ "$missing" = 0 ] || die "missing tools (see above)"

# ---------------------------------------------------------------------------------------------------------------
WORK="$(mktemp -d "${TMPDIR:-/tmp}/ritme-site-build.XXXXXX")"
STAGE="$WORK/app"
mkdir -p "$STAGE"
cleanup() {
    if [ "$KEEP" = 1 ]; then echo "staging kept: $WORK"; else rm -rf "$WORK"; fi
}
trap cleanup EXIT

step "Export source"
if [ "$SOURCE" = ref ]; then
    git -C "$REPO_ROOT" archive --format=tar "$REF" -- "${PREFIX:-.}" | tar -x -C "$WORK/"
    if [ -n "$PREFIX" ]; then rm -rf "$STAGE"; mv "$WORK/$PREFIX" "$STAGE"; fi
else
    (cd "$SITE_DIR" && git ls-files -co --exclude-standard -z | xargs -0 -I{} sh -c 'test -e "$1" && printf "%s\0" "$1"' _ {} \
        | tar --null -T - -cf -) | tar -x -C "$STAGE"
fi
info "$(find "$STAGE" -type f | wc -l | tr -d ' ') files"

step "Composer (no dev, classmap-authoritative, platform PHP 8.2)"
(cd "$STAGE" && COMPOSER_NO_DEV=1 composer install --no-dev --optimize-autoloader --classmap-authoritative \
    --no-interaction --no-progress --prefer-dist --quiet)

step "Temporary build database (pages are rendered for critical CSS)"
BUILD_DB="$STAGE/database/build.sqlite"
: > "$BUILD_DB"
cat > "$STAGE/.env" <<EOF_ENV
APP_NAME=Ritme
APP_ENV=production
APP_KEY=base64:$(php -r 'echo base64_encode(random_bytes(32));')
APP_DEBUG=false
APP_URL=http://127.0.0.1
APP_CANONICAL_REDIRECT=false
APP_LOCALE=fa
APP_TIMEZONE=Asia/Tehran
LOG_CHANNEL=stderr
LOG_LEVEL=error
DB_CONNECTION=sqlite
DB_DATABASE=$BUILD_DB
SESSION_DRIVER=array
CACHE_STORE=array
QUEUE_CONNECTION=sync
MAIL_MAILER=array
PAGE_CACHE_ENABLED=false
EOF_ENV
(cd "$STAGE" && php artisan migrate --force --seed --no-interaction >/dev/null \
    && for s in BlogSeeder DirectorySeeder ShopSeeder; do php artisan db:seed --class="$s" --force --no-interaction >/dev/null; done)
info "migrated + seeded (production defaults + demo content for the template samples)"

step "Frontend build (npm ci → vite → service worker → critical CSS)"
# The staging copy has no .git: hand the commit to the PWA build id (YYYYMMDDHHmmss-<sha>, tools/build-sw.mjs).
export BUILD_ID="$(date -u +%Y%m%d%H%M%S)-$COMMIT"
(cd "$STAGE" && npm ci --no-audit --no-fund --prefer-offline --loglevel=error)
if [ "$STRICT" = 1 ]; then
    (cd "$STAGE" && CRITICAL_STRICT=1 CHROME_PATH="$CHROME" npm run build --silent)
else
    (cd "$STAGE" && CHROME_PATH="$CHROME" npm run build --silent)
fi
[ -f "$STAGE/public/build/manifest.json" ] || die "public/build/manifest.json missing"
[ -f "$STAGE/public/build/build-id.json" ] || die "public/build/build-id.json missing"
[ -f "$STAGE/public/sw.js" ] || die "public/sw.js missing"
if [ "$STRICT" = 1 ]; then [ -f "$STAGE/public/build/critical/manifest.json" ] || die "critical CSS missing"; fi
(cd "$STAGE" && php artisan filament:assets --no-interaction >/dev/null)
[ -d "$STAGE/public/js/filament" ] || die "Filament assets missing"
BUILD_ID="$(php -r 'echo json_decode(file_get_contents($argv[1]), true)["build_id"] ?? "";' "$STAGE/public/build/build-id.json")"
[ -n "$BUILD_ID" ] || die "empty build id"
info "build id $BUILD_ID"

step "Strip dev files and build state"
for ex in $EXCLUDES; do rm -rf "${STAGE:?}/$ex"; done
rm -f "$STAGE"/database/*.sqlite "$STAGE"/database/*.sqlite-* "$STAGE"/database/*.sqlite3
find "$STAGE/storage" -type f ! -name .gitignore -delete
find "$STAGE/bootstrap/cache" -type f ! -name .gitignore ! -name packages.php ! -name services.php -delete
rm -rf "$STAGE/public/media"; mkdir -p "$STAGE/public/media"
find "$STAGE" -name .DS_Store -delete
find "$STAGE" -type d -exec chmod 755 {} +
find "$STAGE" -type f -exec chmod 644 {} +
chmod 755 "$STAGE/artisan"
VERSION_JSON="{\"build_id\":\"$BUILD_ID\",\"commit\":\"$COMMIT$DIRTY\",\"ref\":\"$( [ "$SOURCE" = ref ] && echo "$REF" || echo worktree)\",\"built_at\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"php_platform\":\"8.2\"}"
echo "$VERSION_JSON" > "$STAGE/release.json"

mkdir -p "$DIST_DIR"
package() { # $1 = layout
    local layout="$1" out="$WORK/out-$1" name
    rm -rf "$out"; mkdir -p "$out"
    if [ "$layout" = docroot ]; then
        name="ritme-site-$BUILD_ID$DIRTY"
        cp -R "$STAGE/." "$out/"
        cp "$TEMPLATES/root.htaccess" "$out/.htaccess"
    else
        name="ritme-site-$BUILD_ID$DIRTY-public_html"
        cp -R "$STAGE" "$out/$APP_DIR"
        mv "$out/$APP_DIR/public" "$out/$DOC_ROOT"
        sed "s#__APP_DIR__#$APP_DIR#" "$TEMPLATES/index.public_html.php" > "$out/$DOC_ROOT/index.php"
        sed "s#__DOC_ROOT__#$DOC_ROOT#" "$TEMPLATES/public-path.php" > "$out/$APP_DIR/bootstrap/public-path.php"
        chmod 644 "$out/$DOC_ROOT/index.php" "$out/$APP_DIR/bootstrap/public-path.php"
    fi
    rm -f "$DIST_DIR/$name.zip" "$DIST_DIR/$name.zip.sha256"
    (cd "$out" && zip -qr -X -9 "$DIST_DIR/$name.zip" .)
    (cd "$DIST_DIR" && shasum -a 256 "$name.zip" > "$name.zip.sha256")
    info "$DIST_DIR/$name.zip ($(du -h "$DIST_DIR/$name.zip" | cut -f1 | tr -d ' '), $(unzip -Z1 "$DIST_DIR/$name.zip" | wc -l | tr -d ' ') entries)"
}

step "Package"
if [ "$LAYOUT" = docroot ] || [ "$LAYOUT" = all ]; then package docroot; fi
if [ "$LAYOUT" = public_html ] || [ "$LAYOUT" = all ]; then package public_html; fi

printf '\n\033[1;32m✔ done\033[0m — upload + install: docs/DEPLOY-CPANEL.md\n'
