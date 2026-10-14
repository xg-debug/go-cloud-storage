#!/bin/sh
(
set -eu
# This script runs only when MySQL initializes an EMPTY dedicated volume.
# NO_BACKSLASH_ESCAPES + quote doubling permits arbitrary password characters.
escape_sql() { printf '%s' "$1" | sed "s/'/''/g"; }
user_password=$(escape_sql "$GCS_USER_DB_PASSWORD")
file_password=$(escape_sql "$GCS_FILE_DB_PASSWORD")
share_password=$(escape_sql "$GCS_SHARE_DB_PASSWORD")
notification_password=$(escape_sql "$GCS_NOTIFICATION_DB_PASSWORD")
MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot <<SQL
SET SESSION sql_mode='NO_BACKSLASH_ESCAPES';
CREATE DATABASE cloudbox_user CHARACTER SET utf8mb4;
CREATE DATABASE cloudbox_file CHARACTER SET utf8mb4;
CREATE DATABASE cloudbox_share CHARACTER SET utf8mb4;
CREATE DATABASE cloudbox_notification CHARACTER SET utf8mb4;
CREATE USER 'cloudbox_user'@'%' IDENTIFIED BY '$user_password';
CREATE USER 'cloudbox_file'@'%' IDENTIFIED BY '$file_password';
CREATE USER 'cloudbox_share'@'%' IDENTIFIED BY '$share_password';
CREATE USER 'cloudbox_notification'@'%' IDENTIFIED BY '$notification_password';
GRANT ALL ON cloudbox_user.* TO 'cloudbox_user'@'%';
GRANT ALL ON cloudbox_file.* TO 'cloudbox_file'@'%';
GRANT ALL ON cloudbox_share.* TO 'cloudbox_share'@'%';
GRANT ALL ON cloudbox_notification.* TO 'cloudbox_notification'@'%';
SQL

)
