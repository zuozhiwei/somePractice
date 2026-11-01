#!/bin/sh
# Isolated Linux user-systemd acceptance test; never connects to production.
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/skilltree-smoke.XXXXXX")
chmod 700 "$work"
unit="skilltree-smoke-$$"
port="${SKILLTREE_SMOKE_PORT:-18763}"
key=YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYQ==
cleanup(){ systemctl --user stop "$unit.service" >/dev/null 2>&1 || true; systemctl --user reset-failed "$unit.service" >/dev/null 2>&1 || true; rm -f -- "$work/skilltree.db" "$work/skilltree.db-wal" "$work/skilltree.db-shm" "$work/backup.db"; rmdir -- "$work" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
systemd-run --user --quiet --unit="$unit" --property=NoNewPrivileges=yes --setenv="SKILLTREE_DB=$work/skilltree.db" --setenv="SKILLTREE_LISTEN=127.0.0.1:$port" --setenv="SKILLTREE_JWT_SECRET=$key" "$repo/bin/skilltree"
i=0
until curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null; do i=$((i+1)); if [ "$i" -ge 50 ]; then systemctl --user status "$unit.service" --no-pager >&2 || true; exit 1; fi; sleep .2; done
SMOKE_DB="$work/skilltree.db" SMOKE_PORT="$port" python3 - <<'PY'
import concurrent.futures,json,os,sqlite3,time,urllib.request
base='http://127.0.0.1:'+os.environ['SMOKE_PORT'];op=urllib.request.build_opener(urllib.request.ProxyHandler({}))
def request(method,path,data=None,token=None):
 body=json.dumps(data).encode() if data is not None else None
 req=urllib.request.Request(base+path,data=body,method=method,headers={'Content-Type':'application/json',**({'Authorization':'Bearer '+token} if token else {})})
 with op.open(req,timeout=10) as resp:return json.load(resp)
teacher=request('POST','/api/auth/register',{'username':'teacher-test','password':'test-password','role':'EXTERNAL'})['data'];student=request('POST','/api/auth/register',{'username':'student-test','password':'test-password','role':'EXTERNAL'})['data']
c=sqlite3.connect(os.environ['SMOKE_DB']);c.execute("UPDATE users SET role='TEACHER',level=10 WHERE id=?",(teacher['user']['id'],));c.execute("INSERT INTO maps(id,name,is_builtin,order_index) VALUES(1,'Test map',1,1)");c.execute("INSERT INTO levels(id,map_id,name,is_builtin,order_index,tier,xp_reward,prerequisite_level_ids) VALUES(1,1,'Test level',1,1,1,10,'[]')");c.execute("INSERT INTO tasks(id,level_id,title,is_builtin,ai_review_enabled,xp_reward,sort_order) VALUES(1,1,'Test task',1,0,5,1)");c.commit();c.close()
t=teacher['token'];s=student['token']; assert request('GET','/api/maps',token=s)['data'][0]['name']=='Test map'; assert request('GET','/api/levels/1',token=s)['data']['tasks'][0]['title']=='Test task'
sub=request('POST','/api/tasks/1/submit',{'submissionData':'test work'},s)['data'];request('PUT',f"/api/submissions/{sub['id']}/review",{'status':'APPROVED','reviewComment':'ok'},t)
progress=request('GET','/api/maps/progress',token=s)['data'];assert progress['completedLevels']==1 and progress['totalXp']==15,progress
def load(_):
 start=time.monotonic();request('GET','/api/maps/progress',token=s);return time.monotonic()-start
with concurrent.futures.ThreadPoolExecutor(max_workers=30) as pool: durations=list(pool.map(load,range(30)))
print('30 concurrent progress requests: max %.3fs' % max(durations))
print('HTTP registration, login, learning, review and XP: passed')
PY
SKILLTREE_DB="$work/skilltree.db" SKILLTREE_BACKUP_OUTPUT="$work/backup.db" "$repo/bin/skilltree-backup"
systemctl --user restart "$unit.service"
i=0
until curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null; do i=$((i+1)); [ "$i" -lt 50 ] || exit 1; sleep .2; done
SMOKE_DB="$work/skilltree.db" SMOKE_BACKUP="$work/backup.db" python3 - <<'PY'
import os,sqlite3
for env in ['SMOKE_DB','SMOKE_BACKUP']:
 c=sqlite3.connect(os.environ[env]);assert c.execute('pragma integrity_check').fetchone()[0]=='ok';assert c.execute("select total_xp from users where username='student-test'").fetchone()[0]==15;c.close()
print('systemd restart and backup restore data: passed')
PY
printf 'isolated smoke-test data will be removed\n'
