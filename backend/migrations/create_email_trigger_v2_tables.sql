-- Create email trigger v2 tables

-- Create trigger table
CREATE TABLE IF NOT EXISTS email_triggers_v2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    expressions TEXT NOT NULL, -- JSON array of expressions
    actions TEXT NOT NULL, -- JSON array of actions
    total_executions INTEGER NOT NULL DEFAULT 0,
    success_executions INTEGER NOT NULL DEFAULT 0,
    last_executed_at TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create trigger execution log table
CREATE TABLE IF NOT EXISTS trigger_execution_logs_v2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_id INTEGER NOT NULL,
    trigger_name TEXT NOT NULL,
    email_id INTEGER NOT NULL,
    status TEXT NOT NULL, -- 'success', 'failed', 'partial'
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration INTEGER NOT NULL, -- milliseconds
    condition_result BOOLEAN NOT NULL,
    condition_eval TEXT, -- JSON of condition evaluation details
    actions_executed INTEGER NOT NULL DEFAULT 0,
    actions_succeeded INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    action_results TEXT, -- JSON array of action execution results
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trigger_id) REFERENCES email_triggers_v2(id) ON DELETE CASCADE
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_trigger_execution_logs_v2_trigger_id ON trigger_execution_logs_v2(trigger_id);
CREATE INDEX IF NOT EXISTS idx_trigger_execution_logs_v2_email_id ON trigger_execution_logs_v2(email_id);
CREATE INDEX IF NOT EXISTS idx_trigger_execution_logs_v2_status ON trigger_execution_logs_v2(status);
CREATE INDEX IF NOT EXISTS idx_trigger_execution_logs_v2_start_time ON trigger_execution_logs_v2(start_time);

-- Create migration function to convert old triggers to new format
CREATE TRIGGER IF NOT EXISTS migrate_old_triggers_to_v2
AFTER INSERT ON email_triggers_v2
WHEN (SELECT COUNT(*) FROM email_triggers_v2) = 1 AND (SELECT COUNT(*) FROM email_triggers) > 0
BEGIN
    -- This is a placeholder for the migration logic
    -- In a real implementation, we would insert code here to convert old triggers to the new format
    -- Since SQLite doesn't support complex procedural logic in triggers,
    -- the actual migration would be done in application code
    
    -- For example:
    -- INSERT INTO email_triggers_v2 (name, description, enabled, expressions, actions)
    -- SELECT name, description, enabled, json_array(json_object('id', id, 'type', 'condition', ...)), json_array(...)
    -- FROM email_triggers;
END;