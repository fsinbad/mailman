# TriggerV2 部署指南

## 概述

本文档提供了 TriggerV2 系统在生产环境中的部署、配置和运维指南。

## 系统要求

### 硬件要求

#### 最低配置
- **CPU**: 2核心
- **内存**: 4GB RAM
- **磁盘**: 20GB 可用空间
- **网络**: 100Mbps

#### 推荐配置
- **CPU**: 4核心或以上
- **内存**: 8GB RAM 或以上
- **磁盘**: 100GB SSD
- **网络**: 1Gbps

#### 高负载生产环境
- **CPU**: 8核心或以上
- **内存**: 16GB RAM 或以上
- **磁盘**: 500GB SSD + 网络存储
- **网络**: 10Gbps

### 软件要求

- **操作系统**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **Go**: 1.19 或以上版本
- **数据库**: 
  - PostgreSQL 12+ (推荐)
  - MySQL 8.0+
  - MongoDB 4.4+
- **消息队列**: 
  - Redis 6.0+ (推荐)
  - RabbitMQ 3.8+
  - Apache Kafka 2.8+
- **监控**: 
  - Prometheus (可选)
  - Grafana (可选)
- **负载均衡**: 
  - Nginx 1.18+
  - HAProxy 2.0+

## 部署架构

### 单机部署

```
┌─────────────────────────────────┐
│           Load Balancer         │
│            (Nginx)              │
└─────────────────┬───────────────┘
                  │
┌─────────────────▼───────────────┐
│         TriggerV2 App           │
│    ┌───────────────────────┐    │
│    │    Event Bus          │    │
│    │    Condition Engine   │    │
│    │    Batch Processor    │    │
│    │    Monitor System     │    │
│    └───────────────────────┘    │
└─────────────────┬───────────────┘
                  │
┌─────────────────▼───────────────┐
│           Database              │
│         (PostgreSQL)            │
└─────────────────────────────────┘
```

### 分布式部署

```
┌─────────────────────────────────┐
│           Load Balancer         │
│            (Nginx)              │
└─────────────────┬───────────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
┌───▼───┐    ┌───▼───┐    ┌───▼───┐
│ App 1 │    │ App 2 │    │ App 3 │
│       │    │       │    │       │
└───┬───┘    └───┬───┘    └───┬───┘
    │            │            │
    └─────────────┼─────────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
┌───▼───┐    ┌───▼───┐    ┌───▼───┐
│ Redis │    │  DB   │    │Monitor│
│       │    │       │    │       │
└───────┘    └───────┘    └───────┘
```

### 微服务部署

```
┌─────────────────────────────────┐
│          API Gateway            │
│           (Kong)                │
└─────────────────┬───────────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
┌───▼───┐    ┌───▼───┐    ┌───▼───┐
│Event  │    │Trigger│    │Action │
│Service│    │Service│    │Service│
└───┬───┘    └───┬───┘    └───┬───┘
    │            │            │
    └─────────────┼─────────────┘
                  │
┌─────────────────▼───────────────┐
│         Shared Services         │
│   ┌─────────┐ ┌─────────────┐   │
│   │ Message │ │   Database  │   │
│   │  Queue  │ │             │   │
│   └─────────┘ └─────────────┘   │
└─────────────────────────────────┘
```

## 安装步骤

### 1. 环境准备

#### 系统更新
```bash
# Ubuntu/Debian
sudo apt update && sudo apt upgrade -y

# CentOS/RHEL
sudo yum update -y
```

#### 安装依赖
```bash
# 安装必要的工具
sudo apt install -y curl wget git build-essential

# 安装 Go
curl -fsSL https://golang.org/dl/go1.20.linux-amd64.tar.gz -o go.tar.gz
sudo tar -C /usr/local -xzf go.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 2. 数据库安装

#### PostgreSQL 安装
```bash
# Ubuntu
sudo apt install -y postgresql postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql
CREATE DATABASE triggerv2;
CREATE USER triggerv2 WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE triggerv2 TO triggerv2;
\q
```

#### Redis 安装
```bash
# Ubuntu
sudo apt install -y redis-server

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server

