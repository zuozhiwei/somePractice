#!/usr/bin/env bash
# Production cutover. Old services remain recoverable until Go starts serving traffic.
set -Eeuo pipefail
umask 077
printf 'SESSION_RESULT_BEGIN\n'
base="$HOME/.local/share/skilltree-go"; stage=$(cat "$base/current-stage"); old="$HOME/桌面/app/deploy"
case "$stage" in "$base"/stage.*) ;; *) echo 'INVALID_STAGE'; printf 'SESSION_RESULT_END\n'; exit 1;; esac
exec 9>"$stage/cutover.lock"; flock -n 9 || { echo 'CUTOVER_ALREADY_RUNNING';printf 'SESSION_RESULT_END\n';exit 1; }
suffix=$$; backup=$(mktemp -d "$stage/backups/final.XXXXXX"); prod=$(mktemp -d "$base/production.XXXXXX")
snapshot="skilltree_final_snapshot_$suffix"; restore="skilltree_final_restore_$suffix"; candidate="skilltree_final_candidate_$suffix"; rawvol="skilltree_final_volume_$suffix"
snapshot_created=0;restore_created=0;candidate_created=0;volume_created=0;final_created=0;phase=prelive
printf '%s\n' "$backup" > "$stage/final-backup-dir";printf '%s\n' "$prod" > "$stage/production-dir"
cat > "$stage/rollback.sh" <<'ROLLBACK'
#!/bin/sh
stage=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
if [ -f "$stage/live.marker" ]; then
  running=$(docker inspect --format '{{.State.Running}}' skill_tree_go 2>/dev/null || echo false)
  [ "$running" = true ] || docker start skill_tree_go >/dev/null 2>&1 || true
  exit 0
