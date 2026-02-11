# ginflux

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

ginflux 是一个现代化的 InfluxDB 2.x 客户端封装库，基于官方 `github.com/influxdata/influxdb-client-go/v2` 构建，提供简洁、高性能、易用的 API。

## 特性

- ✅ **全面封装** - 完整支持 InfluxDB 2.x 所有核心功能
- ✅ **现代化设计** - 链式调用、流畅的 API 设计
- ✅ **高性能** - 支持批量写入、异步写入、连接池管理
- ✅ **易于使用** - 简洁的 API，丰富的示例代码
- ✅ **类型安全** - 完整的类型定义和错误处理
- ✅ **生产就绪** - 包含重试机制、超时控制、错误处理
- ✅ **指标收集** - 内置指标收集器，方便监控和统计
- ✅ **查询构建器** - Flux 查询构建器，无需手写复杂查询语句
- ✅ **Bucket 管理** - 完整的 Bucket 生命周期管理

## 安装

```bash
go get github.com/nicexiaonie/ginflux
```

## 快速开始

### 基础使用

```go
package main

import (
    "context"
    "log"

    "github.com/nicexiaonie/ginflux"
)

func main() {
    // 创建配置
    config := ginflux.NewDefaultConfig(
        "http://localhost:8086",
        "your-token",
        "your-org",
        "your-bucket",
    )

    // 创建客户端
    client, err := ginflux.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 写入数据
    point := ginflux.NewPoint("temperature").
        AddTag("location", "room1").
        AddField("value", 23.5).
        Build()

    ctx := context.Background()
    err = client.WriteBlocking(ctx, point)
    if err != nil {
        log.Fatal(err)
    }

    // 查询数据
    query := ginflux.NewQueryBuilder(config.Bucket).
        Measurement("temperature").
        Start("-1h").
        FilterTag("location", "room1").
        Build()

    result, err := client.Query(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    for result.Next() {
        log.Printf("Value: %v", result.Record().Value())
    }
}
```

### 使用全局默认客户端

```go
package main

import (
    "context"
    "log"

    "github.com/nicexiaonie/ginflux"
)

func main() {
    // 连接并设置为默认客户端
    config := ginflux.NewDefaultConfig(
        "http://localhost:8086",
        "your-token",
        "your-org",
        "your-bucket",
    )

    err := ginflux.Connect(config)
    if err != nil {
        log.Fatal(err)
    }
    defer ginflux.Close()

    // 使用全局函数
    point := ginflux.NewPoint("cpu").
        AddTag("host", "server01").
        AddField("usage", 75.5).
        Build()

    ctx := context.Background()
    err = ginflux.WriteBlocking(ctx, point)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 核心功能

### 1. 配置管理

```go
config := ginflux.NewDefaultConfig(
    "http://localhost:8086",
    "your-token",
    "your-org",
    "your-bucket",
).WithBatchSize(1000).                      // 批量写入大小
  WithFlushInterval(5 * time.Second).       // 刷新间隔
  WithMaxRetries(3).                        // 最大重试次数
  WithHTTPRequestTimeout(30 * time.Second). // HTTP 超时
  WithUseGZip(true).                        // 启用压缩
  WithLogLevel(2)                           // 日志级别
```

### 2. 数据写入

#### 单点写入

```go
// 阻塞写入
point := ginflux.NewPoint("temperature").
    AddTag("location", "room1").
    AddTag("sensor", "sensor1").
    AddField("value", 23.5).
    AddField("humidity", 65.0).
    Build()

err := client.WriteBlocking(ctx, point)

// 非阻塞写入
client.Write(point)
```

#### 批量写入

```go
points := []*write.Point{
    ginflux.NewPoint("cpu").AddTag("host", "server01").AddField("usage", 75.5).Build(),
    ginflux.NewPoint("cpu").AddTag("host", "server02").AddField("usage", 82.3).Build(),
    ginflux.NewPoint("cpu").AddTag("host", "server03").AddField("usage", 68.1).Build(),
}

err := client.WriteBatchBlocking(ctx, points...)
```

#### 使用批量写入器

```go
// 创建批量写入器
writer := ginflux.NewWriter(client, "my-bucket", 100, 2*time.Second)
defer writer.Close()

// 监听错误
go func() {
    for err := range writer.Errors() {
        log.Printf("Write error: %v", err)
    }
}()

// 写入数据
for i := 0; i < 1000; i++ {
    point := ginflux.NewPoint("sensor_data").
        AddTag("sensor_id", fmt.Sprintf("sensor_%d", i)).
        AddField("value", rand.Float64()*100).
        Build()

    writer.Write(point)
}

