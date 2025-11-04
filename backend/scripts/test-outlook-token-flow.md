# Outlook Token 添加流程测试文档

## 📋 测试步骤

### 1. 前端界面测试
- [ ] 访问邮箱账户管理页面
- [ ] 点击"添加账户"下拉按钮
- [ ] 确认看到"新增Outlook(已有Token)"选项
- [ ] 点击该选项，打开模态框

### 2. Token输入表单测试
- [ ] 验证表单字段：邮箱地址、Client ID、Refresh Token、Access Token(可选)
- [ ] 验证必填字段验证逻辑
- [ ] 验证Token显示/隐藏功能
- [ ] 验证代理设置选项

### 3. 工作流程测试
- [ ] 步骤1：输入Token信息 → 点击"创建账户"
- [ ] 步骤2：验证连接 → 确认连接成功
- [ ] 步骤3：首次同步 → 配置同步参数并执行
- [ ] 步骤4：同步配置 → 设置自动同步
- [ ] 步骤5：完成 → 确认成功提示

### 4. 后端API测试
#### 4.1 Upsert端点测试
```bash
curl -X POST http://localhost:8080/api/accounts/upsert \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "email_address": "test@outlook.com",
    "mail_provider_id": 1,
    "auth_type": "oauth2",
    "custom_settings": {
      "client_id": "test_client_id",
      "refresh_token": "test_refresh_token",
      "access_token": "test_access_token"
    }
  }'
```

#### 4.2 账户验证测试
```bash
curl -X POST http://localhost:8080/api/accounts/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"account_id": 1}'
```

#### 4.3 同步配置测试
```bash
curl -X POST http://localhost:8080/api/accounts/1/sync-config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "enable_auto_sync": true,
    "sync_interval": 300,
    "sync_folders": ["INBOX"]
  }'
```

### 5. 数据持久化测试
- [ ] 验证账户信息正确保存到数据库
- [ ] 验证CustomSettings字段包含OAuth2信息
- [ ] 验证同步配置正确创建
- [ ] 验证账户验证状态更新

## 🔧 故障排除

### 常见问题
1. **账户创建失败**：检查Outlook邮件提供商配置
2. **验证连接失败**：检查Token有效性
3. **同步失败**：检查IMAP配置和网络连接
4. **代理设置问题**：验证代理服务器配置

### 日志检查
```bash
# 查看后端日志
tail -f /var/log/mailman/mailman.log

# 查看OAuth2相关日志
grep "OAuth2" /var/log/mailman/mailman.log

# 查看账户操作日志
grep "Account" /var/log/mailman/mailman.log
```

## ✅ 验收标准

1. **前端界面**：按钮显示正确，模态框功能完整
2. **API功能**：upsert接口正常工作
3. **数据一致性**：账户和同步配置正确保存
4. **流程完整性**：从Token输入到完成的整个流程无中断
5. **错误处理**：提供清晰的错误信息和恢复机制

## 📊 测试数据

### 测试账户信息
- 邮箱地址：`test@outlook.com`
- Client ID：`test_client_id_123`
- Refresh Token：`test_refresh_token_456`
- Access Token：`test_access_token_789`

### 预期结果
- 账户创建/更新成功
- OAuth2信息正确保存到CustomSettings
- 连接验证通过
- 同步配置创建成功
- 前端显示完整流程

## 🔄 回归测试

1. 确认原有Gmail OAuth2流程不受影响
2. 确认普通账户创建功能正常
3. 确认同步功能正常工作
4. 确认账户管理界面其他功能正常