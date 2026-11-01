#!/bin/sh
set -eu
if [ "$(id -u)" -ne 0 ]; then echo '请使用 root 执行' >&2; exit 1; fi
cd "$(dirname "$0")/.."
[ -f bin/skilltree ] || { echo '请先运行 build.sh 构建' >&2; exit 1; }
[ -f /var/lib/skilltree/skilltree.db ] || { echo '请先将已验证并修复的 SQLite 数据库放入 /var/lib/skilltree/skilltree.db' >&2; exit 1; }
[ -f /etc/skilltree.env ] || { echo '请先配置 /etc/skilltree.env（参考 deploy/skilltree.env.example）' >&2; exit 1; }
if ! id skilltree >/dev/null 2>&1; then useradd --system --home-dir /var/lib/skilltree --shell /usr/sbin/nologin skilltree; fi
install -d -m 0755 /opt/skilltree
install -d -o skilltree -g skilltree -m 0700 /var/lib/skilltree /var/lib/skilltree/backups
chown skilltree:skilltree /var/lib/skilltree/skilltree.db
for file in skilltree skilltree-migrate skilltree-repair skilltree-backup; do install -m 0755 "bin/$file" "/opt/skilltree/$file"; done
install -m 0755 deploy/backup.sh /opt/skilltree/backup.sh
install -m 0644 deploy/skilltree.service /etc/systemd/system/skilltree.service
install -m 0644 deploy/skilltree-backup.service /etc/systemd/system/skilltree-backup.service
install -m 0644 deploy/skilltree-backup.timer /etc/systemd/system/skilltree-backup.timer
chmod 0600 /etc/skilltree.env
systemctl daemon-reload
systemctl enable skilltree.service skilltree-backup.timer
systemctl restart skilltree.service
systemctl start skilltree-backup.timer
