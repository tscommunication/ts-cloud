#!/usr/bin/env bash

set -Eeuo pipefail

readonly TARGET_DIR="/opt/ts-cloud/frontend"
readonly BACKUP_ROOT="${XDG_STATE_HOME:-$HOME/.local/state}/ts-cloud/frontend-backups"
readonly BACKUP_DIR="${1:-}"

if [[ -z "$BACKUP_DIR" ]]; then
  echo "Usage: $0 <backup-directory>" >&2
  exit 1
fi

case "$BACKUP_DIR" in
  "$BACKUP_ROOT"/*) ;;
  *)
    echo "Backup must be inside: $BACKUP_ROOT" >&2
    exit 1
    ;;
esac

if [[ ! -f "$BACKUP_DIR/index.html" || ! -d "$BACKUP_DIR/assets" ]]; then
  echo "Invalid frontend backup: $BACKUP_DIR" >&2
  exit 1
fi

rsync --archive --delete-delay --delay-updates "$BACKUP_DIR/" "$TARGET_DIR/"

curl --fail --silent --show-error --location --max-time 20 \
  --output /dev/null https://cloud.tscommunication.com.bd/

echo "Rollback complete: $BACKUP_DIR"
