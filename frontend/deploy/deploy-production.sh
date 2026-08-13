#!/usr/bin/env bash

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly FRONTEND_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
readonly TARGET_DIR="/opt/ts-cloud/frontend"
readonly BACKUP_ROOT="${XDG_STATE_HOME:-$HOME/.local/state}/ts-cloud/frontend-backups"
readonly RELEASE_ID="$(date -u +%Y%m%dT%H%M%SZ)"
readonly BACKUP_DIR="$BACKUP_ROOT/$RELEASE_ID"

for command_name in npm rsync curl; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done

if [[ ! -d "$TARGET_DIR" ]]; then
  echo "Production target does not exist: $TARGET_DIR" >&2
  exit 1
fi

cd "$FRONTEND_DIR"

echo "Running production checks..."
npm ci
npm run lint
npm run build
npm audit --audit-level=high

mkdir -p "$BACKUP_ROOT"
cp -a "$TARGET_DIR" "$BACKUP_DIR"
echo "Backup created: $BACKUP_DIR"

echo "Deploying frontend assets..."
rsync --archive --delete-delay --delay-updates "$FRONTEND_DIR/dist/" "$TARGET_DIR/"

echo "Verifying production URL..."
curl --fail --silent --show-error --location --max-time 20 \
  --output /dev/null https://cloud.tscommunication.com.bd/

echo "Deployment complete: $RELEASE_ID"
echo "Rollback with: $SCRIPT_DIR/rollback-production.sh $BACKUP_DIR"
