-- Run against file-service schema (or legacy monolith) before new binaries.
CREATE TABLE IF NOT EXISTS `object_deletion` (
 `id` varchar(64) NOT NULL PRIMARY KEY,
 `object_key` varchar(1024) NOT NULL,
 `attempts` int NOT NULL DEFAULT 0,
 `next_attempt_at` datetime(3) NOT NULL,
 `created_at` datetime(3) NULL,
 KEY `idx_object_deletion_next_attempt_at` (`next_attempt_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
