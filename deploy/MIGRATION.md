# 从模块化单体迁移

全新安装直接使用 `compose.yaml`。已有数据不能仅更换启动命令：新拓扑使用四个独立 schema 和不同的数据卷，不会自动导入原库。

## 迁移窗口

1. 备份原 MySQL、MinIO 对象和配置；记录用户、文件、配额、分享、通知的数量及校验值，先在副本演练。停止原入口和后台 worker，完成或取消未完成的分片上传，确保没有并发写入。
2. 原库若继续运行新版单体，需要新增 `file.share_version`、`share.file_version`、`notification.event_id`。参考 `backend/migrations/microservices_compat_2026.sql`，先检查列与索引，仅执行尚未存在的部分。SQL 不会复制数据，也不应重复执行；不要用含 DROP TABLE 的 `db.sql` 升级已有库。
3. 用新 Compose 初始化四个空 schema 及所属表，然后停止应用服务，仅保留基础设施。保留原库；使用临时迁移账号导出/导入以下数据，显式列出字段，禁止 `SELECT *`（原 file 可能含生成列 active_flag）。导入以目标模型列为准，不复制跨服务外键。

| 源表 | 目标数据库与表 | 要求 |
| --- | --- | --- |
| user | cloudbox_user.user | 保留 id、root_folder_id、密码哈希、头像键和注册时间 |
| file | cloudbox_file.file | 保留全部主键、父子关系、对象键、软删除状态；旧库无 share_version 时填 0 |
| storage_quota / favorite / recycle_bin | cloudbox_file 同名表 | 保留配额与回收站记录，不能靠新用户默认配额替代 |
| share | cloudbox_share.share | 保留 token、有效期与提取码；旧库无 file_version 时填 0；移除指向旧 user/file 的外键 |
| notification | cloudbox_notification.notification | 保留通知 ID 与已读状态；旧通知 event_id 为 NULL |

4. `event_outbox`、`event_inbox` 从空表开始，不把历史数据伪造成新上传事件。旧密码重置令牌不迁移，用户重新申请；当前代码表名为 `password_reset_token`，旧 SQL 导出可能为复数。旧 Redis 上传会话、登录刷新令牌不复用，要求重新登录；更换 JWT secret（auth/user 使用相同新值）。旧 MinIO 桶和对象键必须原样保留或完整复制，检查浏览器可达的签名地址。
5. 检查目标记录数、父目录完整性、配额总量与已用量、对象存在性、分享访问与回收站；保留主键意味着导入后还要检查 auto_increment 大于已有最大 ID。当前开发建表已包含活动文件名、收藏、回收记录和分享唯一约束；升级已有拆分库时先审阅 `backend/migrations/service_constraints_2026.sql`，只执行缺失项。若发现重复数据，先人工确认处理，迁移不会自动删数据。
6. 启动新服务，验证登录、文件列表/下载/上传、分享、恢复、通知，并观察 Outbox、队列、死信。完成验证后再将前端入口切换到网关并开放写入。

这是停机导入流程，未提供自动搬迁或在线双写。不要在迁移时允许新旧应用同时写入各自数据库。

## 回退

新拓扑尚未接受写入时，可停止新应用、恢复原入口及原配置。新拓扑已经产生写入后，需要停止写入、备份新库，将新增/变更数据和对象经审阅的反向迁移同步至旧库，再切回；不能直接启动旧库丢弃新数据。保留 `compose.monolith.yaml` 是为了兼容单体启动，不代表提供自动数据回滚。

## 本地隔离演练

`python3 scripts/integration/microservices_smoke.py --migration-rehearsal` 在本次临时 Docker 项目中：

1. 用 API 建立用户、文件、收藏、分享和回收站数据。
2. 停止七个服务，用 `db.sql` 建立真实旧表结构副本，保留 ID、密码哈希、对象键与关系。
3. 使用显式列清单、排除生成列，在事务中导入四个 schema；逐行比较全部迁移字段并核对父目录和配额。
4. 重启服务，验证登录、下载、分享、收藏、恢复以及迁移后的新写入。
5. 停止服务，将包含新写入的数据反向复制到另一个旧表结构副本，逐行比对并确认原副本未改变。

旧结构部分时间字段精度低于新结构，比较按旧字段可表示的时间精度进行。此演练只操作脚本创建的临时数据，不接受生产 DSN，也没有验证在旧版本应用上运行反向副本；不能当作生产无损回退承诺。未提供真实历史数据时不能宣称已完成真实数据迁移。

新增持久化清理任务表：运行新单体或关闭 AutoMigrate 的 file-service 前，执行 `backend/migrations/object_deletion_2026.sql`。清空回收站后物理对象允许异步删除，`object_deletion` 中保留重试任务；已有待处理任务必须随文件库一起保留，不得在迁移/回退时丢弃。演练只在队列已排空时复制业务数据。
