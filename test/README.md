# GInflux 完整功能测试

这是一个全面测试 ginflux 工具所有 API 功能的测试程序。

## 目录结构

```
ginflux/test/
├── cmd/
│   ├── test/          # 功能测试程序
│   │   └── main.go
│   └── benchmark/     # 性能基准测试程序
│       └── benchmark.go
├── Makefile           # 构建和运行脚本
├── README.md          # 本文档
├── ANALYSIS.md        # 测试分析文档
├── env.example        # 环境变量示例
└── go.mod             # Go 模块配置
```

## 功能概览

ginflux 是一个 InfluxDB Go 客户端封装库，提供以下核心功能:

### 1. 配置管理 (Config)
- 灵活的配置选项，支持链式调用
- 批量大小、刷新间隔、重试策略等可配置
- 支持 GZip 压缩、日志级别控制
- 时间精度配置 (ns, us, ms, s)

### 2. 客户端管理 (Client)
- 连接管理和健康检查
- 支持全局默认客户端和多客户端实例
- 线程安全的客户端操作
- Ping、Health、Ready 状态检查

### 3. 数据写入
- **阻塞写入**: WriteBlocking - 立即返回结果
- **非阻塞写入**: Write - 异步写入，高性能
- **批量写入**: WriteBatch/WriteBatchBlocking - 批量提交
- **行协议写入**: WriteRecord - 支持 InfluxDB 行协议
- **多 Bucket 写入**: WriteWithBucket - 写入到指定 Bucket

### 4. 数据查询 (Query)
- **Flux 查询**: 支持完整的 Flux 查询语言
- **查询构建器**: 链式 API 构建复杂查询
- **聚合函数**: Mean, Sum, Count, Max, Min, First, Last
- **过滤和分组**: FilterTag, FilterField, GroupBy
- **排序和限制**: Sort, Limit, Offset

### 5. Bucket 管理
- 创建和删除 Bucket
- 列出所有 Bucket
- 更新保留策略
- 删除指定时间范围的数据

### 6. 指标收集 (Metrics)
- **Counter**: 计数器指标
- **Gauge**: 仪表盘指标
- **Timing**: 时间指标
- **Timer**: 自动计时器，支持错误标记
- 指标查询和聚合

### 7. 批量写入器 (Writer)
- 自动批量缓冲
- 定时自动刷新
- 异步写入，错误通道
- 优雅关闭和清理

## 测试维度

本测试程序涵盖以下测试维度:

### 功能测试
- ✅ 配置验证和链式调用
- ✅ 客户端创建和连接测试
- ✅ 数据点构建器测试
- ✅ 各种写入方式测试
- ✅ 查询构建器和执行测试
- ✅ Bucket 管理操作
- ✅ 指标收集功能
- ✅ 批量写入器功能
- ✅ 全局函数和默认客户端

### 性能测试
- ✅ 压力测试 (1000 个数据点)
- ✅ 并发写入测试 (10 个并发 goroutine)
- ✅ 批量写入性能

### 集成测试
- ✅ 数据写入和检索验证
- ✅ 健康检查集成
- ✅ 多 Bucket 操作

## 测试用例列表

