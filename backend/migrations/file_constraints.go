package migrations

import "gorm.io/gorm"

// EnsureFileConstraints preserves the legacy active-name invariant. A generated
// NULL for deleted rows allows multiple recycled names without blocking restore
// checks for live files. Duplicate live rows fail migration; never discard data.
func EnsureFileConstraints(db *gorm.DB) error {
	if !db.Migrator().HasColumn("file", "active_flag") {
		if err := db.Exec("ALTER TABLE `file` ADD COLUMN `active_flag` tinyint GENERATED ALWAYS AS (CASE WHEN `is_deleted` = 0 THEN 1 ELSE NULL END) VIRTUAL").Error; err != nil {
			return err
		}
	}
	if !db.Migrator().HasIndex("file", "uniq_user_parent_name") {
		return db.Exec("CREATE UNIQUE INDEX `uniq_user_parent_name` ON `file` (`user_id`,`parent_id`,`name`,`active_flag`)").Error
	}
	return nil
}
