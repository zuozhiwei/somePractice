# 公网服务器部署记录（2026-11-01）

正式地址：`https://s.somecoder.cn/`，域名解析到 `82.156.151.237`，IP 地址 HTTPS 和普通 HTTP 均跳转到该域名。Debian 13 上使用系统级 systemd 运行 Go 服务，监听 `127.0.0.1:8081`，由 Nginx 代理 HTTPS 和 WebSocket；HTTP 端口仅用于重定向及 ACME 验证。服务、Nginx、数据库备份定时器和证书续期定时器均已启用开机启动。

切换时先停止原局域网 Go 容器写入，再用 SQLite 在线备份工具导出最终快照，通过 HTTPS 加密传输到新服务器并验证 SHA-256。快照完整性检查通过，含 36 个用户；新服务器备份文件通过完整性检查，表内容与最终快照一致。本机离线副本位于 `data/cloud-cutover-20261101.db`，该文件包含用户数据，不应提交或分享。原局域网容器保留但已停止，不能直接恢复其写入，否则将产生数据分叉。

应用配置为 `/etc/skilltree.env`（仅 root 可读），当前数据库为 `/var/lib/skilltree/skilltree.db`。每日备份存放于 `/var/lib/skilltree/backups/`，不自动清理；定期检查磁盘空间、将备份复制到其他机器并演练恢复。`/root/skilltree-deploy/` 包含切换快照、旧配置和备份，应保持仅 root 可访问。

```sh
systemctl status skilltree nginx
journalctl -u skilltree -f
systemctl start skilltree-backup.service
systemctl list-timers skilltree-backup.timer skilltree-cert-renew.timer
```

Let's Encrypt 为域名及公网 IP 分别签发证书，公网 IP 证书有效期较短。续期由 `skilltree-cert-renew.timer` 每四小时检查，部署钩子在证书更新后重新加载 Nginx。保持域名解析指向本机，并保持公网 80 端口可访问以供验证；定期检查定时器和续期日志。

2026-11-10 七牛附件域名更新为 HTTP；更新前数据库备份在 `/var/lib/skilltree/backups/before-qiniu-domain-20261110T221242.db`。当时 HTTPS 证书尚未生效。

2026-11-11 七牛域名证书已验证，应用和历史作业链接均已切换到 `https://files.s.somecoder.cn`。切换前备份为 `/var/lib/skilltree/backups/before-qiniu-https-20261111T143613.db`；上传令牌和历史附件 HTTPS 签名读取均已验证。
