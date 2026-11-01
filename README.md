# 智阶学习系统（Go + SQLite）

当前生产环境已于 2026-11-01 切换到 Go + SQLite；实际运行方式、备份及运维注意事项见 `docs/production-2026-11-01.md`。

这是系统后续维护的主项目。Vue 前端源码位于 `web/`，构建后嵌入 Go 程序；服务使用本地 SQLite，生产环境推荐单机 systemd。旧 Java 项目保留为迁移数据源，正式切换后不要同时向两套系统写入。

## 当前能力

账号、邀请码、邮件重置、地图与任务、作业及人工/AI 审核、经验与关卡、师徒与团队、排行榜、聊天、七牛附件和管理后台均由 Go 服务提供。聊天通过 HTTP 持久化和分页补拉，WebSocket 负责通知；作业附件由浏览器直传七牛，用户也可以提交自己的外部分享链接。Markdown 在前端渲染时经过净化。

系统支持迁移现有生产 PostgreSQL：逐表逐行核对数据，并对业务关联和重复键进行检查。已在数据副本中发现两条指向已删除用户的聊天成员记录；修复工具会先将它们完整归档到 SQLite 的 `migration_orphan_members`，再移出活跃成员，并校验和启用约束。保留原 PostgreSQL 备份及未修复的迁移副本。

## 构建

使用项目 `go.mod` 指定的 Go 版本和 Node.js。Windows 上运行 `./build.ps1`，会安装前端依赖、构建页面并生成 `bin/` 下的 Linux amd64 可执行文件。Linux 上执行 `sh build.sh`，默认按当前 Linux 架构构建；部署前用 `uname -m` 确认服务器架构。产物包括 `skilltree`、`skilltree-migrate`、`skilltree-repair` 和 `skilltree-backup`。前端改动后必须重新构建，才能更新嵌入的页面。

## 数据迁移

1. 先对 PostgreSQL 做完整备份并演练恢复。在受保护的终端设置 `SKILLTREE_SOURCE_DATABASE_URL` 为 PostgreSQL 连接串，并将 `SKILLTREE_MIGRATION_OUTPUT` 设置为**不存在**的 SQLite 文件路径；运行 `skilltree-migrate`。工具使用只读、可重复读事务，遇到结构差异、导入失败或内容差异会失败并删除本次创建的目标文件，不会覆盖已有文件。
2. 查看 JSON 报告：所有表 `verified` 应为 `true`、`integrity` 为 `ok`；记录 `orphans` 和 `duplicates`。有异常时 `ready_for_cutover` 为 `false`，不能跳过检查。
3. 对该**离线副本**设置相同 `SKILLTREE_MIGRATION_OUTPUT`，运行 `skilltree-repair`。它仅自动归档指向不存在用户的聊天成员；其他异常仍会中止。成功后启用关系触发器和关键唯一索引。关系触发器覆盖账户、地图、任务、进度、提交、师徒和聊天等引用；历史 `created_by` 允许系统保留值，因此不视为普通用户外键。它们是触发器约束，并非 SQLite 原生 `FOREIGN KEY` 声明。
4. 演练通过不代表最终数据已同步。正式切换时，先停止旧系统写入，重新导出并执行上述检查和修复，再部署最终副本。若新系统开始接受写入，直接切回旧库会丢失新数据；须先停写并处理反向同步。

迁移文件和备份含真实用户信息，应限制访问、不要提交到仓库。`data/` 和 `.env` 已列入 `.gitignore`。

## Debian / Ubuntu 部署

1. 将 `bin/` 和 `deploy/` 一起复制到服务器；若从 Windows 生成的 ZIP 解压，先运行 `chmod 0755 bin/*`。准备 `/var/lib/skilltree/skilltree.db`，内容为上述检查通过的最终 SQLite 文件；数据库必须放在本机持久磁盘上。
2. 将 `deploy/skilltree.env.example` 复制为 `/etc/skilltree.env` 并填写：
   - `SKILLTREE_JWT_SECRET`：Base64 编码、解码后至少 32 字节。继续使用旧登录态及旧的加密邀请链接时，应通过安全渠道使用旧系统相同密钥；明文师父 ID 的旧邀请链接出于权限保护已停止支持，需重新生成。
   - `SKILLTREE_TIMEZONE=UTC`：本次旧生产环境 `LocalDateTime` 使用 UTC，迁移保留原值；如果实际源系统时区不同，应在部署前更正。
   - `QINIU_ACCESS_KEY`、`QINIU_SECRET_KEY`、`QINIU_BUCKET`、`QINIU_DOMAIN` 和上传区域对应的 `QINIU_UPLOAD_HOST`：供上传及私有文件签名使用。
   - `DEEPSEEK_API_KEY`、可选 `DEEPSEEK_MODEL`：供 AI 审核使用；`SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`：供密码重置使用。缺失时对应功能返回明确错误，其他功能仍可使用。SMTP 要求 STARTTLS。
3. 以 root 运行 `sh deploy/install.sh`。脚本创建专用服务用户、安装文件、启动 `skilltree.service` 和每日备份 timer；不会自动从旧系统迁移数据库。
4. `systemctl status skilltree` 查看服务，`journalctl -u skilltree -f` 查看日志；用 `curl http://127.0.0.1:8081/healthz` 检查健康状态。默认只监听本机；局域网直连可设置 `SKILLTREE_LISTEN=0.0.0.0:8081` 并配置防火墙。公网访问应配置域名及 HTTPS 反向代理，代理需支持 WebSocket；不要把明文 HTTP 暴露到公网。

## 备份与恢复

定时器每天把在线一致性快照写入 `/var/lib/skilltree/backups/`，生成后执行 SQLite 完整性检查。备份不会自动删除；应监控 40 GB 磁盘剩余空间并定期复制到另一台机器。也可设置 `SKILLTREE_DB`、`SKILLTREE_BACKUP_OUTPUT` 后运行 `skilltree-backup`，目标必须不存在。

恢复演练：停止服务，将备份复制到一个新路径，先做完整性和业务数据检查，再替换服务使用的数据库并启动。**不要在运行中只复制主数据库文件**；WAL 中可能有尚未合并的数据。恢复操作及其数据库替换应在服务停止后进行。

## 验证

```text
go test ./...
go vet ./...
```

`tests/systemd-smoke.sh` 使用临时合成数据在 Linux 用户级 systemd 中验证注册、学习、审核、重启和备份。`tests/systemd-load.sh` 接收一个离线数据库副本，复制后执行 30 并发的读取测试；不要传入运行中的数据库文件。七牛和 AI 已有本地模拟接口测试，邮件有本地 STARTTLS 模拟测试；正式上线前仍需使用实际服务配置验证连通性和权限。
