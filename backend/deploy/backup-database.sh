#!/usr/bin/env bash
set -Eeuo pipefail

readonly ENV_FILE="${TS_CLOUD_ENV_FILE:-/opt/ts-cloud/backend/.env}"
readonly BACKUP_ROOT="${TS_CLOUD_BACKUP_ROOT:-/var/backups/ts-cloud}"
readonly RETENTION_DAYS="${TS_CLOUD_BACKUP_RETENTION_DAYS:-14}"

umask 077
[[ -r "$ENV_FILE" ]] || { echo "Environment file is not readable: $ENV_FILE" >&2; exit 1; }

read_env_value() {
  local key="$1" value
  value="$(sed -n "s/^${key}=//p" "$ENV_FILE" | tail -n 1)"
  value="${value%\"}"; value="${value#\"}"
  value="${value%\'}"; value="${value#\'}"
  printf '%s' "$value"
}

DB_TYPE="${DB_TYPE:-$(read_env_value DB_TYPE)}"
DB_PATH="${DB_PATH:-$(read_env_value DB_PATH)}"
DATABASE_URL="${DATABASE_URL:-$(read_env_value DATABASE_URL)}"
DB_DSN="${DB_DSN:-$(read_env_value DB_DSN)}"

db_type="${DB_TYPE:-sqlite}"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$BACKUP_ROOT"

case "$db_type" in
  sqlite)
    [[ -n "${DB_PATH:-}" ]] || { echo "DB_PATH is required" >&2; exit 1; }
    final="$BACKUP_ROOT/ts-cloud-sqlite-$stamp.db"
    temporary="$final.partial"
    sqlite3 "$DB_PATH" ".timeout 10000" ".backup '$temporary'"
    ;;
  postgres)
    database_url="${DATABASE_URL:-$DB_DSN}"
    [[ -n "$database_url" ]] || { echo "DATABASE_URL or DB_DSN is required" >&2; exit 1; }
    final="$BACKUP_ROOT/ts-cloud-postgres-$stamp.dump"
    temporary="$final.partial"
    pg_dump --dbname="$database_url" --format=custom --compress=9 --file="$temporary"
    ;;
  *) echo "Unsupported DB_TYPE: $db_type" >&2; exit 1 ;;
esac

mv "$temporary" "$final"
(cd "$BACKUP_ROOT" && sha256sum "$(basename "$final")" > "$(basename "$final").sha256")
find "$BACKUP_ROOT" -maxdepth 1 -type f -name 'ts-cloud-*' -mtime "+$RETENTION_DAYS" -delete
echo "Backup complete: $final"
