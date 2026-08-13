#!/bin/bash
# Prove the newest backup actually restores.
#
# Run this monthly. A dump nobody has restored is a hypothesis, not a backup —
# and the moment you need it is the worst possible time to find that out.
#
# Restores run as the DATABASE SUPERUSER, not the application role: a --clean
# dump recreates extensions, and only a superuser may do that. This is normal
# for a restore and is why the application role is deliberately not one.
set -euo pipefail

DEST="${BACKUP_DIR:-/var/backups/evermore}"
TARGET_DB="${RESTORE_DB:-healthy_catering_test}"

LATEST=$(ls -t "$DEST"/evermore-*.sql.gz 2>/dev/null | head -1)
[ -n "$LATEST" ] || { echo "no backups in $DEST" >&2; exit 1; }

echo "restoring $LATEST into $TARGET_DB …"
# Reset the target first, so the check proves a restore ONTO A CLEAN DATABASE
# — which is the situation you are actually in when you need it.
sudo -u postgres psql -q -c "DROP DATABASE IF EXISTS ${TARGET_DB};"
sudo -u postgres psql -q -c "CREATE DATABASE ${TARGET_DB} OWNER healthy_catering;"
zcat "$LATEST" | sudo -u postgres psql -q -v ON_ERROR_STOP=1 -d "$TARGET_DB"

echo
echo "row counts after restore:"
sudo -u postgres psql -d "$TARGET_DB" -tAc "
  SELECT '  ' || relname || ': ' || n_live_tup
    FROM pg_stat_user_tables
   WHERE n_live_tup > 0 ORDER BY n_live_tup DESC LIMIT 8;"

echo
echo "running the security suite against the RESTORED copy…"
cd "$(dirname "$0")/.."
TEST_DATABASE_URL="${TEST_DATABASE_URL}" go test ./test/security/ -count=1

echo
echo "RESTORE VERIFIED: $LATEST"
