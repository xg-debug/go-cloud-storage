-- MySQL dump 10.13  Distrib 8.4.5, for Win64 (x86_64)
--
-- Host: localhost    Database: file-store
-- ------------------------------------------------------
-- Server version	8.4.5
--
-- 注意：本文件已脱敏（不含任何用户数据），仅包含表结构 DDL。
-- 已有数据库的增量迁移脚本见 backend/migrations/。

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `favorite`
--

DROP TABLE IF EXISTS `favorite`;
CREATE TABLE `favorite` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL COMMENT '收藏者ID（文件的拥有者）',
  `file_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件ID',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_fav` (`user_id`,`file_id`),
  KEY `favorite_ibfk_2` (`file_id`),
  CONSTRAINT `favorite_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `favorite_ibfk_2` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `file`
--
-- 说明：
-- 1. active_flag 为虚拟生成列：未删除(is_deleted=0)时为 1，已删除时为 NULL。
--    配合唯一索引 uniq_user_parent_name，保证"同一目录下未删除的同名文件唯一"，
--    同时允许软删后重建同名文件（旧删除行不参与唯一性约束）。
-- 2. idx_user_hash / idx_file_hash / idx_user_deleted 为高频查询（秒传、列表、重复文件检测）补充索引。
--

DROP TABLE IF EXISTS `file`;
CREATE TABLE `file` (
  `id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'UUID或OSS标识',
  `user_id` int NOT NULL COMMENT '用户ID',
  `name` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '原始文件名',
  `size` bigint unsigned DEFAULT NULL COMMENT '字节大小',
  `size_str` varchar(20) COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '文件可读大小，如2.8MB',
  `is_dir` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否为目录',
  `file_extension` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '文件扩展名',
  `file_url` varchar(255) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件访问URL（私有桶下仅作内部引用，对外一律使用预签名URL）',
  `thumbnail_url` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '缩略图URL（同上）',
  `oss_object_key` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'OSS对象键',
  `file_hash` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'SHA256',
  `parent_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '软删除标志',
  `active_flag` tinyint(1) GENERATED ALWAYS AS ((case when (`is_deleted` = 0) then 1 else NULL end)) VIRTUAL COMMENT '辅助唯一索引列',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_parent_name` (`user_id`,`parent_id`,`name`,`active_flag`),
  KEY `file_ibfk_2` (`parent_id`),
  KEY `idx_user_hash` (`user_id`,`file_hash`),
  KEY `idx_file_hash` (`file_hash`),
  KEY `idx_user_deleted` (`user_id`,`is_deleted`),
  CONSTRAINT `file_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `file_ibfk_2` FOREIGN KEY (`parent_id`) REFERENCES `file` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `notification`
--

DROP TABLE IF EXISTS `notification`;
CREATE TABLE `notification` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `title` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'info',
  `is_read` tinyint(1) NOT NULL DEFAULT '0',
  `link` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_notifications_user_id` (`user_id`),
  KEY `idx_notifications_user_read` (`user_id`,`is_read`),
  KEY `idx_notifications_type` (`type`),
  KEY `idx_notifications_created_at` (`created_at`),
  CONSTRAINT `fk_notifications_user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知表';

--
-- Table structure for table `password_reset_tokens`
--

DROP TABLE IF EXISTS `password_reset_tokens`;
CREATE TABLE `password_reset_tokens` (
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

--
-- Table structure for table `recycle_bin`
--

DROP TABLE IF EXISTS `recycle_bin`;
CREATE TABLE `recycle_bin` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '回收站记录ID',
  `file_id` varchar(40) COLLATE utf8mb4_general_ci NOT NULL COMMENT '文件ID',
  `user_id` int NOT NULL COMMENT '所属用户ID',
  `deleted_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '删除时间',
  `expire_at` timestamp NOT NULL COMMENT '过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_file` (`user_id`,`file_id`),
  KEY `idx_expire_at` (`expire_at`),
  KEY `fk_recycle_file` (`file_id`),
  CONSTRAINT `fk_recycle_file` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_recycle_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `share`
--

DROP TABLE IF EXISTS `share`;
CREATE TABLE `share` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL COMMENT '分享者ID',
  `file_id` varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '分享的文件/文件夹ID',
  `share_token` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '唯一分享标识',
  `extraction_code` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '提取码',
  `expire_time` datetime DEFAULT NULL COMMENT '过期时间',
  `access_count` int DEFAULT '0' COMMENT '访问次数',
  `download_count` int DEFAULT '0' COMMENT '下载次数',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '软删除标志',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_share_token` (`share_token`),
  UNIQUE KEY `uk_share_file` (`file_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_share_user_deleted` (`user_id`,`is_deleted`),
  CONSTRAINT `share_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `share_ibfk_2` FOREIGN KEY (`file_id`) REFERENCES `file` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户分享表';

--
-- Table structure for table `storage_quota`
--

DROP TABLE IF EXISTS `storage_quota`;
CREATE TABLE `storage_quota` (
  `user_id` int NOT NULL,
  `total` bigint DEFAULT '10737418240' COMMENT '总存储空间(字节);默认10GB',
  `used` bigint DEFAULT '0' COMMENT '已使用存储空间(字节)',
  `used_percent` float GENERATED ALWAYS AS (round(((`used` * 100.0) / `total`),2)) VIRTUAL COMMENT '使用百分比',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`user_id`),
  CONSTRAINT `storage_quota_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Table structure for table `upload_task`（已废弃，分片上传会话已迁移至 Redis，保留仅为兼容旧数据）
--

DROP TABLE IF EXISTS `upload_task`;
CREATE TABLE `upload_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `file_hash` varchar(64) NOT NULL COMMENT '文件唯一hash (SHA256/MD5)',
  `user_id` int NOT NULL COMMENT '用户ID',
  `file_name` varchar(255) NOT NULL COMMENT '原始文件名',
  `file_size` bigint NOT NULL COMMENT '字节大小',
  `total_chunks` int NOT NULL COMMENT '分片总数',
  `upload_id` varchar(255) DEFAULT NULL COMMENT 'OSS multipart uploadId',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '任务状态: 0-上传中, 1-已完成, 2-失败',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_file_user` (`file_hash`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='大文件上传任务表(已废弃)';

--
-- Table structure for table `user`
--

DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_general_ci NOT NULL,
  `email` varchar(100) COLLATE utf8mb4_general_ci NOT NULL,
  `phone` varchar(20) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `password` varchar(100) COLLATE utf8mb4_general_ci NOT NULL COMMENT 'bcrypt哈希',
  `avatar` varchar(255) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `open_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `register_time` datetime(3) DEFAULT NULL,
  `root_folder_id` varchar(40) COLLATE utf8mb4_general_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_user_username` (`username`),
  UNIQUE KEY `uni_user_email` (`email`),
  UNIQUE KEY `uni_user_phone` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
