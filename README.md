# 📧 Mailman - 智能邮件管理系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Image](https://img.shields.io/docker/pulls/seongminhwan/mailman-all.svg)](https://hub.docker.com/r/seongminhwan/mailman-all)
[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-14.0+-black.svg)](https://nextjs.org)

Mailman 是一个现代化的智能邮件管理系统，提供邮件同步、智能解析、OAuth2认证和AI集成等功能。

## 🚀 快速开始

### 🔥 Docker 一键部署（推荐）

```bash
docker run -d \
  --name mailman \
  -p 8080:80 \
  -v mailman_data:/app/data \
  --restart unless-stopped \
  ghcr.io/seongminhwan/mailman-all:latest
```

然后访问���http://localhost:8080

默认登录：`admin` / `admin123`

### 🏗️ 多架构支持

支持多种处理器架构，Docker 自动选择：

| 架构 | 平台 | 说明 |
|------|------|------|
| `linux/amd64` | Intel/AMD | 台式机、服务器 |
| `linux/arm64` | ARM 64-bit | Apple Silicon (M1/M2/M3) |
| `linux/arm/v7` | ARM 32-bit | 树莓派、ARM 设备 |

**树莓派部署**：
```bash
docker run -d \
  --name mailman \
  -p 8080:80 \
  -v mailman_data:/app/data \
  --platform linux/arm/v7 \
  ghcr.io/seongminhwan/mailman-all:latest
```

## 📋 主要特性

### 📧 邮件管理
- **多账户支持**：Gmail、Outlook、IMAP等
- **OAuth2认证**：安全的第三方认证
- **实时同步**：自动邮件同步
- **智能搜索**：全局邮件搜索
- **邮件解析**：自动提取关键信息

### 🤖 AI集成
- **多AI支持**：OpenAI、Claude、Gemini
- **智能提取**：AI驱动的邮件内容分析
- **模板生成**：自动生成处理模板
- **可视化配置**：拖拽式AI配置界面

### 🛡️ 安全特性
- **多重认证**：用户名密码、OAuth2
- **数据加密**：敏感数据加密存储
- **权限控制**：细粒度权限管理
- **安全传输**：HTTPS/WSS加密通信

## 📖 安装方式

| 安装方式 | 复杂度 | 时间 | 适用场景 |
|---------|--------|------|----------|
| **Docker 一键部署** | ⭐ | 2分钟 | 新手体验 |
| **K3s 轻量级集群** | ⭐⭐ | 5分钟 | 边缘计算、IoT |
| **Docker Compose** | ⭐⭐ | 5分钟 | 生产环境 |
| **Kubernetes 集群** | ⭐⭐⭐ | 10分钟 | 大规模部署 |
| **源码开发** | ⭐⭐⭐⭐ | 15分钟 | 开发调试 |

📖 **详细安装指南**：
- [`INSTALL.md`](./INSTALL.md) - Docker 和本地开发
- [`K3S_QUICKSTART.md`](./K3S_QUICKSTART.md) - K3s 快速部署
- [`HELM_DEPLOYMENT.md`](./HELM_DEPLOYMENT.md) - Kubernetes 集群部署

## 🌐 访问地址

- **前端界面**：http://localhost:8080
- **API文档**：http://localhost:8080/swagger/index.html

## ⚙️ 核心功能

### 邮件账户管理
```bash
# 支持的邮件服务
✅ Gmail (OAuth2)
✅ Outlook (OAuth2)
✅ 通用 IMAP/SMTP
✅ Exchange Server
```

### OAuth2 配置
```bash
# Gmail OAuth2
1. 访问 Google Cloud Console
2. 启用 Gmail API
3. 创建 OAuth2 客户端
4. 配置重定向 URI: http://localhost:8080/api/oauth2/callback/gmail

# Outlook OAuth2
1. 访问 Azure Portal
2. 注册应用程序
3. 添加 API 权限
4. 配置重定向 URI: http://localhost:8080/api/oauth2/callback/outlook
```

### AI 服务集成
```bash
# 支持的 AI 提供商
✅ OpenAI (GPT-3.5, GPT-4)
✅ Claude (Anthropic)
✅ Gemini (Google)

# 配置方式
1. 登录系统
2. 进入"设置" → "AI配置"
3. 添加 AI 服务商配置
4. 输入 API 密钥并测试
```

## 🏗️ 技术架构

### 后端技术栈
- **语言**：Go 1.23+
- **框架**：Gorilla Mux
- **数据库**：MySQL 8.0 / SQLite
- **ORM**：GORM
- **实时通信**：WebSocket

### 前端技术栈
- **框架**：Next.js 14 (App Router)
- **语言**：TypeScript 5.3+
- **样式**：Tailwind CSS
- **状态管理**：Zustand
- **UI组件**：Radix UI

### 部署技术
- **容器化**：Docker + Docker Compose
- **镜像仓库**：GitHub Container Registry
- **反向代理**：Nginx

## 📚 API 文档

### 主要端点

```bash
# 认证
POST   /api/auth/login          # 用户登录
POST   /api/auth/logout         # 用户登出
GET    /api/auth/me            # 获取用户信息

# 邮件账户
GET    /api/accounts           # 获取账户列表
POST   /api/accounts           # 添加账户
PUT    /api/accounts/{id}      # 更新账户
DELETE /api/accounts/{id}      # 删除账户

# 邮件管理
GET    /api/emails             # 获取邮件列表
GET    /api/emails/{id}        # 获取邮件详情
POST   /api/emails/sync        # 同步邮件
POST   /api/emails/search      # 搜索邮件

# OAuth2
GET    /api/oauth2/auth-url/{provider}  # 获取授权URL
POST   /api/oauth2/exchange-token       # 交换令牌
POST   /api/oauth2/refresh-token        # 刷新令牌

# AI 功能
GET    /api/ai/config          # 获取AI配置
POST   /api/ai/config          # 创建AI配置
POST   /api/ai/extract         # AI内容提取
```

## 🛠️ 开发环境

### 前置要求
- Go 1.23+
- Node.js 18+
- MySQL 8.0+ 或 SQLite
- Docker（可选）

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/seongminhwan/mailman.git
cd mailman

# 2. 启动后端
cd backend
go mod download
go run cmd/mailman/main.go

# 3. 启动前端（新终端）
cd frontend
npm install
npm run dev
```

### 访问开发环境
- **前端**：http://localhost:3000
- **后端**：http://localhost:8080
- **API文档**：http://localhost:8080/swagger/index.html

## 📋 项目结构

```
mailman/
├── backend/                   # Go 后端服务
│   ├── cmd/mailman/          # 应用入口
│   ├── internal/             # 内部包
│   │   ├── api/             # API 层
│   │   ├── models/          # 数据模型
│   │   ├── services/        # 业务逻辑
│   │   └── repository/      # 数据访问
│   └── Dockerfile           # 后端镜像
├── frontend/                  # Next.js 前端
│   ├── src/                 # 源代码
│   │   ├── app/             # 页面组件
│   │   ├── components/      # UI 组件
│   │   └── services/        # API 服务
│   ├── Dockerfile.nginx     # 前端镜像
│   └── package.json
├── docs/                     # 文档
├── helm/                     # Kubernetes Helm Chart
├── docker-compose.yml        # Docker Compose 配置
└── README.md                 # 项目说明
```

## 🚀 部署选项

### 1. Docker 一键部署
```bash
docker run -d \
  --name mailman \
  -p 8080:80 \
  -v mailman_data:/app/data \
  ghcr.io/seongminhwan/mailman-all:latest
```

### 2. Docker Compose
```bash
git clone https://github.com/seongminhwan/mailman.git
cd mailman
docker-compose up -d
```

### 3. K3s 轻量级 Kubernetes（推荐）
```bash
# 一键部署
./deploy-k3s.sh

# 或手动部署
helm install mailman ./helm/mailman \
  --values ./helm/mailman/values-matrixdb-production.yaml \
  --values ./helm/mailman/values-k3s.yaml
```

📖 **K3s 部署指南**：
- 🚀 [快速开始](./K3S_QUICKSTART.md) - 5分钟快速部署
- 📖 [完整文档](./K3S_DEPLOYMENT.md) - 详细配置说明

### 4. 标准 Kubernetes
```bash
cd helm/mailman
./deploy.sh -e production -t standard
```

📖 **Kubernetes 部署指南**：[`HELM_DEPLOYMENT.md`](./HELM_DEPLOYMENT.md)

## 🔧 配置说明

### 环境变量

```bash
# 数据库配置
DB_DRIVER=sqlite              # 或 mysql
DB_NAME=./mailman.db         # SQLite 文件路径
# DB_HOST=localhost           # MySQL 主机
# DB_PORT=3306               # MySQL 端口
# DB_USER=mailman            # MySQL 用户名
# DB_PASSWORD=password        # MySQL 密码

# 服务器配置
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=INFO
```

### OAuth2 配置

系统支持通过 Web 界面配置 OAuth2，支持：
- Gmail OAuth2
- Outlook OAuth2
- 自定义 OAuth2 提供商

## 📈 监控和日志

### 健康检查
```bash
# 检查服务状态
curl http://localhost:8080/health

# 查看系统信息
curl http://localhost:8080/api/system/info
```

### 日志管理
```bash
# Docker 部署日志
docker logs -f mailman

# Docker Compose 日志
docker-compose logs -f backend
```

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🆘 获取帮助

- 📖 **安装指南**：[`INSTALL.md`](./INSTALL.md)
- 🐛 **问题报告**：[GitHub Issues](https://github.com/seongminhwan/mailman/issues)
- 📧 **联系邮箱**：support@mailman.dev

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者和用户！

---

**Mailman** - 让邮件管理更智能、更高效！ 🚀