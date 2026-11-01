#!/bin/sh
# Load test a COPY of a migrated database, never the live database.
set -eu
[ "$#" -eq 1 ] || { echo 'usage: systemd-load.sh /path/to/verified-offline-copy.db' >&2; exit 2; }
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/skilltree-load.XXXXXX"); chmod 700 "$work"
cp -- "$1" "$work/data.db"; chmod 600 "$work/data.db"
unit="skilltree-load-$$";port="${SKILLTREE_LOAD_PORT:-18764}"
key=YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYQ==
cleanup(){ systemctl --user stop "$unit.service" >/dev/null 2>&1 || true; systemctl --user reset-failed "$unit.service" >/dev/null 2>&1 || true; rm -f -- "$work/data.db" "$work/data.db-wal" "$work/data.db-shm"; rmdir -- "$work" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
systemd-run --user --quiet --unit="$unit" --property=NoNewPrivileges=yes --setenv="SKILLTREE_DB=$work/data.db" --setenv="SKILLTREE_LISTEN=127.0.0.1:$port" --setenv="SKILLTREE_JWT_SECRET=$key" "$repo/bin/skilltree"
i=0;until curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null 2>&1;do i=$((i+1));if [ "$i" -ge 50 ];then systemctl --user status "$unit.service" --no-pager >&2 || true;exit 1;fi;sleep .2;done
SMOKE_DB="$work/data.db" SMOKE_PORT="$port" SMOKE_KEY="$key" python3 - <<'PY'
import base64,concurrent.futures,hashlib,hmac,json,os,sqlite3,time,urllib.request
c=sqlite3.connect(os.environ['SMOKE_DB']);users=c.execute('select id,username,role from users order by id limit 30').fetchall();c.close();assert users
b64=lambda b:base64.urlsafe_b64encode(b).decode().rstrip('=');now=int(time.time());tokens=[]
for uid,name,role in users:
 raw=(b64(b'{"alg":"HS512","typ":"JWT"}')+'.'+b64(json.dumps({'sub':name,'userId':uid,'role':role,'iat':now,'exp':now+300}).encode())).encode();tokens.append(raw.decode()+'.'+b64(hmac.new(base64.b64decode(os.environ['SMOKE_KEY']),raw,hashlib.sha512).digest()))
base='http://127.0.0.1:'+os.environ['SMOKE_PORT'];op=urllib.request.build_opener(urllib.request.ProxyHandler({}));paths=['/api/maps/progress','/api/levels/tree','/api/adventure/mission']
def request(i):
 start=time.monotonic();req=urllib.request.Request(base+paths[i%len(paths)],headers={'Authorization':'Bearer '+tokens[i%len(tokens)]})
 with op.open(req,timeout=30) as response:
  assert json.load(response)['code']==200
 return time.monotonic()-start
with concurrent.futures.ThreadPoolExecutor(max_workers=30) as pool: durations=list(pool.map(request,range(90)))
durations.sort();print('90 requests, 30 workers: median %.3fs, p95 %.3fs, max %.3fs' %(durations[45],durations[85],durations[-1]))
PY
pid=$(systemctl --user show "$unit.service" -p MainPID --value)
ps -p "$pid" -o rss=,vsz= | awk '{printf "service RSS: %.1f MiB; virtual: %.1f MiB\n", $1/1024, $2/1024}'
printf 'isolated load-test data will be removed\n'
