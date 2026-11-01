#!/bin/sh
# Restore a PostgreSQL custom dump into an isolated local cluster and validate migration.
set -eu
[ "${SKILLTREE_VERIFY_DUMP:-}" ] && [ -f "$SKILLTREE_VERIFY_DUMP" ] || { echo 'SKILLTREE_VERIFY_DUMP is required' >&2;exit 1; }
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd);pgbin=${PGBIN:-/usr/lib/postgresql/18/bin};tmp=$(mktemp -d "${TMPDIR:-/tmp}/skilltree-dump-restore.XXXXXX");chmod 700 "$tmp";started=0
cleanup(){ if [ "$started" = 1 ];then "$pgbin/pg_ctl" -D "$tmp/cluster" -m fast -w stop >/dev/null 2>&1 || true;fi;case "$tmp" in /tmp/skilltree-dump-restore.*) rm -rf -- "$tmp";; *) printf 'Temporary restore retained at %s\n' "$tmp" >&2;; esac; };trap cleanup EXIT INT TERM
mkdir -m 700 "$tmp/socket"
"$pgbin/initdb" -D "$tmp/cluster" -U postgres --auth=trust --no-locale --encoding=UTF8 > "$tmp/initdb.log"
"$pgbin/pg_ctl" -D "$tmp/cluster" -l "$tmp/postgres.log" -o "-k $tmp/socket -p 55479 -c listen_addresses=''" -w start >/dev/null;started=1
pg_restore --exit-on-error --no-owner --no-privileges --host="$tmp/socket" --port=55479 --username=postgres --dbname=postgres "$SKILLTREE_VERIFY_DUMP"
SKILLTREE_SOURCE_DATABASE_URL="host=$tmp/socket port=55479 user=postgres dbname=postgres sslmode=disable" SKILLTREE_MIGRATION_OUTPUT="$tmp/restored.sqlite" "$repo/bin/skilltree-migrate" > "$tmp/report.json"
SKILLTREE_MIGRATION_OUTPUT="$tmp/restored.sqlite" "$repo/bin/skilltree-repair" > "$tmp/repair.json"
VERIFY_DIR="$tmp" python3 - <<'PY'
import json,os,sqlite3
from pathlib import Path
p=Path(os.environ['VERIFY_DIR']);r=json.loads((p/'report.json').read_text());fix=json.loads((p/'repair.json').read_text());assert r['integrity']=='ok' and all(t['verified'] for t in r['tables']);assert all(v==0 for v in r['duplicates'].values());assert all(v==0 for k,v in r['orphans'].items() if k!='members.user_id');c=sqlite3.connect('file:'+str(p/'restored.sqlite')+'?mode=ro',uri=True);assert c.execute('pragma integrity_check').fetchone()[0]=='ok';print('OFFHOST_FINAL_DUMP_RESTORED tables=%d rows=%d archived_orphans=%d schema=%d'%(len(r['tables']),sum(t['rows'] for t in r['tables']),fix['archivedMemberships'],fix['schemaVersion']));c.close()
PY
