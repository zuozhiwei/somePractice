# 生产切换与运维记录（2026-11-01）

## 当前运行状态

- 服务地址：`http://192.168.21.128/`，兼容端口 `http://192.168.21.128:8080/`。
- 当前容器：`skill_tree_go`，镜像 `skilltree-go:stage-20261031235131-secure`，重启策略 `unless-stopped`；容器非 root、根文件系统只读，SQLite 数据单独持久化。
- 服务器当前账户具有 Docker 管理权限，但非交互 sudo 不可用，用户级 systemd 未开启 linger。因此本次采用已有 Docker 守护进程运行单个 Go 服务，而非安装系统级 systemd 单元。Docker 服务已启用开机启动。
- 活跃数据与配置：`/home/user/.local/share/skilltree-go/production.ndODkD/`，其 `data/skilltree.db` 是当前生产数据库；`service.env` 包含密钥，应保持私有。
- `skill_tree_java`、`skill_tree_db`、`skill_tree_nginx` 已停止但保留。原 PostgreSQL 的 5432 端口已关闭。`skill_tree_filebrowser` 和 `skill_tree_minio` 是既有独立服务，本次未停用。
- 迁移时两条指向已删除用户的聊天成员记录保存在生产 SQLite 表 `migration_orphan_members` 中，不参与活跃会话。

## 验收

生产环境停写后取得 PostgreSQL 数据卷快照，恢复到隔离实例，再生成 PostgreSQL 自定义格式备份并恢复到第二个隔离实例；两份迁移数据的 17 张表、1806 行逐行摘要一致。修复历史孤立成员后，SQLite 快照再次备份，逐行摘要一致。旧应用配置、镜像容器信息、PostgreSQL 数据卷、最终 PostgreSQL dump 和最终 SQLite 快照均保存在服务器备份目录。

局域网验证：首页、登录页、80 和 8080 健康检查、用户资料、地图进度、关卡树、冒险任务及 WebSocket 握手通过。生产容器已实际重启并恢复健康。目标机上预检 30 并发、90 次学习读取请求全部成功，中位数 0.054 秒、P95 0.201 秒、最大 0.230 秒；这仅代表当次测试。

最终 PostgreSQL dump 于服务器时间 2026-11-01 07:56:01 创建，SHA-256：`bc7cd4e1f411c31b70e9506073ae1f9102f6b16b06a57679c4ad03dab8608d98`。该文件已通过加密传输保存到本机 `data/production-final-20261101-075749.dump`，并在另一套隔离 PostgreSQL 实例中独立恢复、迁移和校验通过。本机文件含生产数据，不应提交或分享。

服务器备份目录：`/home/user/.local/share/skilltree-go/stage.pM9W43/backups/final.H1929a/`。该目录及配置备份包含敏感数据，应保持用户私有权限并另行做好离机备份。

## 日常操作

以服务器 `user` 账户执行：

```sh
~/.local/share/skilltree-go/manage.sh status
~/.local/share/skilltree-go/manage.sh logs 100
~/.local/share/skilltree-go/manage.sh backup
~/.local/share/skilltree-go/manage.sh restart
```

每天服务器时间 02:15，用户 crontab 执行 SQLite 在线备份，输出至 `/home/user/.local/share/skilltree-go/backups/`。备份执行 SQLite 完整性检查；首份备份已验证。备份不会自动删除，应监控剩余空间并定期复制到另一台机器，安排恢复演练。

当前容器通过 `docker create` 创建，`production.ndODkD/compose.yml` 仅作未来受控重建的配置参考，不能对现有同名容器直接运行 `docker compose up`。更改 `service.env` 后，单纯重启容器不会重新读取环境文件；应先备份，并在维护窗口受控重建容器。

七牛配置已继承，上传凭证生成验证通过，但没有对真实空间进行测试上传。DeepSeek 和 SMTP 在旧生产配置中均未配置，Go 中相应功能在配置前不可用。

**回退注意：** Go 接管后可能产生新数据。旧 PostgreSQL 只包含切换时的快照，不能直接启动旧 Java 接收写入，否则 Go 之后的数据会丢失。回退需先停止 Go 写入，备份当前 SQLite，并制定、验证反向数据同步方案。
