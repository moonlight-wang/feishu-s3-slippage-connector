# S3 Rest API

基于 Go 语言实现的 S3 日志查看服务，提供 Lark（飞书）多维表格自定义连接器接口。

## 功能特性

- ✅ 从 S3 存储读取滑点交易日志
- ✅ 支持 Lark 多维表格自定义连接器
- ✅ CSV 格式日志文件解析
- ✅ 分页读取大量数据
- ✅ 日期过滤
- ✅ 自动化测试
- ✅ GitHub Actions CI/CD

## 项目结构

```
s3-rest/
├── cmd/
│   └── server/
│       ├── main.go          # 主程序入口
│       └── main_test.go     # 单元测试
├── pkg/
│   ├── auth/
│   │   ├── middleware.go    # Lark 认证中间件
│   │   └── types.go         # 数据类型定义
│   ├── config/
│   │   └── config.go        # 配置管理
│   └── s3/
│       └── service.go       # S3 服务封装
├── .env                     # 环境变量配置
├── .github/
│   └── workflows/
│       └── build.yml        # GitHub Actions 配置
└── README.md
```

## 快速开始

### 1. 配置环境变量

复制 `.env` 文件并修改配置：

```env
SERVER_PORT=8080
S3_ENDPOINT=your-s3-endpoint
S3_REGION=us-east-1
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
LARK_VERIFICATION_TOKEN=your-lark-token
```

### 2. 运行服务

```bash
# 开发模式
go run cmd/server/main.go

# 生产模式
go build -o s3-rest ./cmd/server
./s3-rest
```

### 3. 测试

```bash
go test ./... -v
```

## API 接口

### 健康检查

```http
GET /health
```

响应：

```json
{
  "code": 0,
  "message": "ok"
}
```

**使用 curl 测试：**

```bash
curl -X GET http://localhost:8080/health
```

### Lark 连接器接口

```http
POST /connector
Content-Type: application/json
```

**认证方式**：Token 通过请求体中的 `verification_token` 字段传递

#### get_meta - 获取元数据

请求：

```json
{
  "action": "get_meta"
}
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "name": "S3 Slippage Trading Data",
    "description": "Slippage trading records synchronized from S3 storage",
    "fields": [
      {"name": "time", "type": "text", "required": true},
      {"name": "login", "type": "text", "required": true},
      {"name": "symbol", "type": "text", "required": true},
      {"name": "b/s", "type": "text", "required": true},
      {"name": "lot", "type": "number", "required": true},
      {"name": "req price", "type": "number", "required": true},
      {"name": "old price", "type": "number", "required": false},
      {"name": "new price", "type": "number", "required": true},
      {"name": "price diff", "type": "number", "required": false},
      {"name": "slip", "type": "number", "required": false},
      {"name": "action", "type": "text", "required": false},
      {"name": "ccy", "type": "text", "required": false},
      {"name": "pl", "type": "number", "required": false},
      {"name": "order", "type": "text", "required": false},
      {"name": "slip from req", "type": "number", "required": false},
      {"name": "pl from req", "type": "number", "required": false},
      {"name": "profile", "type": "text", "required": false},
      {"name": "spread", "type": "number", "required": false},
      {"name": "tick count", "type": "number", "required": false}
    ]
  }
}
```

**使用 curl 测试：**

```bash
curl -X POST http://localhost:8080/connector \
  -H "Content-Type: application/json" \
  -d '{
    "action": "get_meta",
    "verification_token": "your-lark-token"
  }'
```

#### read_data - 读取数据

请求：

```json
{
  "action": "read_data",
  "params": {
    "page_size": 100,
    "page_token": "0",
    "filter": {
      "date": "2026-03-23"
    }
  }
}
```

响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "records": [
      {
        "id": "record_0",
        "fields": {
          "time": "2026.03.23 01:01:29",
          "login": "390009",
          "symbol": "XAUUSD",
          "b/s": "BUY",
          "lot": 1.50,
          "req price": 4371.50000,
          "old price": 4371.50000,
          "new price": 4371.54000,
          "price diff": 0.04000,
          "slip": 4.00,
          "action": "Trade",
          "ccy": "USD",
          "pl": 6.00,
          "order": "571539",
          "slip from req": 4.00,
          "pl from req": 6.00,
          "profile": "Profile A- Gold Market",
          "spread": 26.00,
          "tick count": 522.00,
          "date": "2026-03-23",
          "sync_time": "2026-03-26 11:45:00"
        }
      }
    ],
    "has_more": true,
    "page_token": "100"
  }
}
```

**使用 curl 测试：**

```bash
# 读取所有数据（第一页）
curl -X POST http://localhost:8080/connector \
  -H "Content-Type: application/json" \
  -d '{
    "action": "read_data",
    "verification_token": "your-lark-token",
    "params": {
      "page_size": 100,
      "page_token": "0"
    }
  }'

# 带日期过滤
curl -X POST http://localhost:8080/connector \
  -H "Content-Type: application/json" \
  -d '{
    "action": "read_data",
    "verification_token": "your-lark-token",
    "params": {
      "page_size": 50,
      "page_token": "0",
      "filter": {
        "date": "2026-03-23"
      }
    }
  }'

# 读取第二页（使用上一页返回的 page_token）
curl -X POST http://localhost:8080/connector \
  -H "Content-Type: application/json" \
  -d '{
    "action": "read_data",
    "verification_token": "your-lark-token",
    "params": {
      "page_size": 100,
      "page_token": "100"
    }
  }'
```

## 日志文件格式

支持 `slippage-YYYYMMDD.txt` 格式的 CSV 文件：

```csv
time,login,symbol,b/s,lot,req price,old price,new price,price diff,slip,action,ccy,pl,order,slip from req,pl from req,profile,spread,tick count
2026.03.23 01:01:29,'390009',XAUUSD,BUY,1.50,4371.50000,4371.50000,4371.54000,0.04000,4.00,Trade,USD,6.00,571539,4.00,6.00,Profile A- Gold Market,26.00,522.00
```

## CI/CD

项目使用 GitHub Actions 进行持续集成和部署：

- **测试**: 每次推送时自动运行单元测试
- **构建**: 为 Windows 和 Linux 平台编译 64 位可执行文件
- **发布**: 创建标签时自动生成 Release 并上传构建产物

### 构建流程

1. 推送代码到 `main` 分支或创建 PR 时：

   - 运行测试
   - 编译 Windows amd64 版本
   - 编译 Linux amd64 版本
2. 创建版本标签（如 `v1.0.0`）时：

   - 运行所有测试
   - 编译两个平台的可执行文件
   - 创建 GitHub Release 并上传构建产物

### 下载构建产物

- 开发版本：在 GitHub Actions 页面下载 Artifacts
- 发布版本：在 Releases 页面下载

## 技术栈

- **语言**: Go 1.21+
- **Web 框架**: Gin
- **S3 SDK**: AWS SDK for Go v2
- **测试**: testify
- **CI/CD**: GitHub Actions

## 开发

### 添加新的日志格式

1. 修改 `pkg/auth/types.go` 中的字段定义
2. 更新 `cmd/server/main.go` 中的 `handleGetMeta` 函数
3. 调整 `parseSlippageLine` 函数的解析逻辑

### 运行测试

```bash
go test ./... -v -cover
```

## 许可证

MIT License

By Moon
