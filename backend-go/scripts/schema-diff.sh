#!/usr/bin/env bash
# schema-diff — prove the goose baseline and the Laravel migrations build the same schema.
#
#   make schema-diff                                  # Laravel-migrated DB vs goose DB (default)
#   scripts/schema-diff.sh --against staging.sql      # a mysqldump --no-data file vs goose DB
#
# Both sides are built in throw-away databases on the test MariaDB 11.4 (docker-compose.test.yml),
# dumped with the same mariadb-dump flags, normalised (AUTO_INCREMENT counters, the `migrations` /
# `goose_db_version` bookkeeping tables) and diffed. In the default mode seed rows are compared too:
# the `languages` rows (minus timestamps) and every other table's row count.
#
# Needs: docker, and for the default mode a local php (pdo_mysql) with backend/vendor installed.
# Exit 0 = identical, 1 = different, 2 = could not run.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
backend="$here/../backend"
compose=(docker compose -f "$here/docker-compose.test.yml")
port="${TEST_DB_PORT:-13317}"
root_pw="root" # docker-compose.test.yml MARIADB_ROOT_PASSWORD (throw-away container)
goose_pkg="github.com/pressly/goose/v3/cmd/goose@v3.26.0"
goose_tags="no_postgres no_sqlite3 no_mssql no_vertica no_clickhouse no_ydb no_libsql"

against=""
if [[ "${1:-}" == "--against" ]]; then
  against="${2:?usage: schema-diff.sh [--against dump.sql]}"
  [[ -r "$against" ]] || { echo "schema-diff: cannot read $against" >&2; exit 2; }
fi

suffix="$(date +%s)_$$"
db_ref="sd_ref_$suffix"
db_goose="sd_goose_$suffix"
work="$(mktemp -d)"

sql() { "${compose[@]}" exec -T mariadb mariadb -uroot -p"$root_pw" "$@"; }
cleanup() {
  sql -e "DROP DATABASE IF EXISTS \`$db_ref\`; DROP DATABASE IF EXISTS \`$db_goose\`;" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT

"${compose[@]}" up -d --wait mariadb >/dev/null 2>&1 || { echo "schema-diff: test MariaDB did not start" >&2; exit 2; }
for db in "$db_ref" "$db_goose"; do
  sql -e "CREATE DATABASE \`$db\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
done

# --- reference side --------------------------------------------------------------------------------
if [[ -n "$against" ]]; then
  echo "schema-diff: loading $against"
  sql "$db_ref" <"$against"
  ref_label="$(basename "$against")"
else
  echo "schema-diff: running the Laravel migrations (backend/, DB_CONNECTION=mysql)"
  [[ -f "$backend/vendor/autoload.php" ]] || { echo "schema-diff: backend/vendor missing (composer install)" >&2; exit 2; }
  # Real env vars win over backend/.env (Dotenv is immutable), so nothing local is touched.
  if ! (cd "$backend" && env APP_ENV=schemadiff APP_TIMEZONE=Asia/Tehran \
      DB_CONNECTION=mysql DB_HOST=127.0.0.1 DB_PORT="$port" DB_DATABASE="$db_ref" \
      DB_USERNAME=root DB_PASSWORD="$root_pw" \
      CACHE_STORE=array SESSION_DRIVER=array QUEUE_CONNECTION=sync LOG_CHANNEL=stderr \
      php artisan migrate --force --no-interaction >"$work/laravel.log" 2>&1); then
    cat "$work/laravel.log" >&2
    exit 2
  fi
  ref_label="laravel"
fi

# --- goose side ------------------------------------------------------------------------------------
echo "schema-diff: applying db/migrations with goose"
goose_dsn="root:$root_pw@tcp(127.0.0.1:$port)/$db_goose?parseTime=true&multiStatements=true"
if ! (cd "$here" && go run -tags "$goose_tags" "$goose_pkg" -dir db/migrations mysql "$goose_dsn" up >"$work/goose.log" 2>&1); then
  cat "$work/goose.log" >&2
  exit 2
fi

# --- compare ---------------------------------------------------------------------------------------
dump_schema() {
  "${compose[@]}" exec -T mariadb mariadb-dump -uroot -p"$root_pw" --no-data --skip-comments --compact \
    --skip-dump-date --ignore-table="$1.migrations" --ignore-table="$1.goose_db_version" "$1" |
    sed -E 's/ AUTO_INCREMENT=[0-9]+//'
}
seed_rows() {
  sql -N "$1" -e "SELECT code, name, english_name, direction, is_active, is_default, sort_order FROM languages ORDER BY id"
  local tables
  tables="$(sql -N -e "SELECT table_name FROM information_schema.tables WHERE table_schema='$1' AND table_name NOT IN ('migrations','goose_db_version','languages') ORDER BY table_name")"
  for t in $tables; do
    echo "$t $(sql -N "$1" -e "SELECT COUNT(*) FROM \`$t\`")"
  done
}

dump_schema "$db_ref" >"$work/$ref_label.schema.sql"
dump_schema "$db_goose" >"$work/goose.schema.sql"
if [[ -z "$against" ]]; then
  seed_rows "$db_ref" >"$work/$ref_label.rows.txt"
  seed_rows "$db_goose" >"$work/goose.rows.txt"
fi

tables="$(grep -c '^CREATE TABLE' "$work/goose.schema.sql" || true)"
status=0
(cd "$work" && diff -u "$ref_label.schema.sql" goose.schema.sql) || status=1
# A --no-data dump has no rows to compare; only the Laravel mode checks seed rows.
[[ -n "$against" ]] || (cd "$work" && diff -u "$ref_label.rows.txt" goose.rows.txt) || status=1

if [[ $status -eq 0 ]]; then
  scope="seed rows equal"
  [[ -z "$against" ]] || scope="schema only"
  echo "schema-diff: OK — $ref_label and goose baseline are identical ($tables tables, $scope; migrations/goose_db_version ignored)"
else
  echo "schema-diff: DIFFERENT — see the diff above (left: $ref_label, right: goose)" >&2
fi
exit $status