// 强制刷新
writer.Flush()
```

### 3. 查询数据

#### 使用查询构建器

```go
// 简单查询
records, err := ginflux.NewQueryBuilder("my-bucket").
    Measurement("cpu").
    Start("-1h").
    FilterTag("host", "server01").
    FilterField("usage").
    Execute(ctx, client)

// 聚合查询
records, err := ginflux.NewQueryBuilder("my-bucket").
    Measurement("temperature").
    Start("-24h").
    FilterTag("location", "room1").
    FilterField("value").
    Mean().
    GroupBy("location").
    Execute(ctx, client)

// 复杂查询
records, err := ginflux.NewQueryBuilder("my-bucket").
    Measurement("sensor_data").
    Start("-7d").
    Stop("-1d").
    FilterTag("sensor_type", "temperature").
    FilterFields("value", "humidity").
    Filter(`r["value"] > 20.0`).
    Mean().
    GroupBy("location", "sensor_id").
    Sort(`"_time"`).
    Limit(100).
    Execute(ctx, client)
```

#### 原始 Flux 查询

```go
query := `
from(bucket: "my-bucket")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "cpu")
  |> filter(fn: (r) => r["host"] == "server01")
  |> mean()
`

result, err := client.Query(ctx, query)
for result.Next() {
    log.Printf("Value: %v", result.Record().Value())
}
```

### 4. 指标收集

```go
// 创建指标收集器
metrics := ginflux.NewMetrics(
    client,
    "api_metrics",
    map[string]string{
        "service": "user-api",
        "env":     "production",
    },
)

// 记录指标
err := metrics.Record(map[string]interface{}{
    "requests_total": 1,
    "response_time":  125,
    "status_code":    200,
})

// 记录计数器
err := metrics.RecordCounter("requests", 1, map[string]string{
    "endpoint": "/api/users",
    "method":   "GET",
})

// 记录仪表盘值
err := metrics.RecordGauge("cpu_usage", 75.5, map[string]string{
    "host": "server01",
})

// 使用计时器
timer := metrics.StartTimer("request_duration", map[string]string{
    "endpoint": "/api/users",
})
// ... 执行操作 ...
timer.Stop()

// 或者带错误处理
timer := metrics.StartTimer("db_query")
result, err := db.Query(...)
timer.StopWithError(err)

// 查询最新值
latestValue, err := metrics.GetLatestValue(ctx, "cpu_usage", map[string]string{
    "host": "server01",
})

// 查询聚合值
avgResponseTime, err := metrics.GetAggregatedValue(
    ctx,
    "response_time",
    "mean()",
    "-1h",
    "",
    map[string]string{"endpoint": "/api/users"},
)
```

### 5. Bucket 管理

```go
bucketMgr := ginflux.NewBucketManager(client)

// 创建 Bucket（保留 30 天）
bucket, err := bucketMgr.CreateBucket(ctx, "my-bucket", 30*24)

// 列出所有 Buckets
buckets, err := bucketMgr.ListBuckets(ctx)

// 获取 Bucket
bucket, err := bucketMgr.GetBucket(ctx, "my-bucket")

// 更新保留策略（改为 60 天）
bucket, err := bucketMgr.UpdateBucketRetention(ctx, "my-bucket", 60*24)

// 删除数据
err := bucketMgr.DeleteData(
    ctx,
    "my-bucket",
    "temperature",
    time.Now().Add(-24*time.Hour),
    time.Now(),
    `location="room1"`,
)

// 删除 Bucket
err := bucketMgr.DeleteBucket(ctx, "my-bucket")
```

### 6. 健康检查

```go
// Ping 测试
ok, err := client.Ping(ctx)

// 健康检查
health, err := client.Health(ctx)
log.Printf("Status: %s", health.Status)

// 就绪检查
ready, err := client.Ready(ctx)
```

## 配置选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| ServerURL | string | - | InfluxDB 服务器地址 |
| Token | string | - | 认证令牌 |
| Organization | string | - | 组织名称 |
| Bucket | string | - | 默认存储桶 |
| Precision | string | "ns" | 时间精度 (ns/us/ms/s) |
| BatchSize | uint | 5000 | 批量写入大小 |
| FlushInterval | time.Duration | 1s | 刷新间隔 |
| RetryInterval | time.Duration | 5s | 重试间隔 |
| MaxRetries | uint | 3 | 最大重试次数 |
| MaxRetryInterval | time.Duration | 125s | 最大重试间隔 |
| HTTPRequestTimeout | time.Duration | 20s | HTTP 请求超时 |
| UseGZip | bool | false | 是否启用 GZip 压缩 |
| LogLevel | uint | 1 | 日志级别 (0-4) |

## 示例代码

查看 `examples/` 目录获取更多示例：

- `basic.go` - 基础使用示例
- `batch_write.go` - 批量写入示例
- `bucket_management.go` - Bucket 管理示例
- `advanced.go` - 高级功能示例
- `metrics.go` - 指标收集示例

运行示例：

```bash
cd examples
go run basic.go
```

## 最佳实践

### 1. 连接管理

```go
// 使用单例模式管理全局客户端
var client *ginflux.Client

