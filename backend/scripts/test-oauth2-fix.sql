-- 测试OAuth2修复的SQL脚本
-- 检查当前Gmail账户的token状态

-- 查看所有OAuth2类型账户的token状态
SELECT
    id,
    email_address,
    auth_type,
    json_extract(custom_settings, '$.refresh_token') as refresh_token,
    length(json_extract(custom_settings, '$.refresh_token')) as refresh_token_length,
    json_extract(custom_settings, '$.access_token') as access_token,
    length(json_extract(custom_settings, '$.access_token')) as access_token_length,
    last_sync_at,
    is_verified
FROM email_accounts
WHERE auth_type = 'oauth2' AND email_address LIKE '%@gmail.com'
ORDER BY id;

-- 检查是否有错误的token字段被更新的账户
SELECT
    id,
    email_address,
    auth_type,
    token,
    length(token) as token_length,
    custom_settings
FROM email_accounts
WHERE auth_type = 'oauth2' AND token IS NOT NULL AND token != ''
ORDER BY id;