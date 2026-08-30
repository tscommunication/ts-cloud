#!/usr/bin/env bash

# Read-only production cutover audit. It deliberately performs no database,
# MikroTik, service, or filesystem mutation.

APP_ROOT="/opt/ts-cloud/backend"
SERVICE="ts-cloud.service"
BACKUP_ROOT="/var/backups/ts-cloud"
EXIT_CODE=0

check() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf 'PASS  %s\n' "$label"
  else
    printf 'FAIL  %s\n' "$label"
    EXIT_CODE=1
  fi
}

printf '%s\n' '=== TS-Cloud production cutover preflight (read-only) ==='
check 'Backend binary exists' test -x "$APP_ROOT/ts-cloud"
# The environment file is intentionally often unreadable by the operator so
# credentials cannot be exposed. Existence is sufficient for this audit.
check 'Backend environment file exists' test -e "$APP_ROOT/.env"
check 'Backend service is active' systemctl is-active --quiet "$SERVICE"
check 'Local backend health endpoint responds' curl --fail --silent --max-time 10 http://127.0.0.1:8080/health
check 'Backup directory exists' test -d "$BACKUP_ROOT"

if [[ -r "$APP_ROOT/.env" ]]; then
  db_type="$(sed -n 's/^DB_TYPE=//p' "$APP_ROOT/.env" | tail -1 | tr -d '[:space:]')"
  db_path="$(sed -n 's/^DB_PATH=//p' "$APP_ROOT/.env" | tail -1 | tr -d '[:space:]')"
  printf 'INFO  Configured database type: %s\n' "${db_type:-sqlite (default)}"
  if [[ "${db_type:-sqlite}" == "sqlite" && -n "$db_path" ]]; then
    check 'Configured SQLite database exists' test -r "$db_path"
  fi
fi

printf '%s\n' ''
printf '%s\n' 'Before deleting test data or importing legacy users:'
printf '%s\n' '1. Create and verify a database backup.'
printf '%s\n' '2. Export the planned source CSV/XLSX and validate its preview in TS-Cloud.'
printf '%s\n' '3. Confirm each target MikroTik router is ACTIVE with API credentials configured.'
printf '%s\n' '4. Test import 2–3 users first; do not change existing RouterOS PPP passwords.'
printf '%s\n' '5. Only after acceptance, run an explicitly reviewed cleanup procedure and full import.'

exit "$EXIT_CODE"
