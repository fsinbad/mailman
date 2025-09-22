-- 删除现有的Outlook OAuth2配置
DELETE FROM o_auth2_global_configs WHERE provider_type = 'outlook';

-- 插入参考代码中的正确Outlook OAuth2配置
INSERT INTO o_auth2_global_configs (
    name,
    provider_type,
    client_id,
    client_secret,
    redirect_uri,
    scopes,
    is_enabled,
    created_at,
    updated_at
) VALUES (
    'Windyl Outlook OAuth2',
    'outlook',
    '6eb766a8-77ae-44ee-91ee-fba28c5dd776',
    '',
    'https://oauth.windyl.de/callback/outlook',
    '["offline_access","https://outlook.office.com/IMAP.AccessAsUser.All","https://outlook.office.com/POP.AccessAsUser.All","https://outlook.office.com/SMTP.Send"]',
    1,
    datetime('now'),
    datetime('now')
);

-- 查看结果
SELECT * FROM o_auth2_global_configs WHERE provider_type = 'outlook';