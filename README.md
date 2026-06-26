# Anox - 微服务治理中台

Anox是一个自用的微服务治理中台，提供服务注册中心、配置中心、日志中心等功能。

## 功能特性

### 1. 服务注册中心
- 基于WebSocket的服务注册与发现
- 自动心跳检测（15秒间隔）
- 服务实例状态实时监控
- 自动剔除失联实例

### 2. 配置中心
- JSON格式配置文件管理
- 支持全局配置和服务特定配置
- 配置热更新，WebSocket推送
- 版本号管理

### 3. 日志中心
- 异步日志收集
- 按服务/实例/日期/小时分层存储
- 支持日志检索和关键字过滤
- 多级别告警（Debug/Info/Important/Emergency）
- 告警去重机制

### 4. Web管理端
- Vue3 + Vite + Arco Design
- 概览面板（服务数、实例数、系统指标）
- 服务管理（折叠面板展示实例详情）
- 配置管理（键值对/JSON双模式编辑）
- 日志检索（多维度筛选）
- 告警配置（企业微信/钉钉/飞书/短信）

### 5. 客户端SDK
- 简化服务注册和心跳维持
- 配置拉取和监听
- 异步日志发送
- 上下文信息自动提取

## 快速开始

### 环境要求
- Go 1.21+
- Node.js 18+

### 安装依赖

```bash
# 下载所有依赖
make deps
```

### 构建

```bash
# 构建前后端
make build

# 仅构建后端
make build-server

# 仅构建前端
make build-web
```

### 运行

```bash
# 创建数据目录
make init

# 完整运行（先构建再运行）
make run

# 仅运行后端开发模式
go run ./cmd/anox-server

# 前端开发模式
cd web && npm run dev
```

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| HOST | 0.0.0.0 | 监听地址 |
| PORT | 8848 | 监听端口 |
| PASS | admin | 登录密码 |

### 访问管理端

启动后访问 `http://localhost:8848`，使用配置的密码登录。

## 目录结构

```
Anox/
├── cmd/anox-server/     # 服务端入口
├── internal/            # 内部实现
│   ├── core/           # 配置存储、热重载
│   ├── registry/       # 服务注册中心
│   ├── logcenter/      # 日志中心
│   └── server/         # HTTP/WebSocket服务
├── pkg/sdk/            # 客户端SDK
├── api/                # 公共协议定义
├── web/                # 前端项目（Vue3）
├── data/configs/       # 配置文件存储
└── logs/               # 日志文件存储
```

## 客户端SDK使用

```go
import "anox/pkg/sdk"

// 创建客户端
client, err := sdk.NewClient(sdk.Config{
    AnoxURL:     "ws://localhost:8080/ws",
    ServiceName: "user-service",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 读取配置
port := client.GetConfig("port")
logLevel := client.GetConfig("log_level")

// 发送日志
client.Log(ctx, "用户登录成功", sdk.LogLevelInfo, "login", true)

// 发送错误日志（带堆栈）
client.LogWithStack(ctx, err.Error(), sdk.LogLevelEmergency, "db_query", nil, true)
```

## 配置文件格式

```json
{
  "version": 1698345600,
  "values": {
    "port": "8080",
    "log_level": "info",
    "mysql_host": "192.168.1.100",
    "redis_addr": "192.168.1.101"
  }
}
```

## 日志格式

```json
{
  "time": "2023-10-27T14:30:00.000Z",
  "service": "user-service",
  "instance": "instance-user-xxxx",
  "level": "Emergency",
  "action": "login",
  "message": "校验用户信息发生意外...",
  "trace_id": "a1b2c3d4",
  "stacks": ["goroutine 1 [running]..."],
  "context": {
    "user_id": "10086",
    "ip": "123.45.67.89"
  }
}
```

## 开发说明

### 添加依赖

```bash
# Go依赖
go get github.com/some/package

# 前端依赖
cd web && npm install some-package
```

### 运行测试

```bash
make test
```

## 许可证

MIT License
