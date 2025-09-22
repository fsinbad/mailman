# 表达式调试器用户指南

## 概述

表达式调试器是一个强大的工具，允许您构建和测试复杂的条件表达式。它支持多种表达式类型，包括字段比较、逻辑运算、函数调用和插件条件。

## 字段访问语法

### 基本语法
使用点表示法 (`.`) 来访问嵌套对象的字段：

```
object.field
object.nested.field
object.array[0].field
```

### 常用字段路径示例

#### 邮件相关字段
```javascript
// 邮件基本信息
email.from          // 发件人地址
email.to            // 收件人地址
email.subject       // 邮件主题
email.body          // 邮件正文
email.size          // 邮件大小(字节)
email.date          // 邮件日期

// 邮件头信息
email.headers.reply-to
email.headers.message-id
email.headers.content-type

// 附件信息
email.attachments[0].filename
email.attachments[0].size
email.attachments[0].content_type
```

#### 用户相关字段
```javascript
user.id             // 用户ID
user.name           // 用户名
user.email          // 用户邮箱
user.role           // 用户角色
user.level          // 用户等级
user.permissions    // 用户权限
```

#### 账户相关字段
```javascript
account.id          // 账户ID
account.name        // 账户名称
account.type        // 账户类型
account.provider    // 邮件提供商
account.domain      // 域名
```

## 表达式类型

### 1. 比较表达式 (comparison)
用于比较字段值与特定值：

```javascript
{
  "type": "comparison",
  "operator": "==",
  "field": "email.from",
  "value": "test@example.com"
}
```

**支持的操作符:**
- `==`: 等于
- `!=`: 不等于
- `>`: 大于
- `>=`: 大于等于
- `<`: 小于
- `<=`: 小于等于
- `contains`: 包含
- `startsWith`: 开头匹配
- `endsWith`: 结尾匹配
- `regex`: 正则表达式匹配

### 2. 逻辑表达式 (and/or/not)
用于组合多个条件：

#### AND条件
```javascript
{
  "type": "and",
  "conditions": [
    {
      "type": "comparison",
      "operator": "==",
      "field": "email.from",
      "value": "test@example.com"
    },
    {
      "type": "comparison",
      "operator": "contains",
      "field": "email.subject",
      "value": "重要"
    }
  ]
}
```

#### OR条件
```javascript
{
  "type": "or",
  "conditions": [
    {
      "type": "comparison",
      "operator": "==",
      "field": "user.role",
      "value": "admin"
    },
    {
      "type": "comparison",
      "operator": "==",
      "field": "user.role",
      "value": "manager"
    }
  ]
}
```

#### NOT条件
```javascript
{
  "type": "not",
  "conditions": [
    {
      "type": "comparison",
      "operator": "==",
      "field": "email.from",
      "value": "spam@example.com"
    }
  ]
}
```

### 3. 插件条件 (plugin)
用于调用插件进行条件判断：

```javascript
{
  "type": "plugin",
  "function": "email_prefix",
  "args": ["test"]
}
```

## 测试数据格式

测试数据应该是一个JSON对象，包含您要测试的所有字段：

```javascript
{
  "email": {
    "from": "test@example.com",
    "to": "admin@example.com",
    "subject": "重要通知",
    "body": "这是一封重要的邮件",
    "size": 1024,
    "date": "2025-07-14T10:30:00Z",
    "headers": {
      "reply-to": "noreply@example.com",
      "message-id": "msg123@example.com"
    },
    "attachments": [
      {
        "filename": "document.pdf",
        "size": 2048,
        "content_type": "application/pdf"
      }
    ]
  },
  "user": {
    "id": 1,
    "name": "管理员",
    "email": "admin@example.com",
    "role": "admin",
    "level": 5,
    "permissions": ["read", "write", "delete"]
  },
  "account": {
    "id": 1,
    "name": "测试账户",
    "type": "gmail",
    "provider": "Google",
    "domain": "example.com"
  }
}
```

## 实际使用示例

### 示例1: 检查重要邮件
```javascript
// 条件：来自特定发件人且主题包含"重要"
{
  "type": "and",
  "conditions": [
    {
      "type": "comparison",
      "operator": "==",
      "field": "email.from",
      "value": "boss@company.com"
    },
    {
      "type": "comparison",
      "operator": "contains",
      "field": "email.subject",
      "value": "重要"
    }
  ]
}
```

### 示例2: 过滤垃圾邮件
```javascript
// 条件：不是来自垃圾邮件发件人
{
  "type": "not",
  "conditions": [
    {
      "type": "or",
      "conditions": [
        {
          "type": "comparison",
          "operator": "contains",
          "field": "email.from",
          "value": "spam"
        },
        {
          "type": "comparison",
          "operator": "contains",
          "field": "email.subject",
          "value": "广告"
        }
      ]
    }
  ]
}
```

### 示例3: 大附件检查
```javascript
// 条件：邮件大小超过1MB
{
  "type": "comparison",
  "operator": ">",
  "field": "email.size",
  "value": 1048576
}
```

### 示例4: 管理员权限检查
```javascript
// 条件：用户是管理员且等级大于等于5
{
  "type": "and",
  "conditions": [
    {
      "type": "comparison",
      "operator": "==",
      "field": "user.role",
      "value": "admin"
    },
    {
      "type": "comparison",
      "operator": ">=",
      "field": "user.level",
      "value": 5
    }
  ]
}
```

## 调试技巧

1. **逐步构建**: 先创建简单的条件，然后逐步添加复杂性
2. **测试边界情况**: 使用不同的测试数据来验证表达式的正确性
3. **使用正确的数据类型**: 确保字段值与期望的数据类型匹配
4. **检查字段路径**: 确保字段路径正确对应测试数据的结构
5. **验证操作符**: 选择正确的操作符进行比较

## 常见问题

**Q: 如何访问数组中的元素？**
A: 使用方括号语法，如 `email.attachments[0].filename`

**Q: 字段不存在时会发生什么？**
A: 系统会返回 `undefined`，在条件判断中被视为 `false`

**Q: 如何处理包含特殊字符的字段名？**
A: 对于包含特殊字符的字段名，可以使用方括号语法，如 `email.headers["content-type"]`

**Q: 表达式评估失败时如何调试？**
A: 检查错误消息，确保字段路径正确，数据类型匹配，操作符使用正确

## 支持的插件

当前支持的插件条件：
- `email_prefix`: 检查邮件发件人是否以特定前缀开头
- `email_suffix`: 检查邮件发件人是否以特定后缀结尾
- `email_account_set`: 检查邮件发件人是否在特定账户集合中
- `email_time_range`: 检查邮件是否在特定时间范围内
- `email_size`: 检查邮件大小是否符合条件

有关插件的详细使用方法，请参考插件文档。