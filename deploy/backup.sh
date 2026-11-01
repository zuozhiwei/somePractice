#!/bin/sh
set -eu
umask 077
backup_dir="${SKILLTREE_BACKUP_DIR:-/var/lib/skilltree/backups}"
mkdir -p "$backup_dir"
SKILLTREE_BACKUP_OUTPUT="$backup_dir/skilltree-$(date -u +%Y%m%dT%H%M%SZ)-$$.db"
export SKILLTREE_BACKUP_OUTPUT
exec /opt/skilltree/skilltree-backup
