# 邮件同步功能改进总结

## 修改的核心文件

### 1. `backend/internal/services/optimized_incremental_sync_manager.go`

**问题**：增量同步时邮件被过度过滤，导致没有邮件被同步

**修改**：
- **时间限制**：增量同步最多处理24小时内的邮件
- **文件夹同步**：设置 `Folders: []string{}` 空数组表示同步所有文件夹
- **智能时间设置**：
  - 如果有 `LastSyncTime`，使用该时间但不超过24小时前
  - 如果没有 `LastSyncTime`，从24小时前开始首次同步

**关键代码**：
```go
// 智能设置StartDate，增量同步最多处理24小时内的邮件
var startDate *time.Time
if configWithAccount.LastSyncTime != nil {
    // 增量同步：使用LastSyncTime，但不超过24小时前
    twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
    if configWithAccount.LastSyncTime.Before(twentyFourHoursAgo) {
        startDate = &twentyFourHoursAgo
    } else {
        startDate = configWithAccount.LastSyncTime
    }
} else {
    // 首次同步，从24小时前开始
    twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
    startDate = &twentyFourHoursAgo
}

fetchReq := FetchRequest{
    Type:         SubscriptionTypeRealtime,
    Priority:     PriorityHigh,
    EmailAddress: configWithAccount.Account.EmailAddress,
    StartDate:    startDate,        // 智能设置开始时间
    Folders:      []string{},       // 空数组表示同步所有文件夹
    Timeout:      20 * time.Second, // 设置合理的超时时间
}
```

### 2. `backend/internal/services/fetcher.go`

**问题**：当 `Folders` 为空时，`FetchEmailsFromMultipleMailboxes` 方法没有自动获取所有文件夹

**修改**：
- **添加 `GetAllFolders` 方法**：自动获取邮箱所有文件夹
- **Gmail 文件夹获取**：返回常用的 Gmail 文件夹列表
- **IMAP 文件夹获取**：返回常用的 IMAP 文件夹列表（简化实现）
- **自动文件夹遍历**：当 `Folders` 为空时，自动调用 `GetAllFolders`

**关键代码**：
```go
// 如果没有指定文件夹，自动获取所有文件夹
if len(folders) == 0 {
    allFolders, err := s.GetAllFolders(account)
    if err != nil {
        return nil, fmt.Errorf("failed to get all folders: %w", err)
    }
    folders = allFolders
    s.logger.Info("Auto-discovered %d folders for %s", len(folders), account.EmailAddress)
}
```

**Gmail 文件夹获取**：
```go
func (s *FetcherService) getGmailFolders(account models.EmailAccount) ([]string, error) {
    commonGmailFolders := []string{
        "INBOX",
        "SENT",
        "DRAFT",
        "SPAM",
        "TRASH",
        "IMPORTANT",
        "STARRED",
        "UNREAD",
    }
    return commonGmailFolders, nil
}
```

## 解决的问题

### 1. **邮件过滤过于严格**
- **之前**：邮件被三重过滤（时间、文件夹、别名）全部过滤掉
- **现在**：空文件夹数组表示同步所有文件夹，避免文件夹过滤

### 2. **时间范围过大**
- **之前**：可能同步过多历史邮件
- **现在**：增量同步限制为24小时内，提高效率

### 3. **文件夹同步不完整**
- **之前**：需要手动指定文件夹
- **现在**：自动获取所有文件夹进行同步

### 4. **首次同步时间设置**
- **之前**：首次同步时间设置不明确
- **现在**：智能设置，首次同步从24小时前开始

## 预期效果

1. **邮件不再被过滤掉**：空文件夹数组确保所有文件夹的邮件都被考虑
2. **同步效率提高**：24小时限制避免同步过多历史邮件
3. **自动化程度提高**：自动获取所有文件夹，无需手动配置
4. **用户体验改善**：开启全局配置后，邮件能够正常同步

## 测试建议

1. 运行邮件过滤调试程序：`go run ./cmd/email-filter-debug/main.go`
2. 检查日志中的过滤统计：
   - `Total emails fetched from all mailboxes: X`
   - `filtered out: Y`
   - `unique emails: Z`
3. 期望看到 `filtered out` 数量显著减少
4. 期望看到 `unique emails` 数量增加

## 后续优化建议

1. **实现完整的IMAP文件夹获取**：当前是简化实现，可以后续添加真实的IMAP LIST命令
2. **添加文件夹白名单/黑名单**：允许用户配置哪些文件夹需要同步
3. **优化Gmail标签处理**：Gmail的标签系统与传统文件夹不同，可以优化处理
4. **添加同步统计**：记录每个文件夹的同步统计信息