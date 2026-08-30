-- 已有数据库增量迁移脚本（按顺序执行）
-- 适用：基于旧版 db.sql 初始化、已存在业务数据的库
-- 说明：每条 ALTER 尽量幂等；若重复执行报错，可忽略"重复列/重复键"类错误。

-- ============================================================
-- 1. file 表：软删后可重建同名文件（唯一索引不再包含软删行）
--    方案：新增虚拟生成列 active_flag（未删除=1，已删除=NULL），
--    唯一索引改为 (user_id, parent_id, name, active_flag)。
--    NULL 不参与唯一性约束 => 软删行可重复，仅未删除行保持唯一。
-- ============================================================
ALTER TABLE `file`
  ADD COLUMN `active_flag` tinyint(1)
  GENERATED ALWAYS AS ((case when (`is_deleted` = 0) then 1 else NULL end)) VIRTUAL
  COMMENT '辅助唯一索引列' AFTER `is_deleted`;

ALTER TABLE `file` DROP INDEX `uniq_user_parent_name`;
ALTER TABLE `file`
  ADD UNIQUE KEY `uniq_user_parent_name` (`user_id`,`parent_id`,`name`,`active_flag`);

-- ============================================================
-- 2. file 表：补充高频查询索引（秒传、重复文件检测、列表过滤）
-- ============================================================
ALTER TABLE `file` ADD KEY `idx_user_hash` (`user_id`,`file_hash`);
ALTER TABLE `file` ADD KEY `idx_file_hash` (`file_hash`);
ALTER TABLE `file` ADD KEY `idx_user_deleted` (`user_id`,`is_deleted`);

-- ============================================================
-- 3. share 表：file_id 唯一索引（防并发重复分享）+ 列表过滤索引
--    若已有重复行，先执行下面的清理语句再建唯一索引：
--    保留每组 (file_id) 中最新的一条，其余软删。
-- ============================================================
-- 清理重复分享（可选，仅在报错时执行）：
-- UPDATE `share` s
-- JOIN (
--   SELECT file_id, MAX(id) AS keep_id
--   FROM `share`
--   GROUP BY file_id
--   HAVING COUNT(*) > 1
-- ) t ON s.file_id = t.file_id AND s.id <> t.keep_id
-- SET s.is_deleted = 1;

ALTER TABLE `share` ADD UNIQUE KEY `uk_share_file` (`file_id`);
ALTER TABLE `share` ADD KEY `idx_share_user_deleted` (`user_id`,`is_deleted`);

-- ============================================================
-- 4. notification 表：未读数查询联合索引
-- ============================================================
ALTER TABLE `notification` ADD KEY `idx_notifications_user_read` (`user_id`,`is_read`);

-- ============================================================
-- 5. password_reset_tokens 表（新环境直接由 db.sql 创建；老库补建）
-- ============================================================
CREATE TABLE IF NOT EXISTS `password_reset_tokens` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `token` varchar(128) NOT NULL,
  `expires_at` datetime NOT NULL,
  `used` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_token` (`token`),
  KEY `idx_prt_user_id` (`user_id`),
  CONSTRAINT `fk_prt_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='密码重置令牌表';

-- ============================================================
-- 6. user 表：root_folder_id 类型修正（longtext -> varchar(40)）
--    长文本列无法直接 MODIFY 为短列，需先建新列再复制再删旧列。
-- ============================================================
ALTER TABLE `user`
  ADD COLUMN `root_folder_id_v2` varchar(40) COLLATE utf8mb4_general_ci DEFAULT NULL AFTER `root_folder_id`;

UPDATE `user` SET `root_folder_id_v2` = LEFT(`root_folder_id`, 40) WHERE `root_folder_id` IS NOT NULL;

ALTER TABLE `user` DROP COLUMN `root_folder_id`;
ALTER TABLE `user` CHANGE COLUMN `root_folder_id_v2` `root_folder_id` varchar(40) COLLATE utf8mb4_general_ci DEFAULT NULL;