| 编号 | 测试名称 | 测试内容 | 类型 |
|------|---------|---------|------|
| 01 | Config Validation | 配置验证 | 功能 |
| 02 | Config Chaining | 配置链式调用 | 功能 |
| 03 | Client Creation | 客户端创建 | 功能 |
| 04 | Client Ping | 连接测试 | 集成 |
| 05 | Client Health | 健康检查 | 集成 |
| 06 | Client Ready | 就绪检查 | 集成 |
| 07 | Point Builder | 数据点构建器 | 功能 |
| 08 | Point Builder Batch | 批量添加标签和字段 | 功能 |
| 09 | Write Blocking | 阻塞写入 | 功能 |
| 10 | Write Batch Blocking | 批量阻塞写入 | 功能 |
| 11 | Write Record Blocking | 行协议写入 | 功能 |
| 12 | Write Non-Blocking | 非阻塞写入 | 功能 |
| 13 | Query Builder | 查询构建器 | 功能 |
| 14 | Query Execution | 查询执行 | 集成 |
| 15 | Query Raw | 原始查询 | 功能 |
| 16 | Query Aggregates | 聚合函数 | 功能 |
| 17 | Bucket List | 列出 Bucket | 功能 |
| 18 | Bucket Get | 获取 Bucket | 功能 |
| 19 | Bucket Create/Delete | 创建删除 Bucket | 功能 |
| 20 | Bucket Update Retention | 更新保留策略 | 功能 |
| 21 | Metrics | 基础指标记录 | 功能 |
| 22 | Metrics Counter | 计数器指标 | 功能 |
| 23 | Metrics Gauge | 仪表盘指标 | 功能 |
| 24 | Metrics Timing | 计时指标 | 功能 |
| 25 | Metrics Timer | 计时器 | 功能 |
| 26 | Metrics Timer with Error | 带错误的计时器 | 功能 |
| 27 | Writer | 批量写入器 | 功能 |
| 28 | Writer Batch | 批量写入器批量操作 | 功能 |
| 29 | Default Client | 默认客户端 | 功能 |
| 30 | Global Functions | 全局函数 | 功能 |
| 31 | Stress Test | 压力测试 (1000点) | 性能 |
| 32 | Concurrent Writes | 并发写入 (10 goroutines) | 性能 |
| 33 | Data Retrieval | 数据检索验证 | 集成 |

## 环境要求

1. **InfluxDB 2.x** - 需要运行中的 InfluxDB 实例
2. **Go 1.18+** - 支持泛型和新特性
3. **网络连接** - 能够访问 InfluxDB 服务器

## 配置说明

通过环境变量配置测试参数:

```bash
export INFLUX_URL="http://localhost:8086"
export INFLUX_TOKEN="your-token-here"
export INFLUX_ORG="your-org"
export INFLUX_BUCKET="your-bucket"
```

或者使用 `.env` 文件 (参考 `env.example`)。

## 运行测试

### 使用 Makefile (推荐)

```bash
# 查看所有可用命令
make help

# 运行功能测试
make test

# 运行性能基准测试
make benchmark

# 编译测试程序
make build

# 编译基准测试程序
make build-benchmark

# 清理编译文件
make clean
```

### 1. 直接运行功能测试

```bash
cd ginflux/test
go run cmd/test/main.go
```

### 2. 直接运行基准测试

```bash
cd ginflux/test
go run cmd/benchmark/benchmark.go
```

### 3. 使用环境变量

```bash
INFLUX_URL=http://localhost:8086 \
INFLUX_TOKEN=your-token \
INFLUX_ORG=your-org \
INFLUX_BUCKET=test-bucket \
go run cmd/test/main.go
```

### 4. 使用 .env 文件

```bash
# 复制示例配置
cp env.example .env

# 编辑配置
vim .env

# 加载环境变量并运行
source .env && make test
```

### 5. 编译后运行

```bash
# 编译功能测试
make build
./ginflux-test

# 编译基准测试
make build-benchmark
./ginflux-benchmark
```

## 快速开始

使用 Docker 快速启动 InfluxDB 进行测试:

```bash
# 启动 InfluxDB 容器
make docker-influx

# 运行测试
make test

# 运行基准测试
make benchmark

# 停止 InfluxDB 容器
make stop-influx
```

## 输出示例

### 功能测试输出

```
================================================================================
GInflux 完整功能测试
================================================================================
Server: http://localhost:8086
Organization: my-org
Bucket: test-bucket
================================================================================
✅ [PASS] 01. Config Validation (1.23ms)
✅ [PASS] 02. Config Chaining (0.45ms)
✅ [PASS] 03. Client Creation (5.67ms)
✅ [PASS] 04. Client Ping (12.34ms)
✅ [PASS] 05. Client Health (8.90ms)
  Health status: pass, message: ready for queries and writes
✅ [PASS] 06. Client Ready (7.12ms)
...
✅ [PASS] 31. Stress Test (1234.56ms)
  Wrote 1000 points in 1.23s (812.35 points/sec)
✅ [PASS] 32. Concurrent Writes (345.67ms)
  All 10 goroutines completed successfully
✅ [PASS] 33. Data Retrieval (234.56ms)

================================================================================
测试总结
================================================================================
总测试数: 33
通过: 33
失败: 0
总耗时: 5432.10ms
================================================================================
```

