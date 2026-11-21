# 课后工坊（Vue + Go + SQLite）

当前生产环境已于 2026-11-01 迁至公网服务器；实际运行方式、备份及运维注意事项见 `docs/cloud-production-2026-11-01.md`。此前局域网切换记录见 `docs/production-2026-11-01.md`。

课后工坊是面向大学生的项目练习空间。访客可在首页查看项目路线。登录后从项目世界地图选择关卡。成果册可回看通过的任务，完成全章后点亮章节印章。项目路线、章节目录和技能树也可单独查看。

Vue 前端源码位于 `web/`，构建后嵌入 Go 程序。平台服务使用本地 SQLite，生产环境推荐单机 systemd。旧 Java 项目保留为迁移数据源，正式切换后不要同时向两套系统写入。

## 当前能力

账号、邀请码、邮件重置、地图与任务、作业及人工/AI 审核、经验与关卡、师徒与团队、排行榜、聊天、七牛附件和管理后台均由 Go 服务提供。聊天通过 HTTP 持久化和分页补拉，WebSocket 负责通知；作业附件由浏览器直传七牛，用户也可以提交自己的外部分享链接。Markdown 在前端渲染时经过净化。

教师负责有权审核的作业和日常师徒协作。超级管理员（`SUPERADMIN`）负责系统资料，包括用户、关卡、任务、教程和邀请码。学生在「找回密码」输入登录用户名或绑定邮箱，可向唯一绑定的邮箱申请 5 分钟有效的验证码。没有绑定邮箱或收不到邮件时，学生可向教师或管理员提供用户名。超级管理员在「用户管理 → 重置密码」核实本人身份，生成 15 分钟有效的一次性恢复码，单独交给本人；学生选择「管理员恢复码」并自行设置新密码。每个验证码最多可试 5 次，同一方式重新签发后旧码失效。只有成功设置新密码后才会撤销旧登录会话。超级管理员账号由服务器运维人员为确认过的账号设置，后台会保护这些账号的角色和资料。启用新版本前应确认生产库至少有一位可登录的超级管理员，防止管理入口无人可用。

系统支持迁移现有生产 PostgreSQL：逐表逐行核对数据，并对业务关联和重复键进行检查。已在数据副本中发现两条指向已删除用户的聊天成员记录；修复工具会先将它们完整归档到 SQLite 的 `migration_orphan_members`，再移出活跃成员，并校验和启用约束。保留原 PostgreSQL 备份及未修复的迁移副本。

## 构建

使用项目 `go.mod` 指定的 Go 版本和 Node.js。Windows 上运行 `./build.ps1`，会安装前端依赖、构建页面并生成 `bin/` 下的 Linux amd64 可执行文件。Linux 上执行 `sh build.sh`，默认按当前 Linux 架构构建；部署前用 `uname -m` 确认服务器架构。产物包括 `skilltree`、`skilltree-migrate`、`skilltree-repair`、`skilltree-backup` 和 `skilltree-curriculum`。前端改动后必须重新构建，才能更新嵌入的页面。

### 本地预览

在项目根目录初始化一个独立的练习库，再启动服务。

```sh
mkdir -p data
go run ./cmd/curriculum -db data/workshop-local.db -init -apply
SKILLTREE_DB=data/workshop-local.db SKILLTREE_LISTEN=127.0.0.1:8080 \
  SKILLTREE_JWT_SECRET="$(openssl rand -base64 32)" go run ./cmd/server
```

初始化只需运行一次。另开终端运行 `cd web && npm ci && npm run dev`，打开 Vite 显示的本地地址。可在网页上注册自主学习账号。

## 课程版本与发布

课程源文件位于 `internal/curriculum/`，当前版本为 `campus-treehole-2026-11-v4`。8 章、46 关、179 项任务围绕同一个匿名互动社区「校园树洞」。学生在同一个仓库中扩展帖子列表、草稿、共享 API、匿名投稿、评论、回复提醒和举报处理，再完成性能验证、试运行与正式发布。最后一次提交包含自己发布的访问链接和同学试用的反馈。学生作品使用 Java 17、Spring Boot 和 MySQL 作为后端练习路线，前两章使用本地编辑器和浏览器。第一关从建文件夹、显示文件扩展名开始。学生先练习查找 7-Zip 官网与准备压缩工具，再从官网下载并安装 VS Code，最后分别验收 ZIP 压缩和在新目录解压。进入关卡直接打开当前要做的任务。概念和代码示例收在「本关参考」，第一关先认识 HTML，再编辑并刷新自己的网页。任务按电脑系统显示步骤，初次使用默认 Windows。

