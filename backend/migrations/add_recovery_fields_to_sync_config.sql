-- Add recovery-related fields to email_account_sync_configs table
-- These fields support automatic account recovery after OAuth errors

-- SQLite requires separate ALTER TABLE statements for each column
ALTER TABLE email_account_sync_configs ADD COLUMN recovery_attempts INTEGER DEFAULT 0;
ALTER TABLE email_account_sync_configs ADD COLUMN last_recovery_attempt TIMESTAMP NULL;

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_sync_config_auto_disabled_recovery ON email_account_sync_configs(auto_disabled, last_error_time, recovery_attempts);
CREATE INDEX IF NOT EXISTS idx_sync_config_recovery_attempt ON email_account_sync_configs(last_recovery_attempt);