# 配置 Redis
sudo vim /etc/redis/redis.conf
# 设置密码
requirepass your_redis_password
```

### 3. 应用部署

#### 获取源码
```bash
# 克隆仓库
git clone https://github.com/your-org/mailman.git
cd mailman/backend

# 构建应用
go mod tidy
go build -o triggerv2 ./cmd/triggerv2
```

#### 配置文件
```bash
# 创建配置目录
sudo mkdir -p /etc/triggerv2

# 复制配置文件
sudo cp configs/triggerv2.yaml /etc/triggerv2/
```

#### 配置文件示例
```yaml
# /etc/triggerv2/config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  database: "triggerv2"
  username: "triggerv2"
  password: "your_password"
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 3600s

redis:
  host: "localhost"
  port: 6379
  password: "your_redis_password"
  database: 0
  max_idle: 10
  max_active: 100

triggerv2:
  eventbus:
    max_event_size: 1048576
    buffer_size: 1000
    worker_count: 10
    process_timeout: 30s
    enable_metrics: true
    
  batch_processor:
    max_batch_size: 100
    flush_interval: 5s
    max_retries: 3
    retry_interval: 1s
    max_concurrency: 10
    
  monitor:
    metrics_retention: 24h
    alert_cooldown: 5m
    health_check_interval: 30s
    
  plugins:
    email:
      smtp_host: "smtp.example.com"
      smtp_port: 587
      username: "user@example.com"
      password: "email_password"
      
    webhook:
      timeout: 30s
      max_retries: 3
      
    slack:
      webhook_url: "https://hooks.slack.com/services/..."

logging:
  level: "info"
  format: "json"
  file: "/var/log/triggerv2/app.log"
  max_size: 100  # MB
  max_backups: 5
  max_age: 30    # days
  compress: true

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

#### 创建服务用户
```bash
# 创建系统用户
sudo useradd -r -s /bin/false triggerv2

# 创建必要的目录
sudo mkdir -p /var/log/triggerv2
sudo mkdir -p /var/lib/triggerv2
sudo chown triggerv2:triggerv2 /var/log/triggerv2
sudo chown triggerv2:triggerv2 /var/lib/triggerv2
```

#### 创建 systemd 服务
```bash
# 创建服务文件
sudo vim /etc/systemd/system/triggerv2.service
```

```ini
[Unit]
Description=TriggerV2 Event Processing Service
After=network.target postgresql.service redis.service
Requires=postgresql.service redis.service

[Service]
Type=simple
User=triggerv2
Group=triggerv2
WorkingDirectory=/opt/triggerv2
ExecStart=/opt/triggerv2/triggerv2 -config /etc/triggerv2/config.yaml
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
KillSignal=SIGINT
TimeoutStopSec=30
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/triggerv2 /var/lib/triggerv2

[Install]
WantedBy=multi-user.target
```

#### 安装和启动服务
```bash
# 复制二进制文件
sudo mkdir -p /opt/triggerv2
sudo cp triggerv2 /opt/triggerv2/
sudo chown triggerv2:triggerv2 /opt/triggerv2/triggerv2
sudo chmod +x /opt/triggerv2/triggerv2

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable triggerv2
sudo systemctl start triggerv2

# 检查状态
sudo systemctl status triggerv2
```

## 配置管理

### 环境变量配置

```bash
# 数据库配置
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=triggerv2
export DB_USER=triggerv2
export DB_PASSWORD=your_password

# Redis 配置
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=your_redis_password

# 应用配置
export TRIGGERV2_LOG_LEVEL=info
export TRIGGERV2_METRICS_ENABLED=true
export TRIGGERV2_EVENTBUS_WORKER_COUNT=10
```

### 配置文件模板