课程 Markdown 中，单独一行的 `:::windows` 或 `:::mac` 开始系统专属内容，单独一行的 `:::` 结束。共用步骤直接写在块外。多个系统可用空格并列，例如 `:::mac linux`。远程服务器命令写在共用正文，两种电脑都需要阅读。

部署人员在系统备份恢复成功后，以服务账号运行以下命令。第一条在事务中完整试跑并回滚；第二条发布本版。已发布的同一版本再次运行会保留教师在后台的后续编辑。程序发现内置课程内容与迁移基线冲突时会报出编号，整次课程更新回滚。

```sh
sudo -u skilltree /opt/skilltree/skilltree-curriculum -db /var/lib/skilltree/skilltree.db
sudo -u skilltree /opt/skilltree/skilltree-curriculum -db /var/lib/skilltree/skilltree.db -apply
```

全新安装可用 `skilltree-curriculum -db /path/to/new.db -init -apply` 建立业务结构并安装课程。已有教学记录的数据库直接使用上面的更新命令。课程版本升级沿用已有的关卡、教程与任务编号，也保留对应的进度、提交和评分。旧版已通过的任务沿用通过状态，原提交正文仍记录当时的作品。已在旧版开始的同学可以把原有列表、存储和接口代码接着用，逐步将原业务数据替换为帖子和评论。运行中的 Web 服务按数据库内容提供课程，发版工具仅在发布时运行。

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
   - `DEEPSEEK_API_KEY`、可选 `DEEPSEEK_MODEL`：供 AI 审核使用；`SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`：供密码重置使用。SMTP 的 465 端口在连接时直接启用 TLS，其他端口要求 STARTTLS（默认 587）。阿里邮箱可使用 `smtp.qiye.aliyun.com:465` 和三方客户端安全密码。缺失配置时对应功能返回明确错误，其他功能仍可使用。
3. 以 root 运行 `sh deploy/install.sh`。脚本创建专用服务用户、安装文件、启动 `skilltree.service` 和每日备份 timer；不会自动从旧系统迁移数据库。
4. `systemctl status skilltree` 查看服务，`journalctl -u skilltree -f` 查看日志；用 `curl http://127.0.0.1:8081/healthz` 检查健康状态。默认只监听本机；局域网直连可设置 `SKILLTREE_LISTEN=0.0.0.0:8081` 并配置防火墙。公网访问应配置域名及 HTTPS 反向代理，代理需支持 WebSocket；不要把明文 HTTP 暴露到公网。

Go 服务会为支持 gzip 的客户端压缩网页文本资源和大体量课程读取响应。Nginx 模板将压缩协商交给 Go，透传 `Accept-Encoding`。带内容哈希的资源可长期缓存，网页入口和其他静态文件通过 ETag 重验证，API 响应禁止缓存。登录、账号恢复和聊天仍使用各自原有的传输方式。

## 备份与恢复

定时器每天把在线一致性快照写入 `/var/lib/skilltree/backups/`，生成后执行 SQLite 完整性检查。备份不会自动删除；应监控 40 GB 磁盘剩余空间并定期复制到另一台机器。也可设置 `SKILLTREE_DB`、`SKILLTREE_BACKUP_OUTPUT` 后运行 `skilltree-backup`，目标必须不存在。

恢复演练：停止服务，将备份复制到一个新路径，先做完整性和业务数据检查，再替换服务使用的数据库并启动。**不要在运行中只复制主数据库文件**；WAL 中可能有尚未合并的数据。恢复操作及其数据库替换应在服务停止后进行。

若回退到不支持会话版本的旧服务，请在启动前为该服务重新生成 `SKILLTREE_JWT_SECRET` 并更新服务环境文件。旧密钥签发的登录凭据会统一失效，所有用户需重新登录。这也会使使用该密钥签发的师徒邀请链接失效，需重新生成。

## 验证

```text
go test ./...
go vet ./...
```

`tests/systemd-smoke.sh` 使用临时合成数据在 Linux 用户级 systemd 中验证注册、学习、审核、重启和备份。`tests/systemd-load.sh` 接收一个离线数据库副本，复制后执行 30 并发的读取测试；不要传入运行中的数据库文件。七牛和 AI 已有本地模拟接口测试，邮件有直接 TLS 和 STARTTLS 测试；正式上线前仍需使用实际服务配置验证连通性和权限。
