#!/bin/sh
# Create a source-consistent PostgreSQL backup and prove it restores in isolation.
# Connection must be provided through PGHOST, PGPORT, PGUSER, PGDATABASE, PGPASSWORD.
set -eu
umask 077
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
root="${SKILLTREE_BACKUP_ROOT:-$HOME/.local/share/skilltree-cutover/backups}"
mkdir -p "$root"; chmod 700 "$root"
backup=$(mktemp -d "$root/pg-$(date -u +%Y%m%dT%H%M%SZ).XXXXXX")
tmp=$(mktemp -d "${TMPDIR:-/tmp}/skilltree-pgrestore.XXXXXX")
pgbin=${PGBIN:-/usr/lib/postgresql/18/bin}
started=0
cleanup(){ if [ "$started" -eq 1 ]; then "$pgbin/pg_ctl" -D "$tmp/cluster" -m fast -w stop >/dev/null 2>&1 || true; fi; case "$tmp" in /tmp/skilltree-pgrestore.*) rm -rf -- "$tmp" ;; *) echo "temporary restore files retained at $tmp" >&2 ;; esac; }
trap cleanup EXIT INT TERM
printf 'backup_directory=%s\n' "$backup"
date -u '+snapshot_started_utc=%Y-%m-%dT%H:%M:%SZ' | tee "$backup/metadata.txt"
pg_dump -Fc --no-owner --no-privileges --file="$backup/production.dump"
unset PGPASSWORD
sha256sum "$backup/production.dump" > "$backup/SHA256SUMS"
pg_restore --list "$backup/production.dump" > "$backup/archive-contents.txt"
mkdir -m 700 "$tmp/socket"
"$pgbin/initdb" -D "$tmp/cluster" -U postgres --auth=trust --no-locale --encoding=UTF8 > "$tmp/initdb.log"
"$pgbin/pg_ctl" -D "$tmp/cluster" -l "$tmp/postgres.log" -o "-k $tmp/socket -p 55479 -c listen_addresses=''" -w start >/dev/null
started=1
pg_restore --exit-on-error --no-owner --no-privileges --host="$tmp/socket" --port=55479 --username=postgres --dbname=postgres "$backup/production.dump"
psql -X -v ON_ERROR_STOP=1 -h "$tmp/socket" -p 55479 -U postgres -d postgres -A -t > "$backup/restored-counts.txt" <<'SQL'
SELECT format('SELECT %L, count(*) FROM %I.%I;',tablename,schemaname,tablename) FROM pg_catalog.pg_tables WHERE schemaname='public' ORDER BY tablename \gexec
SQL
SKILLTREE_SOURCE_DATABASE_URL="host=$tmp/socket port=55479 user=postgres dbname=postgres sslmode=disable" SKILLTREE_MIGRATION_OUTPUT="$backup/restore-verified.sqlite" "$repo/bin/skilltree-migrate" > "$backup/sqlite-migration-report.json"
SKILLTREE_MIGRATION_OUTPUT="$backup/restore-verified.sqlite" "$repo/bin/skilltree-repair" > "$backup/sqlite-repair-report.json"
SKILLTREE_DB="$backup/restore-verified.sqlite" SKILLTREE_BACKUP_OUTPUT="$backup/sqlite-backup.sqlite" "$repo/bin/skilltree-backup" > "$backup/sqlite-backup-report.txt"
BACKUP_DIR="$backup" python3 - <<'PY'
import hashlib,json,os,sqlite3
from pathlib import Path
p=Path(os.environ['BACKUP_DIR']); report=json.loads((p/'sqlite-migration-report.json').read_text()); assert all(t['verified'] for t in report['tables']); assert report['integrity']=='ok'
a=sqlite3.connect(f'file:{p}/restore-verified.sqlite?mode=ro',uri=True);b=sqlite3.connect(f'file:{p}/sqlite-backup.sqlite?mode=ro',uri=True)
assert a.execute('pragma integrity_check').fetchone()[0]=='ok' and b.execute('pragma integrity_check').fetchone()[0]=='ok'
tables=[r[0] for r in a.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")]
for table in tables:
 def hashes(conn):
  return sorted(hashlib.sha256(json.dumps(row,ensure_ascii=False,default=str,separators=(',',':')).encode()).digest() for row in conn.execute('select * from "'+table.replace('"','""')+'"'))
 assert hashes(a)==hashes(b),table
print('restored_tables=%d rows=%d sqlite_backup_row_hashes=matched'%(len(report['tables']),sum(t['rows'] for t in report['tables'])))
print('orphan_members_archived=%d'%json.loads((p/'sqlite-repair-report.json').read_text())['archivedMemberships'])
a.close();b.close()
PY
date -u '+verification_finished_utc=%Y-%m-%dT%H:%M:%SZ' | tee -a "$backup/metadata.txt"
