#!/bin/sh
# Run on the production server after verifying the Qiniu HTTPS certificate.
set -eu
umask 077
stamp=$(date +%Y%m%dT%H%M%S)
backup="/var/lib/skilltree/backups/before-qiniu-https-$stamp.db"
envbackup="/root/skilltree-deploy/service-env-before-qiniu-https-$stamp"
install -m 0600 /etc/skilltree.env "$envbackup"
systemctl stop skilltree
SKILLTREE_DB=/var/lib/skilltree/skilltree.db SKILLTREE_BACKUP_OUTPUT="$backup" /opt/skilltree/skilltree-backup
python3 - "$backup" <<'PY'
import sqlite3, sys
from pathlib import Path
cfg=Path('/etc/skilltree.env')
lines=cfg.read_text().splitlines()
old='http://files.s.somecoder.cn'
new='https://files.s.somecoder.cn'
assert any(x == 'QINIU_DOMAIN='+old for x in lines)
db=sqlite3.connect('/var/lib/skilltree/skilltree.db')
check=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True)
assert check.execute('pragma integrity_check').fetchone()[0]=='ok'
assert list(check.iterdump()) == list(db.iterdump())
with db:
    result=db.execute('update user_task_submissions set submission_data=replace(submission_data, ?, ?) where instr(submission_data, ?) > 0', (old+'/',new+'/',old+'/'))
    print('historical_records_updated',result.rowcount)
    assert db.execute('pragma integrity_check').fetchone()[0]=='ok'
    assert db.execute('select count(*) from user_task_submissions where instr(submission_data,?) > 0',(old+'/',)).fetchone()[0]==0
cfg.write_text('\n'.join(('QINIU_DOMAIN='+new if x.startswith('QINIU_DOMAIN=') else x) for x in lines)+'\n')
cfg.chmod(0o600)
print('backup_verified',sys.argv[1])
PY
systemctl start skilltree
for n in 1 2 3 4 5 6 7 8 9 10; do
    if curl -fsS http://127.0.0.1:8081/healthz; then break; fi
    sleep 1
done
systemctl is-active --quiet skilltree
echo QINIU_HTTPS_DEPLOYED
