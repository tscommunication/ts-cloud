#!/usr/bin/env bash
set -Eeuo pipefail

readonly ENV_FILE="${TS_CLOUD_ENV_FILE:-/opt/ts-cloud/backend/.env}"
backup_file="${1:-}"
confirmation="${2:-}"

[[ -f "$backup_file" ]] || { echo "Usage: $0 <backup-file> --confirm-restore" >&2; exit 1; }
[[ "$confirmation" == "--confirm-restore" ]] || { echo "Restore requires --confirm-restore" >&2; exit 1; }
[[ -r "$ENV_FILE" ]] || { echo "Environment file is not readable: $ENV_FILE" >&2; exit 1; }
if [[ -f "$backup_file.sha256" ]]; then
  backup_dir="$(dirname "$backup_file")"
  (cd "$backup_dir" && sha256sum --check "$(basename "$backup_file").sha256")
fi

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

if [[ -n "$DB_PATH" && "$DB_PATH" != /* ]]; then
  backend_root="$(cd "$(dirname "$0")/.." && pwd)"
  DB_PATH="$backend_root/$DB_PATH"
fi

db_type="${DB_TYPE:-sqlite}"
systemctl stop ts-cloud.service
trap 'systemctl start ts-cloud.service' EXIT

case "$db_type" in
  sqlite)
    [[ -n "${DB_PATH:-}" ]] || { echo "DB_PATH is required" >&2; exit 1; }
    cp -a "$DB_PATH" "$DB_PATH.before-restore-$(date -u +%Y%m%dT%H%M%SZ)"
    install -o tscloud -g tscloud -m 0600 "$backup_file" "$DB_PATH"
    ;;
  postgres)
    database_url="${DATABASE_URL:-$DB_DSN}"
    [[ -n "$database_url" ]] || { echo "DATABASE_URL or DB_DSN is required" >&2; exit 1; }
    pg_restore --dbname="$database_url" --clean --if-exists --no-owner --no-privileges "$backup_file"
    ;;
  *) echo "Unsupported DB_TYPE: $db_type" >&2; exit 1 ;;
esac

echo "Restore complete: $backup_file"
