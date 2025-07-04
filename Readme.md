# Sunny 框架使用指南

## 项目简介

Sunny 是一个基于 Gin Web 框架的 Golang 微服务框架，提供了角色-组-动作的三层架构模式，内置缓存、数据库、消息队列、JWT认证等组件支持。

## 核心架构

### 三层架构模式
- **角色 (Role)**: 定义访问权限和执行环境
- **组 (Group)**: 业务功能模块分组  
- **动作 (Action)**: 具体的业务处理逻辑

### 路由格式
```
/{path}/{group}/{action}
```

## 快速开始

### 1. 项目初始化

```go
package main

import (
    "context"
    "github.com/hfup/sunny"
)

func main() {
    app := sunny.GetApp()
      
    // 启动应用
    err = app.Start(context.Background())
    if err != nil {
        panic(err)
    }
}
```

### 2. 配置文件设置

创建配置目录 `resources/` 并添加配置文件：

#### application.yaml (基础配置)
```yaml
active_env: "dev"

services:
  - protocol: "http"
    port: 8080
    
web_routes:
  - path: "/api"
    role: "api_role"
    role_desc: "API服务角色"
```

#### application-dev.yaml (开发环境配置)
```yaml
redis:
  is_cluster: false
  addrs: ["localhost:6379"]
  password: ""
  db: 0
  pool_size: 10

database_client_manager:
  is_debug: 1
  dbs:
    - db_id: "main"
      database_info:
        driver: "mysql"
        host: "localhost"
        port: 3306
        user: "root"
        password: "password"
        db_name: "test"
        charset: "utf8mb4"
```

### 3. 业务逻辑开发

#### 创建角色处理器
```go
// 注册角色前置处理器
app.RegisterRoleBeforeHandler("api_role", 1, func(c *sunny.Context) {
    // 权限验证、请求日志等
    fmt.Println("API角色前置处理")
})

// 注册角色后置处理器  
app.RegisterRoleAfterHandler("api_role", 1, func(c *sunny.Context) {
    // 响应日志、清理资源等
    fmt.Println("API角色后置处理")
})
```

#### 创建业务组
```go
// 创建用户管理组
userGroup := sunny.NewGroup("api_role", "user", "用户管理")

// 绑定具体动作
userGroup.BindAction("login", func(c *sunny.Context) {
    // 用户登录逻辑
    username := c.PostForm("username")
    password := c.PostForm("password")
    
    // 业务处理...
    
    c.JsonResult(0, "登录成功", gin.H{
        "token": "xxx",
        "user_id": 123,
    })
}, "用户登录")

userGroup.BindAction("info", func(c *sunny.Context) {
    // 获取用户信息
    c.JsonResult(0, "获取成功", gin.H{
        "id": 123,
        "name": "张三",
    })
}, "获取用户信息")

// 注册组到应用
app.RegisterGroup(userGroup)
```

### 4. 数据库操作

```go
userGroup.BindAction("create", func(c *sunny.Context) {
    // 获取数据库连接
    db, err := c.GetDB()
    if err != nil {
        c.JsonResult(-1, "数据库连接失败")
        return
    }
    
    // 使用 GORM 操作数据库
    var user User
    db.Where("id = ?", 1).First(&user)
    
    c.JsonResult(0, "查询成功", user)
})
```

### 5. Redis 缓存操作

```go
userGroup.BindAction("cache", func(c *sunny.Context) {
    // 获取 Redis 客户端
    rdb, err := c.GetRedisClient()
    if err != nil {
        c.JsonResult(-1, "Redis连接失败")
        return
    }
    
    // 设置缓存
    err = rdb.Set(context.Background(), "key", "value", time.Hour).Err()
    if err != nil {
        c.JsonResult(-1, "缓存设置失败")
        return
    }
    
    c.JsonResult(0, "缓存设置成功")
})
```

## 高级功能

### 1. 消息队列支持

#### RabbitMQ 配置
```yaml
mq:
  type: "rabbitmq"
  rabbitmq:
    host: "localhost"
    port: 5672
    username: "guest"
    password: "guest"
    max_retries: 3
    retry_interval: 5
    channel_pool_size: 10
```

#### Kafka 配置
```yaml
mq:
  type: "kafka"
  kafka:
    brokers: ["localhost:9092"]
    max_retries: 3
    retry_interval: 5
    security_protocol: "PLAINTEXT"
```

### 3. gRPC 服务支持

```go
// 注册 gRPC 服务
app.BindGrpcServices(yourGrpcService)

// 设置 gRPC 拦截器
app.SetGrpcInterceptorHandler(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // 拦截器逻辑
    return handler(ctx, req)
})
```

### 4. 子服务管理

```go
// 添加子服务
app.AddSubServices(customService)

// 添加同步执行任务
app.AddSyncRunAbles(syncTask)

// 添加异步执行任务  
app.AddAsyncRunAbles(asyncTask)
```

## API 请求示例

### 用户登录
```bash
POST http://localhost:8080/api/user/login
Content-Type: application/x-www-form-urlencoded

username=admin&password=123456
```

### 获取用户信息
```bash
GET http://localhost:8080/api/user/info
```

## 响应格式

所有 API 响应统一格式：
```json
{
  "err_code": 0,
  "message": "操作成功", 
  "data": {
    // 具体数据
  }
}
```

- `err_code`: 错误码，0表示成功，非0表示失败
- `message`: 响应消息
- `data`: 响应数据（可选）


## 最佳实践

1. **配置管理**: 使用不同环境的配置文件，避免硬编码
2. **错误处理**: 统一错误码和错误信息格式
3. **日志记录**: 在角色前后置处理器中添加日志
4. **数据验证**: 在动作处理器中进行参数验证
5. **资源管理**: 及时释放数据库连接等资源
6. **安全性**: 做好输入验证和权限控制

## 注意事项

1. 确保配置文件路径正确，默认为 `./resources/`
2. 数据库连接信息需要根据实际环境配置
3. Redis 配置支持单机和集群模式
4. 消息队列需要提前部署相应的服务
5. 路由注册顺序会影响匹配规则

通过以上指南，你可以快速上手 Sunny 框架并构建高效的微服务应用。