#### 开发环境配置
```yaml
# config-dev.yaml
server:
  host: "127.0.0.1"
  port: 8080

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  database: "triggerv2_dev"

triggerv2:
  eventbus:
    worker_count: 2
    enable_metrics: false
    
logging:
  level: "debug"
  format: "console"
```

#### 生产环境配置
```yaml
# config-prod.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  driver: "postgres"
  host: "db.example.com"
  port: 5432
  database: "triggerv2"
  max_open_conns: 100
  max_idle_conns: 10

triggerv2:
  eventbus:
    worker_count: 20
    enable_metrics: true
    
logging:
  level: "info"
  format: "json"
  file: "/var/log/triggerv2/app.log"
```

## 负载均衡配置

### Nginx 配置

```nginx
# /etc/nginx/sites-available/triggerv2
upstream triggerv2_backend {
    least_conn;
    server 127.0.0.1:8080 weight=1 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8081 weight=1 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8082 weight=1 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name triggerv2.example.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name triggerv2.example.com;
    
    # SSL 证书配置
    ssl_certificate /etc/ssl/certs/triggerv2.crt;
    ssl_certificate_key /etc/ssl/private/triggerv2.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # 安全头
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
    
    # 日志配置
    access_log /var/log/nginx/triggerv2-access.log;
    error_log /var/log/nginx/triggerv2-error.log;
    
    # 通用配置
    client_max_body_size 10M;
    client_body_timeout 60s;
    client_header_timeout 60s;
    
    # API 路由
    location /api/ {
        proxy_pass http://triggerv2_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # 健康检查
        proxy_next_upstream error timeout invalid_header http_500 http_502 http_503 http_504;
        proxy_next_upstream_tries 3;
    }
    
    # 健康检查端点
    location /health {
        proxy_pass http://triggerv2_backend;
        access_log off;
    }
    
    # 指标端点
    location /metrics {
        proxy_pass http://triggerv2_backend;
        allow 127.0.0.1;
        allow 10.0.0.0/8;
        deny all;
    }
    
    # 静态文件
    location /static/ {
        alias /var/www/triggerv2/static/;
        expires 1d;
        add_header Cache-Control "public, immutable";
    }
}
```

### HAProxy 配置

```
# /etc/haproxy/haproxy.cfg
global
    daemon
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    
    # SSL 配置
    tune.ssl.default-dh-param 2048
    ssl-default-bind-options no-sslv3 no-tlsv10 no-tlsv11

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms
    option httplog
    option dontlognull
    option http-server-close
    option forwardfor except 127.0.0.0/8
    option redispatch
    retries 3
    
    # 健康检查
    option httpchk GET /health
    http-check expect status 200

frontend triggerv2_frontend
    bind *:80
    bind *:443 ssl crt /etc/ssl/certs/triggerv2.pem
    redirect scheme https if !{ ssl_fc }
    
    # 访问控制
    acl is_api path_beg /api/
    acl is_metrics path_beg /metrics
    acl allowed_networks src 10.0.0.0/8 192.168.0.0/16
    
    # 路由规则
    use_backend triggerv2_backend if is_api
    use_backend triggerv2_backend if is_metrics allowed_networks
    
    # 默认后端
    default_backend triggerv2_backend

backend triggerv2_backend
    balance roundrobin
    option httpchk GET /health
    
    # 服务器定义
    server triggerv2-1 127.0.0.1:8080 check inter 5s rise 2 fall 3
    server triggerv2-2 127.0.0.1:8081 check inter 5s rise 2 fall 3
    server triggerv2-3 127.0.0.1:8082 check inter 5s rise 2 fall 3

# 统计页面
listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 5s
    stats admin if TRUE
```

## 监控配置

### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "triggerv2_rules.yml"

scrape_configs:
  - job_name: 'triggerv2'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 30s
    metrics_path: '/metrics'
    
  - job_name: 'triggerv2-nodes'
    static_configs:
      - targets: 
        - 'node1.example.com:9090'
        - 'node2.example.com:9090'
        - 'node3.example.com:9090'

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Grafana 仪表盘