func InitInfluxDB() error {
    config := ginflux.NewDefaultConfig(...)
    var err error
    client, err = ginflux.NewClient(config)
    return err
}

func GetInfluxDB() *ginflux.Client {
    return client
}

func CloseInfluxDB() {
    if client != nil {
        client.Close()
    }
}
```

### 2. 错误处理

```go
// 非阻塞写入时监听错误
writeAPI := client.WriteAPI()
go func() {
    for err := range client.Errors() {
        log.Printf("InfluxDB write error: %v", err)
        // 可以实现重试逻辑或告警
    }
}()
```

### 3. 批量写入优化

```go
// 使用批量写入器处理大量数据
writer := ginflux.NewWriter(client, bucket, 1000, 5*time.Second)
defer writer.Close()

// 并发写入
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        for j := 0; j < 1000; j++ {
            point := ginflux.NewPoint("data").
                AddTag("worker", fmt.Sprintf("worker_%d", id)).
                AddField("value", j).
                Build()
            writer.Write(point)
        }
    }(i)
}
wg.Wait()
```

### 4. 查询优化

```go
// 使用时间范围限制查询
qb := ginflux.NewQueryBuilder(bucket).
    Measurement("metrics").
    Start("-1h").              // 只查询最近1小时
    FilterField("cpu_usage").
    Limit(1000)                // 限制返回数量

// 使用聚合减少数据量
qb := ginflux.NewQueryBuilder(bucket).
    Measurement("metrics").
    Start("-24h").
    Mean().                    // 计算平均值
    GroupBy("host")            // 按主机分组
```

### 5. 标签设计

```go
// 好的标签设计：低基数、有意义
point := ginflux.NewPoint("http_requests").
    AddTag("method", "GET").           // 低基数
    AddTag("endpoint", "/api/users").  // 低基数
    AddTag("status", "200").           // 低基数
    AddField("duration_ms", 125).      // 高基数数据用字段
    AddField("bytes", 1024).
    Build()

// 避免：高基数标签
// AddTag("user_id", "12345")  // ❌ 用户ID基数太高
// AddTag("timestamp", "...")  // ❌ 时间戳基数太高
```

## 性能建议

1. **批量写入** - 使用批量写入而不是单点写入
2. **异步写入** - 对于非关键数据使用非阻塞写入
3. **启用压缩** - 网络带宽有限时启用 GZip 压缩
4. **合理的批次大小** - 根据数据量调整 BatchSize（1000-5000）
5. **连接复用** - 复用客户端连接，避免频繁创建
6. **查询优化** - 使用时间范围、限制返回数量、使用聚合

## 故障排查

### 连接失败

```go
// 检查连接
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

ok, err := client.Ping(ctx)
if err != nil {
    log.Printf("Connection failed: %v", err)
    // 检查：
    // 1. ServerURL 是否正确
    // 2. Token 是否有效
    // 3. 网络是否可达
    // 4. InfluxDB 服务是否运行
}
```

### 写入失败

```go
// 启用调试日志
config := ginflux.NewDefaultConfig(...).
    WithLogLevel(4)  // 最详细的日志

// 检查错误
err := client.WriteBlocking(ctx, point)
if err != nil {
    log.Printf("Write failed: %v", err)
    // 常见原因：
    // 1. Bucket 不存在
    // 2. Token 权限不足
    // 3. 数据格式错误
    // 4. 网络超时
}
```

### 查询失败

```go
result, err := client.Query(ctx, query)
if err != nil {
    log.Printf("Query failed: %v", err)
    // 检查：
    // 1. Flux 语法是否正确
    // 2. Bucket 是否存在
    // 3. 时间范围是否合理
}

for result.Next() {
    // 处理结果
}

if result.Err() != nil {
    log.Printf("Query result error: %v", result.Err())
}
```

## 版本要求

- Go >= 1.18
- InfluxDB >= 2.0

## 依赖

- `github.com/influxdata/influxdb-client-go/v2` v2.13.0

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 相关链接

- [InfluxDB 官方文档](https://docs.influxdata.com/influxdb/v2/)
- [Flux 查询语言](https://docs.influxdata.com/flux/v0/)
- [官方 Go 客户端](https://github.com/influxdata/influxdb-client-go)