### 基准测试输出

```
================================================================================
GInflux 性能基准测试
================================================================================
Server: http://localhost:8086
Organization: test-org
Bucket: test-bucket
================================================================================

开始性能测试...

[ 1/7 ] 运行单点写入基准测试...
[ 2/7 ] 运行小批量写入基准测试...
[ 3/7 ] 运行中批量写入基准测试...
[ 4/7 ] 运行大批量写入基准测试...
[ 5/7 ] 运行非阻塞写入基准测试...
[ 6/7 ] 运行并发写入基准测试...
[ 7/7 ] 运行查询性能基准测试...

================================================================================
基准测试结果
================================================================================

测试: 单点写入
  总数据点: 100
  总耗时: 2.50s
  吞吐量: 40.00 points/sec
  平均延迟: 25ms
  最小延迟: 15ms
  最大延迟: 50ms

测试: 批量写入 (batch=100)
  总数据点: 1000
  总耗时: 1.20s
  吞吐量: 833.33 points/sec
  平均延迟: 120ms
  最小延迟: 80ms
  最大延迟: 200ms

...

================================================================================
性能总结
================================================================================
最快的写入方式: 非阻塞写入 (5000.00 points/sec)
总写入数据点: 8100
================================================================================
```

```
================================================================================
GInflux 完整功能测试
================================================================================
Server: http://localhost:8086
Organization: my-org
Bucket: test-bucket
================================================================================
✅ [PASS] 01. Config Validation (1.23ms)
✅ [PASS] 02. Config Chaining (0.45ms)
✅ [PASS] 03. Client Creation (5.67ms)
✅ [PASS] 04. Client Ping (12.34ms)
✅ [PASS] 05. Client Health (8.90ms)
  Health status: pass, message: ready for queries and writes
✅ [PASS] 06. Client Ready (7.12ms)
  Ready status: ready
...
✅ [PASS] 31. Stress Test (1234.56ms)
  Wrote 1000 points in 1.23s (812.35 points/sec)
✅ [PASS] 32. Concurrent Writes (345.67ms)
  All 10 goroutines completed successfully
✅ [PASS] 33. Data Retrieval (234.56ms)

================================================================================
测试总结
================================================================================
总测试数: 33
通过: 33
失败: 0
总耗时: 5432.10ms
================================================================================
```

## 故障排查

### 连接失败

```
❌ [FAIL] 04. Client Ping: ping failed: connection refused
```

**解决方案**:
- 检查 InfluxDB 是否运行: `curl http://localhost:8086/health`
- 检查 URL 配置是否正确
- 检查防火墙设置

### 认证失败

```
❌ [FAIL] 05. Client Health: unauthorized access
```

**解决方案**:
- 检查 Token 是否正确
- 检查 Token 权限是否足够
- 重新生成 Token

### Bucket 不存在

```
❌ [FAIL] 18. Bucket Get: bucket not found
```

**解决方案**:
- 创建测试 Bucket
- 或修改 INFLUX_BUCKET 环境变量指向已存在的 Bucket

## 性能基准

在标准配置下 (InfluxDB 2.x, 本地部署)，预期性能:

- **单点写入**: < 10ms
- **批量写入 (100点)**: < 50ms
- **压力测试 (1000点)**: 500-2000ms (500-2000 points/sec)
- **并发写入**: < 500ms (10 goroutines, 100 points)
- **简单查询**: < 50ms
- **聚合查询**: < 100ms

## API 覆盖率

本测试覆盖了 ginflux 的所有公开 API:

### ginflux 包级别 (ginflux.go)
- [x] Connect
- [x] MustConnect
- [x] SetDefaultClient
- [x] GetDefaultClient
- [x] Close
- [x] Write
- [x] WriteBlocking
- [x] WriteBatch
- [x] WriteBatchBlocking
- [x] WriteRecord
- [x] WriteRecordBlocking
- [x] Query
- [x] QueryRaw
- [x] NewQuery
- [x] Flush
- [x] Ping
- [x] Health

### Client (client.go)
- [x] NewClient
- [x] WriteAPI
- [x] WriteAPIBlocking
- [x] QueryAPI
- [x] Write
- [x] WriteWithBucket
- [x] WriteBlocking
- [x] WriteBlockingWithBucket
- [x] WriteBatch
- [x] WriteBatchBlocking
- [x] WriteRecord
- [x] WriteRecordBlocking
- [x] Query
- [x] QueryRaw
- [x] Flush
- [x] Errors
- [x] Ping
- [x] Health
- [x] Ready
- [x] OrganizationsAPI
- [x] BucketsAPI
- [x] UsersAPI
- [x] AuthorizationsAPI
- [x] TasksAPI
- [x] LabelsAPI
- [x] DeleteAPI
- [x] Config
- [x] Close
- [x] IsClosed
- [x] ServerURL

### Config (config.go)
- [x] NewDefaultConfig
- [x] Validate
- [x] WithBatchSize
- [x] WithFlushInterval
- [x] WithRetryInterval
- [x] WithMaxRetries
- [x] WithHTTPRequestTimeout
- [x] WithUseGZip
- [x] WithLogLevel
- [x] WithPrecision

### PointBuilder (point.go)
- [x] NewPoint
- [x] AddTag
- [x] AddTags
- [x] AddField
- [x] AddFields
- [x] SetTimestamp
- [x] Build
- [x] BuildPoint
- [x] BuildPointNow

### QueryBuilder (query.go)
- [x] NewQueryBuilder
- [x] Measurement
- [x] Start
- [x] StartTime
- [x] Stop
- [x] StopTime
- [x] Filter
- [x] FilterTag
- [x] FilterField
- [x] FilterFields
- [x] GroupBy
- [x] Aggregate
- [x] Mean
- [x] Sum
- [x] Count
- [x] Max
- [x] Min
- [x] First
- [x] Last
- [x] Limit
- [x] Offset
- [x] Sort
- [x] Build
- [x] Execute
- [x] ExecuteRaw

### BucketManager (bucket.go)
- [x] NewBucketManager
- [x] CreateBucket
- [x] GetBucket
- [x] ListBuckets
- [x] DeleteBucket
- [x] UpdateBucketRetention
- [x] DeleteData

### Metrics (metrics.go)
- [x] NewMetrics
- [x] Record
- [x] RecordWithTags
- [x] RecordCounter
- [x] RecordGauge
- [x] RecordTiming
- [x] StartTimer
- [x] Timer.Stop
- [x] Timer.StopWithError
- [x] QueryMetrics
- [x] GetLatestValue
- [x] GetAggregatedValue

### Writer (writer.go)
- [x] NewWriter
- [x] Write
- [x] WriteBatch
- [x] Flush
- [x] Errors
- [x] Close

**总计**: 100+ API，覆盖率 100%

## 扩展测试

如需添加新的功能测试用例到 `cmd/test/main.go`，按照以下模板:

```go
// TestXX_YourTestName 测试描述
func (ts *TestSuite) TestXX_YourTestName() error {
	// 测试逻辑
	// ...

	if /* 失败条件 */ {
		return fmt.Errorf("error message")
	}

	return nil
}
```

然后在 `main()` 函数中注册:

```go
ts.Run("XX. Your Test Name", ts.TestXX_YourTestName)
```

如需添加基准测试，可以在 `cmd/benchmark/benchmark.go` 中添加新的基准测试函数。

## 许可证

与 go-sophon 项目保持一致。