```json
{
  "dashboard": {
    "title": "TriggerV2 监控仪表盘",
    "panels": [
      {
        "title": "事件处理速率",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(triggerv2_events_processed_total[5m])",
            "legendFormat": "Events/sec"
          }
        ]
      },
      {
        "title": "系统资源使用",
        "type": "graph",
        "targets": [
          {
            "expr": "process_resident_memory_bytes",
            "legendFormat": "Memory Usage"
          },
          {
            "expr": "rate(process_cpu_seconds_total[5m])",
            "legendFormat": "CPU Usage"
          }
        ]
      }
    ]
  }
}
```

## 安全配置

### 防火墙配置

```bash
# UFW 配置
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow from 10.0.0.0/8 to any port 9090  # Prometheus metrics
sudo ufw enable

# iptables 配置
sudo iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -s 10.0.0.0/8 -p tcp --dport 9090 -j ACCEPT
sudo iptables -A INPUT -j DROP
```

### SSL/TLS 配置

```bash
# 生成自签名证书（开发环境）
openssl req -x509 -newkey rsa:4096 -keyout triggerv2.key -out triggerv2.crt -days 365 -nodes

# 使用 Let's Encrypt （生产环境）
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d triggerv2.example.com
```

### 访问控制

```yaml
# 在配置文件中添加访问控制
security:
  api_key_required: true
  allowed_origins:
    - "https://admin.example.com"
    - "https://dashboard.example.com"
  rate_limits:
    - path: "/api/events"
      limit: 1000
      window: "1m"
    - path: "/api/triggers"
      limit: 100
      window: "1m"
```

## 容器化部署

### Dockerfile

```dockerfile
# Dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o triggerv2 ./cmd/triggerv2

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/triggerv2 .
COPY --from=builder /app/configs ./configs

EXPOSE 8080 9090

CMD ["./triggerv2", "-config", "configs/config.yaml"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  triggerv2:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    depends_on:
      - postgres
      - redis
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    volumes:
      - ./configs:/app/configs
      - ./logs:/var/log/triggerv2
    restart: unless-stopped

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: triggerv2
      POSTGRES_USER: triggerv2
      POSTGRES_PASSWORD: password123
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:6-alpine
    command: redis-server --requirepass password123
    volumes:
      - redis_data:/data
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
    depends_on:
      - triggerv2
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Kubernetes 部署

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: triggerv2
  labels:
    app: triggerv2
spec:
  replicas: 3
  selector:
    matchLabels:
      app: triggerv2
  template:
    metadata:
      labels:
        app: triggerv2
    spec:
      containers:
      - name: triggerv2
        image: triggerv2:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: DB_HOST
          value: "postgres-service"
        - name: REDIS_HOST
          value: "redis-service"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: triggerv2-service
spec:
  selector:
    app: triggerv2
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer
```

## 备份和恢复

### 数据库备份

```bash
#!/bin/bash
# backup-db.sh

DB_NAME="triggerv2"
DB_USER="triggerv2"
BACKUP_DIR="/var/backups/triggerv2"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
pg_dump -U $DB_USER -h localhost $DB_NAME | gzip > $BACKUP_DIR/triggerv2_$DATE.sql.gz

# 保留最近 7 天的备份
find $BACKUP_DIR -name "triggerv2_*.sql.gz" -mtime +7 -delete

echo "Database backup completed: triggerv2_$DATE.sql.gz"
```

### 配置备份

```bash
#!/bin/bash
# backup-config.sh

CONFIG_DIR="/etc/triggerv2"
BACKUP_DIR="/var/backups/triggerv2"
DATE=$(date +%Y%m%d_%H%M%S)

# 备份配置文件
tar -czf $BACKUP_DIR/config_$DATE.tar.gz -C $CONFIG_DIR .

echo "Configuration backup completed: config_$DATE.tar.gz"
```

### 恢复脚本