fi
docker start skill_tree_db >/dev/null 2>&1 || true
for i in $(seq 1 90); do docker exec skill_tree_db sh -c 'pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null 2>&1 && break; sleep .5; done
docker start skill_tree_java >/dev/null 2>&1 || true
docker start skill_tree_nginx >/dev/null 2>&1 || true
ROLLBACK
chmod 700 "$stage/rollback.sh"
on_exit(){
 rc=$?; trap - EXIT HUP INT TERM
 [ "$candidate_created" = 0 ] || docker rm -f "$candidate" >/dev/null 2>&1 || true
 [ "$restore_created" = 0 ] || docker rm -vf "$restore" >/dev/null 2>&1 || true
 [ "$snapshot_created" = 0 ] || docker rm -f "$snapshot" >/dev/null 2>&1 || true
 [ "$volume_created" = 0 ] || docker volume rm "$rawvol" >/dev/null 2>&1 || true
 if [ "$rc" -ne 0 ];then
  if [ "$phase" = prelive ];then
   [ "$final_created" = 0 ] || docker rm -f skill_tree_go >/dev/null 2>&1 || true
   sh "$stage/rollback.sh"; printf 'ROLLBACK_ATTEMPTED=old_services\n'
  else
   printf 'GO_LIVE_FAILED_DATA_PRESERVED=manual_recovery_required\n'
  fi
 fi
 printf 'CUTOVER_EXIT=%s\n' "$rc";printf 'SESSION_RESULT_END\n';exit "$rc"
}
trap on_exit EXIT
trap 'exit 130' HUP INT TERM
# Watchdog waits for the cutover lock. If the session dies before traffic is switched,
# it restores the old containers. After the live marker it only starts Go, never old DB.
nohup sh -c 'sleep 240; flock -x "$1" sh "$2"' _ "$stage/cutover.lock" "$stage/rollback.sh" 9>&- </dev/null >"$stage/watchdog.log" 2>&1 &
[ ! -f "$stage/live.marker" ] || { echo 'ALREADY_SWITCHED';exit 1; }
[ "$(docker inspect --format '{{.State.Running}}' skill_tree_java)" = true ]
[ "$(docker inspect --format '{{.State.Running}}' skill_tree_db)" = true ]
[ "$(docker inspect --format '{{.State.Running}}' skill_tree_nginx)" = true ]
if docker inspect skill_tree_go >/dev/null 2>&1;then echo 'GO_CONTAINER_NAME_EXISTS';exit 1;fi
tag=$(cat "$stage/image-tag");docker image inspect "$tag" >/dev/null
pgimage=$(docker inspect --format '{{.Image}}' skill_tree_db)
for f in docker-compose.yml .env java/app.jar java/application.yml java/application-docker.yml java/logback-spring.xml nginx/default.conf web/dist/index.html;do [ -f "$old/$f" ] || { echo "LEGACY_FILE_MISSING=$f";exit 1; };done
tar -C "$old" -czf "$backup/legacy-application.tar.gz" docker-compose.yml .env java/app.jar java/application.yml java/application-docker.yml java/logback-spring.xml nginx web/dist
tar -tzf "$backup/legacy-application.tar.gz" >/dev/null
docker inspect skill_tree_java skill_tree_db skill_tree_nginx > "$backup/legacy-containers.json"
printf 'LEGACY_CONFIG_BACKUP=verified\n'
# Stop every writer by stopping the application and then PostgreSQL itself.
docker stop -t 45 skill_tree_java >/dev/null
docker stop -t 45 skill_tree_db >/dev/null
[ "$(docker inspect --format '{{.State.Running}}' skill_tree_db)" = false ]
code=$(docker inspect --format '{{.State.ExitCode}}' skill_tree_db);[ "$code" = 0 ] || { printf 'DATABASE_STOP_EXIT=%s\n' "$code";exit 1; }
printf 'PRODUCTION_WRITES_FROZEN=yes\n'
# Archive the stopped volume without mounting it writable.
docker run --rm --network none --volumes-from skill_tree_db:ro -v "$backup:/backup" --entrypoint sh "$pgimage" -c 'set -e; tar -C /var/lib/postgresql/data -czf /backup/postgres-data.tar.gz .; chown 1000:1000 /backup/postgres-data.tar.gz'
tar -tzf "$backup/postgres-data.tar.gz" >/dev/null
[ "$(docker inspect --format '{{.State.Running}}' skill_tree_db)" = false ]
# Restore that raw archive to a brand-new isolated volume.
if docker volume inspect "$rawvol" >/dev/null 2>&1;then echo 'TEMP_VOLUME_EXISTS';exit 1;fi
docker volume create "$rawvol" >/dev/null;volume_created=1
docker run --rm --network none -v "$rawvol:/var/lib/postgresql/data" -v "$backup:/backup:ro" --entrypoint sh "$pgimage" -c 'set -e; tar -C /var/lib/postgresql/data -xzf /backup/postgres-data.tar.gz; chown -R postgres:postgres /var/lib/postgresql/data'
docker run -d --name "$snapshot" --network none -e POSTGRES_USER=admin -e POSTGRES_DB=skill_tree -v "$rawvol:/var/lib/postgresql/data" -v "$stage/bin:/tools:ro" -v "$backup:/out" "$pgimage" >/dev/null;snapshot_created=1
ready=0;for i in $(seq 1 120);do if docker exec "$snapshot" pg_isready -U admin -d skill_tree >/dev/null 2>&1;then ready=1;break;fi;sleep .25;done;[ "$ready" = 1 ] || { echo 'RAW_RESTORE_FAILED';exit 1; };sleep 2
docker exec "$snapshot" sh -c 'exec pg_dump -Fc --no-owner --no-privileges -U "$POSTGRES_USER" "$POSTGRES_DB"' > "$backup/final.dump"
test -s "$backup/final.dump"
docker exec --user 1000:1000 -e 'SKILLTREE_SOURCE_DATABASE_URL=host=/var/run/postgresql port=5432 user=admin dbname=skill_tree sslmode=disable' -e SKILLTREE_MIGRATION_OUTPUT=/out/original.sqlite "$snapshot" /tools/skilltree-migrate > "$backup/original-migration.json" 2> "$backup/original-migration.err"
# Restore the final logical dump into another empty, isolated database.
mkdir -p "$prod/data";cp "$stage/service.env" "$prod/service.env";chmod 600 "$prod/service.env"
docker run -d --name "$restore" --network none -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_USER=restore -v "$stage/bin:/tools:ro" -v "$prod/data:/data" "$pgimage" >/dev/null;restore_created=1
ready=0;for i in $(seq 1 120);do if docker exec "$restore" pg_isready -U restore -d postgres >/dev/null 2>&1;then ready=1;break;fi;sleep .25;done;[ "$ready" = 1 ] || { echo 'LOGICAL_RESTORE_START_FAILED';exit 1; };sleep 2
docker exec -i "$restore" pg_restore -U restore -d postgres --exit-on-error --no-owner --no-privileges < "$backup/final.dump" > "$backup/final-restore.log" 2>&1
docker exec --user 1000:1000 -e 'SKILLTREE_SOURCE_DATABASE_URL=host=/var/run/postgresql port=5432 user=restore dbname=postgres sslmode=disable' -e SKILLTREE_MIGRATION_OUTPUT=/data/skilltree.db "$restore" /tools/skilltree-migrate > "$backup/final-migration.json" 2> "$backup/final-migration.err"
BACKUP_DIR="$backup" PROD_DIR="$prod" python3 - <<'PY'
import hashlib,json,os,pathlib,sqlite3
backup=pathlib.Path(os.environ['BACKUP_DIR']);prod=pathlib.Path(os.environ['PROD_DIR']);a=sqlite3.connect('file:'+str(backup/'original.sqlite')+'?mode=ro',uri=True);b=sqlite3.connect('file:'+str(prod/'data/skilltree.db')+'?mode=ro',uri=True)
r=json.loads((backup/'final-migration.json').read_text());assert r['integrity']=='ok' and all(t['verified'] for t in r['tables']);assert all(v==0 for v in r['duplicates'].values());assert all(v==0 for k,v in r['orphans'].items() if k!='members.user_id')
tables=[x[0] for x in a.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' and name!='schema_migrations' order by name")]
for table in tables:
 def hashes(c):return sorted(hashlib.sha256(json.dumps(row,ensure_ascii=False,default=str,separators=(',',':')).encode()).digest() for row in c.execute('select * from "'+table.replace('"','""')+'"'))
 if hashes(a)!=hashes(b):raise RuntimeError('restored data mismatch in '+table)
print('FINAL_RESTORE_TABLES=%d FINAL_RESTORE_ROWS=%d ROW_HASHES=matched'%(len(r['tables']),sum(x['rows'] for x in r['tables'])))
a.close();b.close()
PY
SKILLTREE_MIGRATION_OUTPUT="$prod/data/skilltree.db" "$stage/bin/skilltree-repair" > "$backup/final-repair.json"
SKILLTREE_DB="$prod/data/skilltree.db" SKILLTREE_BACKUP_OUTPUT="$backup/final-sqlite.db" "$stage/bin/skilltree-backup" > "$backup/sqlite-backup.log"
BACKUP_DIR="$backup" PROD_DIR="$prod" python3 - <<'PY'
import hashlib,json,os,pathlib,sqlite3
backup=pathlib.Path(os.environ['BACKUP_DIR']);prod=pathlib.Path(os.environ['PROD_DIR']);a=sqlite3.connect('file:'+str(prod/'data/skilltree.db')+'?mode=ro',uri=True);b=sqlite3.connect('file:'+str(backup/'final-sqlite.db')+'?mode=ro',uri=True)
assert a.execute('pragma integrity_check').fetchone()[0]=='ok' and b.execute('pragma integrity_check').fetchone()[0]=='ok'
for (name,) in a.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name"):
 def hashes(c):return sorted(hashlib.sha256(json.dumps(row,ensure_ascii=False,default=str,separators=(',',':')).encode()).digest() for row in c.execute('select * from "'+name.replace('"','""')+'"'))
 if hashes(a)!=hashes(b):raise RuntimeError('SQLite backup mismatch in '+name)
print('FINAL_SQLITE_BACKUP_ROW_HASHES=matched'); print('ARCHIVED_ORPHANS='+str(json.loads((backup/'final-repair.json').read_text())['archivedMemberships']))
a.close();b.close()
PY
(cd "$backup" && sha256sum legacy-application.tar.gz postgres-data.tar.gz final.dump final-sqlite.db > SHA256SUMS)
# Verify the final database in a restricted, loopback-only Go container.
docker run -d --name "$candidate" --network bridge --read-only --cap-drop ALL --security-opt no-new-privileges --pids-limit 128 --memory 512m --env-file "$prod/service.env" -v "$prod/data:/data" -p 127.0.0.1:18082:8080 "$tag" >/dev/null;candidate_created=1
ready=0;for i in $(seq 1 60);do if curl --noproxy '*' -fsS http://127.0.0.1:18082/healthz >/dev/null 2>&1;then ready=1;break;fi;sleep .25;done;[ "$ready" = 1 ] || { echo 'FINAL_CANDIDATE_HEALTH_FAILED';exit 1; }
PROD_DIR="$prod" python3 - <<'PY'
import base64,hashlib,hmac,json,os,pathlib,sqlite3,time,urllib.request
p=pathlib.Path(os.environ['PROD_DIR']);env=dict(line.split('=',1) for line in (p/'service.env').read_text().splitlines() if '=' in line);db=sqlite3.connect('file:'+str(p/'data/skilltree.db')+'?mode=ro',uri=True);uid,username,role=db.execute("SELECT id,username,role FROM users WHERE role='EXTERNAL' ORDER BY id LIMIT 1").fetchone();total=db.execute('select count(*) from maps').fetchone()[0];db.close();key=base64.b64decode(env['SKILLTREE_JWT_SECRET']);alg='HS512' if len(key)>=64 else ('HS384' if len(key)>=48 else 'HS256');digest={'HS256':hashlib.sha256,'HS384':hashlib.sha384,'HS512':hashlib.sha512}[alg];b=lambda x:base64.urlsafe_b64encode(x).decode().rstrip('=');now=int(time.time());raw=(b(json.dumps({'alg':alg,'typ':'JWT'}).encode())+'.'+b(json.dumps({'sub':username,'userId':uid,'role':role,'iat':now,'exp':now+120}).encode())).encode();token=raw.decode()+'.'+b(hmac.new(key,raw,digest).digest());op=urllib.request.build_opener(urllib.request.ProxyHandler({}));request=urllib.request.Request('http://127.0.0.1:18082/api/maps/progress',headers={'Authorization':'Bearer '+token});data=json.load(op.open(request,timeout=10));assert data['code']==200 and data['data']['totalMaps']==total
with op.open('http://127.0.0.1:18082/',timeout=10) as response:assert '<div id="app">' in response.read().decode()
print('FINAL_CANDIDATE_API_AND_FRONTEND=passed')
PY
docker rm -f "$candidate" >/dev/null;candidate_created=0
# Persist a reproducible compose definition. This does not start the production container yet.
cat > "$prod/compose.yml" <<COMPOSE
services:
  app:
    image: $tag
    container_name: skill_tree_go
    restart: unless-stopped
    read_only: true
    cap_drop: [ALL]
    security_opt: [no-new-privileges:true]
    pids_limit: 128
    mem_limit: 512m
    env_file: [./service.env]
    volumes: [./data:/data]
    ports: ["80:8080", "8080:8080"]
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
COMPOSE
(cd "$prod" && docker compose config --quiet)
docker create --name skill_tree_go --restart unless-stopped --network bridge --read-only --cap-drop ALL --security-opt no-new-privileges --pids-limit 128 --memory 512m --env-file "$prod/service.env" -v "$prod/data:/data" -p 80:8080 -p 8080:8080 --log-opt max-size=10m --log-opt max-file=3 "$tag" >/dev/null;final_created=1
# Point of no return: after the new service can accept writes, do not restart old Java automatically.
docker stop -t 30 skill_tree_nginx >/dev/null
touch "$stage/live.marker";phase=live
if ! docker start skill_tree_go >/dev/null;then echo 'GO_START_FAILED';exit 1;fi
ready=0;for i in $(seq 1 80);do if curl --noproxy '*' -fsS http://127.0.0.1/healthz >/dev/null 2>&1 && curl --noproxy '*' -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1;then ready=1;break;fi;sleep .25;done;[ "$ready" = 1 ] || { echo 'GO_HEALTH_FAILED';docker logs --tail 10 skill_tree_go 2>&1;exit 1; }
curl --noproxy '*' -fsS http://127.0.0.1/ | grep -q '<div id="app">'
ln -sfn "$prod" "$base/current"
mkdir -p "$base/bin" "$base/backups";chmod 700 "$base/bin" "$base/backups";cp "$stage/bin/skilltree-backup" "$base/bin/skilltree-backup";chmod 700 "$base/bin/skilltree-backup"
cat > "$base/backup.sh" <<'BACKUP'
#!/bin/sh
set -eu
umask 077
base="$HOME/.local/share/skilltree-go"; current=$(readlink -f "$base/current")
export SKILLTREE_DB="$current/data/skilltree.db"
export SKILLTREE_BACKUP_OUTPUT="$base/backups/skilltree-$(date -u +%Y%m%dT%H%M%SZ)-$$.db"
exec "$base/bin/skilltree-backup"
BACKUP
chmod 700 "$base/backup.sh"
"$base/backup.sh" > "$backup/first-live-backup.log"
(crontab -l 2>/dev/null || true) > "$backup/previous-crontab.txt"
if ! grep -Fq '# skilltree-go daily backup' "$backup/previous-crontab.txt";then
 { cat "$backup/previous-crontab.txt"; printf '15 2 * * * %s/backup.sh >> %s/backups/cron.log 2>&1 # skilltree-go daily backup\n' "$base" "$base"; } | crontab -
fi
printf 'GO_SERVICE=running\nPRODUCTION_URL=http://192.168.21.128/\n'
printf 'PRODUCTION_DIR=%s\nFINAL_BACKUP_DIR=%s\n' "$prod" "$backup"
printf 'GO_RESTART_POLICY=';docker inspect --format '{{.HostConfig.RestartPolicy.Name}}' skill_tree_go
printf 'GO_MEMORY=';docker stats --no-stream --format '{{.MemUsage}}' skill_tree_go
printf 'LEGACY_STATUS\n';for n in skill_tree_java skill_tree_db skill_tree_nginx skill_tree_filebrowser skill_tree_minio;do docker inspect --format '{{.Name}} {{.State.Status}}' "$n";done
printf 'DAILY_BACKUP=installed\n'
