# ginflux

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![InfluxDB](https://img.shields.io/badge/InfluxDB-2.x-purple)](https://docs.influxdata.com/influxdb/v2/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

`ginflux` 是一个面向 InfluxDB 2.x 的 Go 客户端封装库，基于官方 [`github.com/influxdata/influxdb-client-go/v2`](https://github.com/influxdata/influxdb-client-go) 构建，提供更简洁的配置、写入、查询、Bucket 管理、指标记录和 OpenTelemetry Trace 接入能力。

它适合在业务服务、后台任务、采集程序、监控系统和数据写入 SDK 中作为 InfluxDB 访问层使用。

## 目录

- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [OpenTelemetry Trace](#opentelemetry-trace)
- [配置](#配置)
- [Client Option](#client-option)
- [数据点构建](#数据点构建)
- [写入数据](#写入数据)
- [查询数据](#查询数据)
- [指标收集](#指标收集)
- [Bucket 管理](#bucket-管理)
- [健康检查](#健康检查)
- [全局默认客户端](#全局默认客户端)
- [错误处理](#错误处理)
- [测试](#测试)
- [生产实践建议](#生产实践建议)
- [版本要求](#版本要求)
- [依赖](#依赖)
- [许可证](#许可证)

## 特性

- **简洁配置**：通过 `NewDefaultConfig` 获取生产可用默认配置，并支持链式调整。
- **Client Option 扩展机制**：`NewClient(config, opts...)` 支持按需挂载 HTTP Transport 能力。
- **OpenTelemetry Trace 接入**：通过 `contrib/otel` 子包提供 `WithTracing()`，按需启用 HTTP client trace。
- **阻塞与非阻塞写入**：同时支持官方 SDK 的 blocking write 和 async write。
- **批量写入器**：提供 `Writer`，支持缓冲、定时 flush、错误通道和关闭等待。
- **Flux 查询构建器**：通过链式 API 构建常见 Flux 查询，减少手写字符串。
- **指标收集工具**：提供 counter、gauge、timing、timer 等常用指标写入封装。
- **Bucket 管理**：封装 Bucket 创建、查询、删除、保留策略更新和数据删除。
- **健康检查**：封装 `Ping`、`Health`、`Ready`。
- **兼容官方 SDK**：保留访问底层 Organizations、Buckets、Users、Authorizations、Tasks、Labels、Delete API 的能力。

## 安装

```bash
go get github.com/nicexiaonie/ginflux
```

如需使用 OpenTelemetry Trace：

```bash
go get github.com/nicexiaonie/ginflux/contrib/otel
```

## 快速开始

```go
package main

import (
    "context"
    "log"

    "github.com/nicexiaonie/ginflux"
)

func main() {
    config := ginflux.NewDefaultConfig(
        "http://localhost:8086",
        "your-token",
        "your-org",
        "your-bucket",
    )

    client, err := ginflux.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    point := ginflux.NewPoint("temperature").
        AddTag("location", "room1").
        AddField("value", 23.5).
        Build()

    if err := client.WriteBlocking(ctx, point); err != nil {
        log.Fatal(err)
    }

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
        log.Printf("value=%v", result.Record().Value())
    }

    if result.Err() != nil {
        log.Fatal(result.Err())
    }
}
```

## OpenTelemetry Trace

`ginflux` 的核心包不直接初始化 OpenTelemetry SDK，也不替业务设置 exporter、resource、sampling 策略。应用侧应先按自己的观测平台完成 OpenTelemetry 初始化，然后通过 `contrib/otel` 子包启用 ginflux 的 HTTP client trace。

### 使用方式

```go
package main

import (
    "context"
    "log"

    "github.com/nicexiaonie/ginflux"
    ginfluxotel "github.com/nicexiaonie/ginflux/contrib/otel"
)

func main() {
    config := ginflux.NewDefaultConfig(
        "http://localhost:8086",
        "your-token",
        "your-org",
        "your-bucket",
    )

    client, err := ginflux.NewClient(
        config,
        ginfluxotel.WithTracing(),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    point := ginflux.NewPoint("cpu").AddField("usage", 75.5).Build()

    if err := client.WriteBlocking(ctx, point); err != nil {
        log.Fatal(err)
    }
}
```

`ginfluxotel.WithTracing()` 返回的是 `ginflux.ClientOption`，内部会将 ginflux 使用的 HTTP Transport 包装为 OpenTelemetry 的 `otelhttp.NewTransport`。之后，底层 InfluxDB HTTP 请求会自动生成 HTTP client span。

### 自定义 otelhttp.Option

`WithTracing` 支持透传 `otelhttp.Option`：

```go
import (
    "net/http"

    "github.com/nicexiaonie/ginflux"
    ginfluxotel "github.com/nicexiaonie/ginflux/contrib/otel"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

client, err := ginflux.NewClient(
    config,
    ginfluxotel.WithTracing(
        otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
            return "InfluxDB " + req.Method + " " + req.URL.Path
        }),
    ),
)
```

### Trace context 传播边界

`WithTracing()` 只负责 HTTP Transport instrumentation，不改变 API 的 context 传播语义。

推荐在需要完整父子链路的场景使用带 `context.Context` 的 API：

- `WriteBlocking`
- `WriteBlockingWithBucket`
- `WriteBatchBlocking`
- `WriteRecordBlocking`
- `Query`
- `QueryRaw`
- `Ping`
- `Health`
- `Ready`
- `Setup`

这些方法会把调用方传入的 `ctx` 继续传给底层 SDK。只要调用方 `ctx` 中已有上游 span，InfluxDB HTTP client span 就可以挂在上游 span 下面。

当前不建议对以下路径承诺完整父子链路：

- `Write`
- `WriteWithBucket`
- `WriteBatch`
- `WriteRecord`
- `Writer`

原因是非阻塞写入方法没有 `context.Context` 参数；`Writer` 内部异步 flush 会使用后台 context。它们仍会经过 wrapped transport，但不保证继承业务请求的上游 span。

## 配置

### 创建默认配置

```go
config := ginflux.NewDefaultConfig(
    "http://localhost:8086",
    "your-token",
    "your-org",
    "your-bucket",
)
```

默认值：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `ServerURL` | `string` | 必填 | InfluxDB 地址，例如 `http://localhost:8086` |
| `Token` | `string` | 必填 | InfluxDB API Token |
| `Organization` | `string` | 必填 | 组织名称 |
| `Bucket` | `string` | 必填 | 默认 Bucket |
| `Precision` | `string` | `ns` | 时间精度，支持 `ns`、`us`、`ms`、`s` |
| `BatchSize` | `uint` | `5000` | 官方异步写入 API 批大小 |
| `FlushInterval` | `time.Duration` | `1s` | 官方异步写入 API 刷新间隔 |
| `RetryInterval` | `time.Duration` | `5s` | 重试间隔 |
| `MaxRetries` | `uint` | `3` | 最大重试次数 |
| `MaxRetryInterval` | `time.Duration` | `125s` | 最大重试间隔 |
| `ExponentialBase` | `uint` | `2` | 指数退避基数 |
| `HTTPRequestTimeout` | `time.Duration` | `20s` | HTTP 请求超时时间 |
| `UseGZip` | `bool` | `false` | 是否启用 GZip |
| `LogLevel` | `uint` | `1` | 日志级别，`0` 无日志，`1` 错误，`2` 警告，`3` 信息，`4` 调试 |
| `TransportWrapper` | `func(http.RoundTripper) http.RoundTripper` | `nil` | HTTP Transport 包装入口 |

### 链式配置

```go
config := ginflux.NewDefaultConfig(
    "http://localhost:8086",
    "your-token",
    "your-org",
    "your-bucket",
).
    WithBatchSize(1000).
    WithFlushInterval(5 * time.Second).
    WithRetryInterval(3 * time.Second).
    WithMaxRetries(5).
    WithHTTPRequestTimeout(30 * time.Second).
    WithUseGZip(true).
    WithLogLevel(2).
    WithPrecision("ms")
```

### 配置校验

`NewClient` 会调用 `config.Validate()`。以下配置会返回错误：

- `ServerURL` 为空
- `Token` 为空
- `Organization` 为空
- `Bucket` 为空
- `Precision` 不是 `ns`、`us`、`ms`、`s` 之一

## Client Option

`NewClient` 支持变参 option：

```go
client, err := ginflux.NewClient(config, opts...)
```

当前主包提供：

```go
ginflux.WithTransportWrapper(wrapper)
```

它可以包装 ginflux 内部创建的 `http.RoundTripper`，适合接入 tracing、监控、自定义 header、请求日志等 HTTP 层能力。

```go
client, err := ginflux.NewClient(
    config,
    ginflux.WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
        return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
            req.Header.Set("X-Client", "ginflux")
            return base.RoundTrip(req)
        })
    }),
)
```

如果同时设置了 `config.WithTransportWrapper(...)` 和 `NewClient(config, opts...)`，顺序为：

1. 基础 `http.Transport`
2. `config.TransportWrapper`
3. `NewClient` 传入的 `ClientOption` transport wrapper，按传入顺序继续包裹

## 数据点构建

### 链式构建

```go
point := ginflux.NewPoint("http_requests").
    AddTag("method", "GET").
    AddTag("endpoint", "/api/users").
    AddTag("status", "200").
    AddField("duration_ms", 125).
    AddField("bytes", 1024).
    SetTimestamp(time.Now()).
    Build()
```

### 批量添加标签和字段

```go
point := ginflux.NewPoint("system_metrics").
    AddTags(map[string]string{
        "host": "server01",
        "region": "cn-east",
    }).
    AddFields(map[string]interface{}{
        "cpu":    75.5,
        "memory": 8192,
        "disk":   "healthy",
    }).
    Build()
```

### 直接构建

```go
point := ginflux.BuildPoint(
    "temperature",
    map[string]string{"location": "room1"},
    map[string]interface{}{"value": 23.5},
    time.Now(),
)

pointNow := ginflux.BuildPointNow(
    "temperature",
    map[string]string{"location": "room1"},
    map[string]interface{}{"value": 23.5},
)
```

## 写入数据

### 阻塞写入

阻塞写入会返回明确错误，适合需要确认写入结果、需要 trace 关联、需要请求级超时控制的场景。

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

point := ginflux.NewPoint("temperature").
    AddTag("location", "room1").
    AddField("value", 23.5).
    Build()

if err := client.WriteBlocking(ctx, point); err != nil {
    log.Printf("write failed: %v", err)
}
```

### 写入指定 Bucket

```go
err := client.WriteBlockingWithBucket(ctx, "another-bucket", point)
```

### 批量阻塞写入

```go
points := []*write.Point{
    ginflux.NewPoint("cpu").AddTag("host", "server01").AddField("usage", 75.5).Build(),
    ginflux.NewPoint("cpu").AddTag("host", "server02").AddField("usage", 82.3).Build(),
}

if err := client.WriteBatchBlocking(ctx, points...); err != nil {
    log.Printf("batch write failed: %v", err)
}
```

### 行协议写入

```go
record := `cpu,host=server01 usage=75.5`

if err := client.WriteRecordBlocking(ctx, record); err != nil {
    log.Printf("record write failed: %v", err)
}
```

### 非阻塞写入

非阻塞写入使用底层官方 SDK 的异步写入 API。调用会快速返回，写入错误需要从错误通道读取。

```go
client.Write(point)
client.WriteBatch(points...)
client.WriteRecord(`cpu,host=server01 usage=75.5`)

client.Flush()
```

监听异步错误：

```go
go func() {
    for err := range client.Errors() {
        log.Printf("async write error: %v", err)
    }
}()
```

### 批量写入器 Writer

`Writer` 提供独立缓冲区，达到批大小或定时器触发时异步 flush。

```go
writer := ginflux.NewWriter(client, "my-bucket", 100, 2*time.Second)
defer writer.Close()

go func() {
    for err := range writer.Errors() {
        log.Printf("writer error: %v", err)
    }
}()

for i := 0; i < 1000; i++ {
    point := ginflux.NewPoint("sensor_data").
        AddTag("sensor_id", fmt.Sprintf("sensor_%d", i)).
        AddField("value", rand.Float64()*100).
        Build()

    if err := writer.Write(point); err != nil {
        log.Printf("buffer write failed: %v", err)
    }
}

if err := writer.Flush(); err != nil {
    log.Printf("flush failed: %v", err)
}
```

`Writer.Close()` 会先 flush，随后等待内部 goroutine 结束并关闭错误通道。

## 查询数据

### 使用 QueryBuilder

```go
records, err := ginflux.NewQueryBuilder("my-bucket").
    Measurement("cpu").
    Start("-1h").
    FilterTag("host", "server01").
    FilterField("usage").
    Mean().
    GroupBy("host").
    Execute(ctx, client)
if err != nil {
    log.Printf("query failed: %v", err)
}

for _, record := range records {
    log.Printf("record=%v", record)
}
```

### 构建复杂查询

```go
query := ginflux.NewQueryBuilder("my-bucket").
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
    Build()
```

### 使用 time.Time 设置时间范围

```go
records, err := ginflux.NewQueryBuilder("my-bucket").
    Measurement("cpu").
    StartTime(time.Now().Add(-1 * time.Hour)).
    StopTime(time.Now()).
    FilterField("usage").
    Execute(ctx, client)
```

### 原始 Flux 查询

```go
flux := `
from(bucket: "my-bucket")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "cpu")
  |> filter(fn: (r) => r["host"] == "server01")
  |> mean()
`

result, err := client.Query(ctx, flux)
if err != nil {
    log.Printf("query failed: %v", err)
    return
}

for result.Next() {
    log.Printf("value=%v", result.Record().Value())
}

if result.Err() != nil {
    log.Printf("query result error: %v", result.Err())
}
```

### 查询原始字符串

```go
raw, err := client.QueryRaw(ctx, flux)
if err != nil {
    log.Printf("query raw failed: %v", err)
}
log.Println(raw)
```

## 指标收集

`Metrics` 是基于 `Client` 的轻量指标写入工具，适合记录应用层计数器、仪表值、耗时等。

```go
metrics := ginflux.NewMetrics(
    client,
    "api_metrics",
    map[string]string{
        "service": "user-api",
        "env":     "production",
    },
)
```

### 记录普通字段

```go
err := metrics.Record(map[string]interface{}{
    "requests_total": 1,
    "response_time":  125,
    "status_code":    200,
})
```

### 记录 Counter / Gauge / Timing

```go
err := metrics.RecordCounter("requests", 1, map[string]string{
    "endpoint": "/api/users",
    "method":   "GET",
})

err = metrics.RecordGauge("cpu_usage", 75.5, map[string]string{
    "host": "server01",
})

err = metrics.RecordTiming("request_duration", 120*time.Millisecond, map[string]string{
    "endpoint": "/api/users",
})
```

### Timer

```go
timer := metrics.StartTimer("db_query", map[string]string{
    "operation": "list_users",
})

err := doQuery()

if stopErr := timer.StopWithError(err); stopErr != nil {
    log.Printf("record timer failed: %v", stopErr)
}
```

### 查询指标

```go
latest, err := metrics.GetLatestValue(ctx, "cpu_usage", map[string]string{
    "host": "server01",
})

avg, err := metrics.GetAggregatedValue(
    ctx,
    "response_time",
    "mean()",
    "-1h",
    "",
    map[string]string{"endpoint": "/api/users"},
)

records, err := metrics.QueryMetrics(ctx, "-1h", "", map[string]string{
    "method": "GET",
})

_ = latest
_ = avg
_ = records
```

## Bucket 管理

```go
bucketMgr := ginflux.NewBucketManager(client)
```

### 创建 Bucket

`retentionHours` 单位是小时。

```go
bucket, err := bucketMgr.CreateBucket(ctx, "metrics", 30*24)
if err != nil {
    log.Printf("create bucket failed: %v", err)
}
_ = bucket
```

### 查询 Bucket

```go
bucket, err := bucketMgr.GetBucket(ctx, "metrics")

buckets, err := bucketMgr.ListBuckets(ctx)

_ = bucket
_ = buckets
```

### 更新保留策略

```go
bucket, err := bucketMgr.UpdateBucketRetention(ctx, "metrics", 60*24)
_ = bucket
```

### 删除数据

```go
err := bucketMgr.DeleteData(
    ctx,
    "metrics",
    "temperature",
    time.Now().Add(-24*time.Hour),
    time.Now(),
    `location="room1"`,
)
```

### 删除 Bucket

```go
err := bucketMgr.DeleteBucket(ctx, "metrics")
```

## 健康检查

```go
ok, err := client.Ping(ctx)
if err != nil {
    log.Printf("ping failed: %v", err)
}

health, err := client.Health(ctx)
if err != nil {
    log.Printf("health failed: %v", err)
}

ready, err := client.Ready(ctx)
if err != nil {
    log.Printf("ready failed: %v", err)
}

_ = ok
_ = health
_ = ready
```

## 全局默认客户端

`ginflux` 提供全局默认客户端，适合简单应用或脚本。大型服务中更推荐显式持有 `*ginflux.Client`。

```go
config := ginflux.NewDefaultConfig(
    "http://localhost:8086",
    "your-token",
    "your-org",
    "your-bucket",
)

if err := ginflux.Connect(config); err != nil {
    log.Fatal(err)
}
defer ginflux.Close()

point := ginflux.NewPoint("cpu").
    AddTag("host", "server01").
    AddField("usage", 75.5).
    Build()

if err := ginflux.WriteBlocking(context.Background(), point); err != nil {
    log.Fatal(err)
}
```

全局客户端也支持 `ClientOption`：

```go
err := ginflux.Connect(
    config,
    ginfluxotel.WithTracing(),
)
```

## 错误处理

### 创建客户端失败

```go
client, err := ginflux.NewClient(config)
if err != nil {
    log.Fatalf("create ginflux client failed: %v", err)
}
```

常见原因：

- 必填配置为空
- `Precision` 非法

### 写入失败

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := client.WriteBlocking(ctx, point); err != nil {
    log.Printf("write failed: %v", err)
}
```

常见原因：

- InfluxDB 地址错误或不可达
- Token 无效或权限不足
- Bucket 不存在
- 请求超时
- 数据格式不符合 InfluxDB line protocol 要求

### 查询失败

```go
result, err := client.Query(ctx, flux)
if err != nil {
    log.Printf("query failed: %v", err)
    return
}

for result.Next() {
    // process record
}

if result.Err() != nil {
    log.Printf("query result error: %v", result.Err())
}
```

常见原因：

- Flux 语法错误
- Bucket 不存在
- 时间范围不合理
- Token 缺少查询权限

### 非阻塞写入错误

```go
go func() {
    for err := range client.Errors() {
        log.Printf("async write error: %v", err)
    }
}()
```

## 测试

运行全部测试：

```bash
go test ./...
```

当前测试覆盖：

- 配置校验与链式配置
- PointBuilder
- QueryBuilder
- Metrics 基础结构
- Client 创建与连接检查
- TransportWrapper 请求路径
- HTTP 请求超时
- ClientOption transport wrapper 组合
- `contrib/otel` tracing option 请求路径

仓库中还包含 `test/` 目录，用于独立的集成测试、基准测试和环境配置示例。运行集成测试前需要准备 InfluxDB 地址、Token、Org 和 Bucket。

## 生产实践建议

### 1. 复用 Client

`Client` 内部复用 HTTP 连接。生产环境应复用一个或少量 client 实例，不要每次写入或查询都创建新 client。

```go
var influxClient *ginflux.Client

func InitInflux(config *ginflux.Config) error {
    client, err := ginflux.NewClient(config)
    if err != nil {
        return err
    }
    influxClient = client
    return nil
}

func CloseInflux() {
    if influxClient != nil {
        influxClient.Close()
    }
}
```

### 2. 请求级超时

带 `context.Context` 的 API 建议传入带超时的 context，避免业务请求长期阻塞。

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := client.WriteBlocking(ctx, point)
```

### 3. 高吞吐写入

高吞吐场景优先考虑：

- 使用批量写入
- 合理配置 `BatchSize` 和 `FlushInterval`
- 复用 client
- 网络带宽有限时启用 GZip
- 监听异步写入错误

```go
config := ginflux.NewDefaultConfig(...).
    WithBatchSize(5000).
    WithFlushInterval(2 * time.Second).
    WithUseGZip(true)
```

### 4. Trace 场景优先使用 blocking API

如果需要把 InfluxDB 请求挂到上游业务 trace 下，优先使用带 `context.Context` 的 blocking/query/health API。非阻塞写入和当前 `Writer` 不保证继承调用方 span。

### 5. 标签基数控制

InfluxDB 标签适合低基数字段，字段适合高基数或连续变化的数据。

推荐：

```go
point := ginflux.NewPoint("http_requests").
    AddTag("method", "GET").
    AddTag("endpoint", "/api/users").
    AddTag("status", "200").
    AddField("duration_ms", 125).
    AddField("user_id", "12345").
    Build()
```

避免把用户 ID、订单 ID、请求 ID、时间戳等高基数值放入 tag。

### 6. 查询范围控制

查询时尽量设置明确时间范围、measurement、field、tag 过滤条件，并限制返回数量。

```go
records, err := ginflux.NewQueryBuilder("metrics").
    Measurement("cpu").
    Start("-1h").
    FilterField("usage").
    FilterTag("host", "server01").
    Limit(1000).
    Execute(ctx, client)
```

## 版本要求

- Go >= 1.18
- InfluxDB >= 2.0

## 依赖

核心依赖：

- `github.com/influxdata/influxdb-client-go/v2` v2.13.0

OpenTelemetry Trace 子包依赖：

- `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` v0.45.0
- `go.opentelemetry.io/otel` v1.19.0

## 许可证

MIT License

## 相关链接

- [InfluxDB 官方文档](https://docs.influxdata.com/influxdb/v2/)
- [Flux 查询语言](https://docs.influxdata.com/flux/v0/)
- [InfluxDB Go Client](https://github.com/influxdata/influxdb-client-go)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