```bash
#!/bin/bash
# restore-db.sh

if [ -z "$1" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

DB_NAME="triggerv2"
DB_USER="triggerv2"
BACKUP_FILE="$1"

# 恢复数据库
gunzip -c $BACKUP_FILE | psql -U $DB_USER -h localhost $DB_NAME

echo "Database restored from: $BACKUP_FILE"
```

## 性能调优

### 系统级优化

```bash
# 内核参数优化
echo 'net.core.somaxconn = 1024' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 1024' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_fin_timeout = 30' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_keepalive_time = 1200' >> /etc/sysctl.conf
echo 'fs.file-max = 65536' >> /etc/sysctl.conf

# 应用系统参数
sysctl -p
```

### 数据库优化

```sql
-- PostgreSQL 优化
-- postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.7
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
```

### 应用优化

```yaml
# 应用配置优化
triggerv2:
  eventbus:
    buffer_size: 2000        # 增加缓冲区大小
    worker_count: 20         # 增加工作线程数
    batch_size: 100          # 批处理大小
    
  database:
    max_open_conns: 100      # 最大连接数
    max_idle_conns: 20       # 最大空闲连接数
    conn_max_lifetime: 300s  # 连接最大生存时间
    
  redis:
    max_idle: 20             # 最大空闲连接数
    max_active: 100          # 最大活跃连接数
    idle_timeout: 300s       # 空闲超时时间
```

## 运维脚本

### 启动脚本

```bash
#!/bin/bash
# start-triggerv2.sh

set -e

CONFIG_FILE="/etc/triggerv2/config.yaml"
PID_FILE="/var/run/triggerv2.pid"
LOG_FILE="/var/log/triggerv2/startup.log"

# 检查配置文件
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Configuration file not found: $CONFIG_FILE"
    exit 1
fi

# 检查是否已经运行
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null; then
        echo "TriggerV2 is already running (PID: $PID)"
        exit 1
    fi
fi

# 启动应用
echo "Starting TriggerV2..."
nohup /opt/triggerv2/triggerv2 -config "$CONFIG_FILE" > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

echo "TriggerV2 started successfully"
```

### 监控脚本

```bash
#!/bin/bash
# monitor-triggerv2.sh

PID_FILE="/var/run/triggerv2.pid"
LOG_FILE="/var/log/triggerv2/monitor.log"

check_process() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null; then
            return 0
        fi
    fi
    return 1
}

check_health() {
    curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health
}

# 检查进程
if ! check_process; then
    echo "$(date): TriggerV2 process not running" >> "$LOG_FILE"
    exit 1
fi

# 检查健康状态
HTTP_CODE=$(check_health)
if [ "$HTTP_CODE" != "200" ]; then
    echo "$(date): TriggerV2 health check failed (HTTP $HTTP_CODE)" >> "$LOG_FILE"
    exit 1
fi

echo "$(date): TriggerV2 is healthy" >> "$LOG_FILE"
```

## 故障排除

### 常见问题

1. **应用无法启动**
   - 检查配置文件语法
   - 确认数据库连接
   - 检查端口占用情况
   - 查看启动日志

2. **数据库连接失败**
   - 检查数据库服务状态
   - 确认连接参数
   - 检查防火墙设置
   - 验证用户权限

3. **高内存使用**
   - 检查事件积压情况
   - 调整批处理配置
   - 优化数据库查询
   - 检查内存泄漏

4. **性能问题**
   - 检查系统资源使用
   - 优化数据库索引
   - 调整工作线程数
   - 使用连接池

### 日志分析

```bash
# 查看错误日志
tail -f /var/log/triggerv2/app.log | grep ERROR

# 分析访问日志
awk '{print $1}' /var/log/nginx/triggerv2-access.log | sort | uniq -c | sort -nr

# 监控系统资源
top -p $(pgrep triggerv2)
```

---

*本部署指南基于 TriggerV2 v2.0.0 版本，最后更新于：2024年*