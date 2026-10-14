-- Existing monolith compatibility migration, run once against a BACKUP-VERIFIED
-- source database before using the new binaries. Skip columns already present
-- (e.g. if AutoMigrate or the updated db.sql has already created them).
-- This does not split or copy business data; see deploy/MIGRATION.md.
ALTER TABLE `file` ADD COLUMN `share_version` bigint NOT NULL DEFAULT 0;
ALTER TABLE `share` ADD COLUMN `file_version` bigint NOT NULL DEFAULT 0;
ALTER TABLE `notification` ADD COLUMN `event_id` varchar(64) NULL,
  ADD UNIQUE INDEX `idx_notification_event_id` (`event_id`);
