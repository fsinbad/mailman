# 邮件内容提取功能测试文档

## 概述

本文档说明了为邮件内容提取功能编写的单元测试，用于验证各种类型的表达式解析器是否可以正常工作。

## 测试文件

### 1. 单元测试文件 (`internal/services/extractor_test.go`)

包含完整的单元测试，测试以下功能：

- **正则表达式提取器** (`ExtractorTypeRegex`)
- **JavaScript提取器** (`ExtractorTypeJS`)
- **Go模板提取器** (`ExtractorTypeGoTemplate`)
- **匹配条件功能**
- **不同字段提取**
- **多个提取器组合**
- **边界情况处理**

### 2. 集成测试脚本 (`cmd/test-extractor/main.go`)

提供完整的集成测试，演示各种提取器的实际使用场景。

## 测试用例详情

### 正则表达式提取器测试

```go
// 测试订单号提取
pattern: `ORD-\d{4}-\d{3}`
field: ExtractorFieldSubject
expected: ["ORD-2024-001"]

// 测试金额提取
pattern: `￥[\d.]+`
field: ExtractorFieldBody
expected: ["￥299.99"]

// 测试邮箱地址提取
pattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
field: ExtractorFieldBody
expected: ["support@example.com"]

// 测试URL提取
pattern: `https?://[^\s]+`
field: ExtractorFieldBody
expected: ["https://shop.example.com"]

// 测试手机号提取
pattern: `1[3-9]\d{9}`
field: ExtractorFieldBody
expected: ["13800138000"]

// 测试替换语法
pattern: `订单号：(ORD-\d{4}-\d{3})|||$1`
field: ExtractorFieldSubject
expected: ["ORD-2024-001"]
```

### JavaScript提取器测试

```javascript
// 提取订单号
var matches = [];
for (var i = 0; i < parsedContent.length; i++) {
    var match = parsedContent[i].match(/ORD-\d{4}-\d{3}/);
    if (match) matches.push(match[0]);
}
return matches;

// 复杂数据处理
var result = [];
for (var i = 0; i < parsedContent.length; i++) {
    var text = parsedContent[i];
    
    // 提取订单号
    var orderMatch = text.match(/订单号：([A-Z0-9-]+)/);
    if (orderMatch) {
        result.push("ORDER:" + orderMatch[1]);
    }
    
    // 提取金额
    var priceMatch = text.match(/￥([\d.]+)/);
    if (priceMatch) {
        result.push("PRICE:" + priceMatch[1]);
    }
    
    // 提取快递单号
    var trackingMatch = text.match(/快递单号：([A-Z0-9]+)/);
    if (trackingMatch) {
        result.push("TRACKING:" + trackingMatch[1]);
    }
}
return result;
```

### Go模板提取器测试

```go
// 提取订单信息
template: `{{range (regex "ORD-\\d{4}-\\d{3}" .AllText)}}{{.}}{{end}}`
expected: ["ORD-2024-001"]

// 使用内置函数
template: `{{if contains .Subject "订单确认"}}{{.Subject | replace "订单确认 - " ""}}{{end}}`
expected: ["订单号：ORD-2024-001"]

// 提取邮箱地址
template: `{{range (extractEmails .AllText)}}{{.}}{{end}}`
expected: ["support@example.com"]

// 提取链接
template: `{{range (extractLinks .AllText)}}{{.}}{{end}}`
expected: ["https://shop.example.com"]

// 条件判断
template: `{{if contains .Subject "订单"}}ORDER_EMAIL{{else}}OTHER_EMAIL{{end}}`
expected: ["ORDER_EMAIL"]
```

### 匹配条件测试

测试在提取前进行条件匹配：

```go
// 正则表达式匹配条件
matchConfig: "订单确认"
extractType: ExtractorTypeRegex
shouldMatch: true

// JavaScript匹配条件
matchConfig: `
    return parsedContent.some(function(text) {
        return text.includes("订单确认");
    });
`
extractType: ExtractorTypeJS
shouldMatch: true

