-- Apply with all writers stopped and a verified backup. Inspect SHOW INDEX and
-- SHOW COLUMNS first; execute only missing statements. Existing duplicate rows
-- must be reviewed and resolved explicitly; never delete them automatically.
-- cloudbox_file (or legacy monolith):
ALTER TABLE `file` ADD COLUMN `active_flag` tinyint GENERATED ALWAYS AS
 (CASE WHEN `is_deleted` = 0 THEN 1 ELSE NULL END) VIRTUAL;
CREATE UNIQUE INDEX `uniq_user_parent_name` ON `file` (`user_id`,`parent_id`,`name`,`active_flag`);
CREATE UNIQUE INDEX `uniq_fav` ON `favorite` (`user_id`,`file_id`);
CREATE UNIQUE INDEX `uniq_user_file` ON `recycle_bin` (`user_id`,`file_id`);
-- cloudbox_share (or legacy monolith):
CREATE UNIQUE INDEX `uk_share_file` ON `share` (`file_id`);
