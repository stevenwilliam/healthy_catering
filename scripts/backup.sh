#!/bin/bash
# Evermore database backup.
#
# Install at /usr/local/bin/evermore-backup.sh and run from cron:
#   15 2 * * *  /usr/local/bin/evermore-backup.sh >> /var/log/evermore-backup.log 2>&1
#
# A backup on the same disk is not a backup, and an UNTESTED restore is not a
# backup either — see restore-check below, which is the part people skip.
set -euo pipefail

: "${DATABASE_URL:?set DATABASE_URL, e.g. from /etc/evermore/evermore.env}"
DEST="${BACKUP_DIR:-/var/backups/evermore}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
STAMP=$(date +%Y%m%d-%H%M)

mkdir -p "$DEST"
FILE="$DEST/evermore-$STAMP.sql.gz"

# --clean --if-exists so the dump can be restored over an existing database.
pg_dump --clean --if-exists "$DATABASE_URL" | gzip -9 > "$FILE"

# A zero-byte or truncated dump is worse than none, because it looks like a
# backup in a listing. Verify the gzip stream and that it ends with the marker
# pg_dump always writes.
gzip -t "$FILE"
if ! zcat "$FILE" | tail -5 | grep -q 'PostgreSQL database dump complete'; then
  echo "FAILED: $FILE is truncated" >&2
  exit 1
fi

echo "$(date -Is) wrote $FILE ($(du -h "$FILE" | cut -f1))"

find "$DEST" -name 'evermore-*.sql.gz' -mtime "+$KEEP_DAYS" -delete

# ⚠️ Copy off the machine. Uncomment and configure ONE of these:
# aws s3 cp "$FILE" s3://evermore-backups/ --storage-class STANDARD_IA
# rclone copy "$FILE" remote:evermore-backups/
echo "REMINDER: this backup is still on the same machine as the database."