// Go模板匹配条件
matchConfig: `{{contains .Subject "订单确认"}}`
extractType: ExtractorTypeGoTemplate
shouldMatch: true
```

### 不同字段提取测试

测试从不同邮件字段提取内容：

- **ExtractorFieldFrom**: 发件人字段
- **ExtractorFieldTo**: 收件人字段
- **ExtractorFieldCC**: 抄送字段
- **ExtractorFieldSubject**: 主题字段
- **ExtractorFieldBody**: 正文字段
- **ExtractorFieldHTMLBody**: HTML正文字段
- **ExtractorFieldAll**: 所有字段
- **ExtractorFieldHeaders**: 头部字段（当前为空）

### 多个提取器组合测试

测试同时使用多个提取器：

```go
configs := []ExtractorConfig{
    {
        Field:   ExtractorFieldSubject,
        Type:    ExtractorTypeRegex,
        Extract: `ORD-\d{4}-\d{3}`,
    },
    {
        Field:   ExtractorFieldBody,
        Type:    ExtractorTypeRegex,
        Extract: `￥[\d.]+`,
    },
    {
        Field:   ExtractorFieldBody,
        Type:    ExtractorTypeJS,
        Extract: `/* JavaScript提取手机号 */`,
    },
}
```

### 边界情况测试

- 空邮件内容
- 空提取器配置
- 不支持的提取器类型
- 无效的正则表达式
- JavaScript语法错误
- 无效的Go模板语法

## 测试数据

使用真实的邮件数据进行测试：

```go
email := models.Email{
    Subject: "订单确认 - 订单号：ORD-2024-001",
    From: models.StringSlice{"sender@example.com", "noreply@shop.com"},
    To: models.StringSlice{"recipient@example.com"},
    Cc: models.StringSlice{"cc@example.com"},
    Body: `亲爱的客户，

您的订单已确认：
订单号：ORD-2024-001
总金额：￥299.99
快递单号：SF1234567890
联系电话：13800138000

感谢您的购买！

网站：https://shop.example.com
邮箱：support@example.com`,
    HTMLBody: `<html>
<body>
<h1>订单确认</h1>
<p>您的订单已确认：</p>
<ul>
<li>订单号：<strong>ORD-2024-001</strong></li>
<li>总金额：<strong>￥299.99</strong></li>
<li>快递单号：<strong>SF1234567890</strong></li>
<li>联系电话：<strong>13800138000</strong></li>
</ul>
<p>网站：<a href="https://shop.example.com">https://shop.example.com</a></p>
<p>邮箱：<a href="mailto:support@example.com">support@example.com</a></p>
</body>
</html>`,
}
```

## 如何运行测试

### 运行单元测试

```bash
cd backend
go test ./internal/services -v
```

### 运行集成测试

```bash
cd backend
go run ./cmd/test-extractor/main.go
```

## 测试覆盖的功能

✅ **正则表达式提取器**
- 基本模式匹配
- 捕获组和替换语法
- 多个匹配结果
- 无效正则表达式处理

✅ **JavaScript提取器**
- 基本JavaScript执行
- 复杂数据处理逻辑
- 数组和对象操作
- 错误处理和语法验证

✅ **Go模板提取器**
- 模板语法解析
- 内置函数使用
- 条件判断和循环
- 自定义函数支持

✅ **匹配条件功能**
- 三种提取器类型的匹配条件
- 条件成功和失败场景
- 无匹配条件的默认行为

✅ **字段提取功能**
- 所有支持的邮件字段
- 字段内容正确性验证
- 空字段处理

✅ **多提取器组合**
- 多个提取器同时执行
- 结果合并和去重
- 执行顺序验证

✅ **边界情况处理**
- 空数据处理
- 错误输入验证
- 异常情况恢复

## 测试结果验证

所有测试都包含以下验证：

1. **功能正确性**: 提取结果是否符合预期
2. **错误处理**: 异常情况是否正确处理
3. **性能稳定性**: 大数据量下的表现
4. **边界条件**: 极端情况的处理能力

## 总结

通过这些全面的测试，我们验证了邮件内容提取功能的：

- **可靠性**: 各种提取器都能正常工作
- **准确性**: 提取结果准确无误
- **稳定性**: 异常情况处理得当
- **扩展性**: 支持多种提取方式和字段

这些测试确保了邮件提取功能在生产环境中的可靠性和稳定性。