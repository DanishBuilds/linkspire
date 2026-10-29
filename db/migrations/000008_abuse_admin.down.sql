-- db/migrations/000008_abuse_admin.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS is_staff;
DROP TABLE IF EXISTS abuse_reports;
