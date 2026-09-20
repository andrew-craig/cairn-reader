#!/bin/bash
# Nightly pg_dumpall backup for the cairn-selfhost stack.
#
# Usage: run directly, or schedule via cron, e.g.:
#   0 2 * * * BACKUP_DIR=/home/operator/cairn-backups /path/to/backup.sh >> /home/operator/cairn-backups/backup.log 2>&1
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

BACKUP_DIR="${BACKUP_DIR:-/var/backups/cairn}"
DB_USER="${DB_USER:-cairn}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

OUT_FILE="$BACKUP_DIR/cairn_all_$DATE.sql.gz"
TMP_FILE="$OUT_FILE.tmp"

if docker compose --project-directory "$COMPOSE_DIR" -f "$COMPOSE_DIR/docker-compose.yml" \
     exec -T cairn-db pg_dumpall -U "$DB_USER" | gzip > "$TMP_FILE"; then
  mv "$TMP_FILE" "$OUT_FILE"
else
  rm -f "$TMP_FILE"
  echo "Backup failed: pg_dumpall did not complete" >&2
  exit 1
fi

# Keep only the last $RETENTION_DAYS days of backups
find "$BACKUP_DIR" -name "cairn_all_*.sql.gz" -mtime "+$RETENTION_DAYS" -delete

echo "Backup completed: $DATE"
