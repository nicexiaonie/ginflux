package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/redcoast/go-sophon/ginflux"
)

// TestConfig 测试配置
type TestConfig struct {
	ServerURL    string
	Token        string
	Organization string
	Bucket       string
}

// TestResult 测试结果
type TestResult struct {
	TestName string
	Passed   bool
	Error    error
	Duration time.Duration
}

// TestSuite 测试套件
type TestSuite struct {
	config  *TestConfig
	client  *ginflux.Client
	results []TestResult
	ctx     context.Context
}

// NewTestSuite 创建测试套件
func NewTestSuite(config *TestConfig) *TestSuite {
	return &TestSuite{
		config:  config,
		results: make([]TestResult, 0),
		ctx:     context.Background(),
	}
}

// Run 运行测试
func (ts *TestSuite) Run(name string, fn func() error) {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	result := TestResult{
		TestName: name,
		Passed:   err == nil,
		Error:    err,
		Duration: duration,
	}
	ts.results = append(ts.results, result)

	if err != nil {
		log.Printf("❌ [FAIL] %s (%.2fms): %v\n", name, float64(duration.Microseconds())/1000, err)
	} else {
		log.Printf("✅ [PASS] %s (%.2fms)\n", name, float64(duration.Microseconds())/1000)
	}
}

// PrintSummary 打印测试总结
func (ts *TestSuite) PrintSummary() {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range ts.results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("测试总结")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("总测试数: %d\n", len(ts.results))
	fmt.Printf("通过: %d\n", passed)
	fmt.Printf("失败: %d\n", failed)
	fmt.Printf("总耗时: %.2fms\n", float64(totalDuration.Microseconds())/1000)
	fmt.Println(strings.Repeat("=", 80))
}

// ============================================================================
// 测试用例
// ============================================================================

// Test1_ConfigValidation 测试配置验证
func (ts *TestSuite) Test1_ConfigValidation() error {
	// 测试有效配置
	validConfig := ginflux.NewDefaultConfig(
		ts.config.ServerURL,
		ts.config.Token,
		ts.config.Organization,
		ts.config.Bucket,
	)
	if err := validConfig.Validate(); err != nil {
		return fmt.Errorf("valid config validation failed: %w", err)
	}

	// 测试无效配置
	invalidConfig := &ginflux.Config{
		ServerURL:    "",
		Token:        "token",
		Organization: "org",
		Bucket:       "bucket",
	}
	if err := invalidConfig.Validate(); err == nil {
		return fmt.Errorf("invalid config should fail validation")
	}

	return nil
}

// Test2_ConfigChaining 测试配置链式调用
func (ts *TestSuite) Test2_ConfigChaining() error {
	config := ginflux.NewDefaultConfig(
		ts.config.ServerURL,
		ts.config.Token,
		ts.config.Organization,
		ts.config.Bucket,
	).WithBatchSize(1000).
		WithFlushInterval(5 * time.Second).
		WithUseGZip(true).
		WithLogLevel(2).
		WithPrecision("ms")

	if config.BatchSize != 1000 {
		return fmt.Errorf("BatchSize not set correctly")
	}
	if config.FlushInterval != 5*time.Second {
		return fmt.Errorf("FlushInterval not set correctly")
	}
	if !config.UseGZip {
		return fmt.Errorf("UseGZip not set correctly")
	}
	if config.Precision != "ms" {
		return fmt.Errorf("Precision not set correctly")
	}

	return nil
}

// Test3_ClientCreation 测试客户端创建
func (ts *TestSuite) Test3_ClientCreation() error {
	config := ginflux.NewDefaultConfig(
		ts.config.ServerURL,
		ts.config.Token,
		ts.config.Organization,
		ts.config.Bucket,
	)

	client, err := ginflux.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if client == nil {
		return fmt.Errorf("client is nil")
	}

	if client.IsClosed() {
		return fmt.Errorf("client should not be closed initially")
	}

	ts.client = client
	return nil
}

// Test4_ClientPing 测试连接健康检查
func (ts *TestSuite) Test4_ClientPing() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	ok, err := ts.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if !ok {
		return fmt.Errorf("ping returned false")
	}

	return nil
}

// Test5_ClientHealth 测试健康状态检查
func (ts *TestSuite) Test5_ClientHealth() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	health, err := ts.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if health == nil {
		return fmt.Errorf("health is nil")
	}

	log.Printf("  Health status: %s, message: %s", health.Status, *health.Message)
	return nil
}

// Test6_ClientReady 测试服务就绪检查
func (ts *TestSuite) Test6_ClientReady() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	ready, err := ts.client.Ready(ctx)
	if err != nil {
		return fmt.Errorf("ready check failed: %w", err)
	}

	if ready == nil {
		return fmt.Errorf("ready is nil")
	}

	log.Printf("  Ready status: %s", ready.Status)
	return nil
}

// Test7_PointBuilder 测试数据点构建器
func (ts *TestSuite) Test7_PointBuilder() error {
	point := ginflux.NewPoint("test_measurement").
		AddTag("host", "server01").
		AddTag("region", "us-west").
		AddField("cpu", 75.5).
		AddField("memory", 8192).
		AddField("status", "healthy").
		SetTimestamp(time.Now()).
		Build()

	if point == nil {
		return fmt.Errorf("point is nil")
	}

	return nil
}

// Test8_PointBuilderBatch 测试批量添加标签和字段
func (ts *TestSuite) Test8_PointBuilderBatch() error {
	tags := map[string]string{
		"host":   "server02",
		"region": "us-east",
		"dc":     "dc1",
	}

	fields := map[string]interface{}{
		"cpu":    85.2,
		"memory": 16384,
		"disk":   95.5,
	}

	point := ginflux.NewPoint("system_metrics").
		AddTags(tags).
		AddFields(fields).
		Build()

	if point == nil {
		return fmt.Errorf("point is nil")
	}

	return nil
}

// Test9_WriteBlocking 测试阻塞写入单个数据点
func (ts *TestSuite) Test9_WriteBlocking() error {
	point := ginflux.NewPoint("test_write").
		AddTag("test", "blocking").
		AddField("value", 100).
		Build()

	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	err := ts.client.WriteBlocking(ctx, point)
	if err != nil {
		return fmt.Errorf("write blocking failed: %w", err)
	}

	return nil
}

// Test10_WriteBatchBlocking 测试阻塞批量写入
func (ts *TestSuite) Test10_WriteBatchBlocking() error {
	points := make([]*write.Point, 10)
	for i := 0; i < 10; i++ {
		points[i] = ginflux.NewPoint("test_batch").
			AddTag("batch", "test").
			AddTag("index", fmt.Sprintf("%d", i)).
			AddField("value", i*10).
			Build()
	}

	ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	err := ts.client.WriteBatchBlocking(ctx, points...)
	if err != nil {
		return fmt.Errorf("batch write failed: %w", err)
	}

	return nil
}

// Test11_WriteRecordBlocking 测试行协议写入
func (ts *TestSuite) Test11_WriteRecordBlocking() error {
	record := "test_record,host=server03 value=200"

	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	err := ts.client.WriteRecordBlocking(ctx, record)
	if err != nil {
		return fmt.Errorf("write record failed: %w", err)
	}

	return nil
}

// Test12_WriteNonBlocking 测试非阻塞写入
func (ts *TestSuite) Test12_WriteNonBlocking() error {
	point := ginflux.NewPoint("test_nonblocking").
		AddTag("async", "true").
		AddField("value", 300).
		Build()

	ts.client.Write(point)
	ts.client.Flush()

	// 等待一下确保写入完成
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Test13_QueryBuilder 测试查询构建器
func (ts *TestSuite) Test13_QueryBuilder() error {
	qb := ginflux.NewQueryBuilder(ts.config.Bucket).
		Measurement("test_write").
		Start("-1h").
		Stop("now()").
		FilterTag("test", "blocking").
		FilterField("value").
		Limit(10)

	query := qb.Build()
	if query == "" {
		return fmt.Errorf("query is empty")
	}

	log.Printf("  Generated query: %s", query)
	return nil
}

// Test14_QueryExecution 测试查询执行
func (ts *TestSuite) Test14_QueryExecution() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "test_write")
  |> limit(n: 5)`, ts.config.Bucket)

	result, err := ts.client.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	count := 0
	for result.Next() {
		count++
		record := result.Record()
		log.Printf("  Record: measurement=%s, field=%s, value=%v",
			record.Measurement(), record.Field(), record.Value())
	}

	if result.Err() != nil {
		return fmt.Errorf("query result error: %w", result.Err())
	}

	log.Printf("  Query returned %d records", count)
	return nil
}

// Test15_QueryRaw 测试原始查询
func (ts *TestSuite) Test15_QueryRaw() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "test_write")
  |> limit(n: 1)`, ts.config.Bucket)

	raw, err := ts.client.QueryRaw(ctx, query)
	if err != nil {
		return fmt.Errorf("raw query failed: %w", err)
	}

	if raw == "" {
		log.Printf("  No data returned from raw query")
	} else {
		log.Printf("  Raw query result length: %d bytes", len(raw))
	}

	return nil
}

// Test16_QueryBuilderAggregates 测试聚合函数
func (ts *TestSuite) Test16_QueryBuilderAggregates() error {
	tests := []struct {
		name    string
		builder func(*ginflux.QueryBuilder) *ginflux.QueryBuilder
	}{
		{"Mean", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Mean() }},
		{"Sum", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Sum() }},
		{"Count", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Count() }},
		{"Max", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Max() }},
		{"Min", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Min() }},
		{"First", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.First() }},
		{"Last", func(qb *ginflux.QueryBuilder) *ginflux.QueryBuilder { return qb.Last() }},
	}

	for _, tt := range tests {
		qb := ginflux.NewQueryBuilder(ts.config.Bucket).
			Measurement("test_batch").
			Start("-1h")
		qb = tt.builder(qb)
		query := qb.Build()

		if query == "" {
			return fmt.Errorf("%s: query is empty", tt.name)
		}
	}

	return nil
}

// Test17_BucketList 测试列出所有 Buckets
func (ts *TestSuite) Test17_BucketList() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	bm := ginflux.NewBucketManager(ts.client)
	buckets, err := bm.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("list buckets failed: %w", err)
	}

	log.Printf("  Found %d buckets", len(*buckets))
	for _, bucket := range *buckets {
		log.Printf("    - %s (ID: %s)", bucket.Name, *bucket.Id)
	}

	return nil
}

// Test18_BucketGet 测试获取 Bucket
func (ts *TestSuite) Test18_BucketGet() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	bm := ginflux.NewBucketManager(ts.client)
	bucket, err := bm.GetBucket(ctx, ts.config.Bucket)
	if err != nil {
		return fmt.Errorf("get bucket failed: %w", err)
	}

	log.Printf("  Bucket: %s, Org: %s", bucket.Name, *bucket.OrgID)
	if len(bucket.RetentionRules) > 0 {
		log.Printf("  Retention: %d seconds", bucket.RetentionRules[0].EverySeconds)
	}

	return nil
}

// Test19_BucketCreateAndDelete 测试创建和删除 Bucket
func (ts *TestSuite) Test19_BucketCreateAndDelete() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 30*time.Second)
	defer cancel()

	bm := ginflux.NewBucketManager(ts.client)
	testBucketName := fmt.Sprintf("test_bucket_%d", time.Now().Unix())

	// 创建
	bucket, err := bm.CreateBucket(ctx, testBucketName, 24) // 24小时保留
	if err != nil {
		return fmt.Errorf("create bucket failed: %w", err)
	}
	log.Printf("  Created bucket: %s", bucket.Name)

	// 验证创建
	getBucket, err := bm.GetBucket(ctx, testBucketName)
	if err != nil {
		return fmt.Errorf("get created bucket failed: %w", err)
	}
	if getBucket.Name != testBucketName {
		return fmt.Errorf("bucket name mismatch")
	}

	// 删除
	err = bm.DeleteBucket(ctx, testBucketName)
	if err != nil {
		return fmt.Errorf("delete bucket failed: %w", err)
	}
	log.Printf("  Deleted bucket: %s", testBucketName)

	return nil
}

// Test20_BucketUpdateRetention 测试更新保留策略
func (ts *TestSuite) Test20_BucketUpdateRetention() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 30*time.Second)
	defer cancel()

	bm := ginflux.NewBucketManager(ts.client)
	testBucketName := fmt.Sprintf("test_retention_%d", time.Now().Unix())

	// 创建 bucket
	_, err := bm.CreateBucket(ctx, testBucketName, 24)
	if err != nil {
		return fmt.Errorf("create bucket failed: %w", err)
	}
	defer bm.DeleteBucket(ctx, testBucketName)

	// 更新保留策略
	updated, err := bm.UpdateBucketRetention(ctx, testBucketName, 48) // 改为48小时
	if err != nil {
		return fmt.Errorf("update retention failed: %w", err)
	}

	if len(updated.RetentionRules) == 0 {
		return fmt.Errorf("no retention rules after update")
	}

	expectedSeconds := int64(48 * 3600)
	if updated.RetentionRules[0].EverySeconds != expectedSeconds {
		return fmt.Errorf("retention not updated correctly: expected %d, got %d",
			expectedSeconds, updated.RetentionRules[0].EverySeconds)
	}

	log.Printf("  Updated retention to %d hours", 48)
	return nil
}

// Test21_Metrics 测试指标收集器
func (ts *TestSuite) Test21_Metrics() error {
	metrics := ginflux.NewMetrics(ts.client, "test_metrics", map[string]string{
		"service": "test-service",
		"env":     "test",
	})

	if metrics == nil {
		return fmt.Errorf("metrics is nil")
	}

	// 记录指标
	err := metrics.Record(map[string]interface{}{
		"requests": 100,
		"errors":   5,
		"latency":  125.5,
		"success":  true,
	})
	if err != nil {
		return fmt.Errorf("record metrics failed: %w", err)
	}

	return nil
}

// Test22_MetricsCounter 测试计数器指标
func (ts *TestSuite) Test22_MetricsCounter() error {
	metrics := ginflux.NewMetrics(ts.client, "test_counters", map[string]string{
		"app": "test",
	})

	err := metrics.RecordCounter("api_calls", 100)
	if err != nil {
		return fmt.Errorf("record counter failed: %w", err)
	}

	return nil
}

// Test23_MetricsGauge 测试仪表盘指标
func (ts *TestSuite) Test23_MetricsGauge() error {
	metrics := ginflux.NewMetrics(ts.client, "test_gauges", map[string]string{
		"app": "test",
	})

	err := metrics.RecordGauge("cpu_usage", 75.5)
	if err != nil {
		return fmt.Errorf("record gauge failed: %w", err)
	}

	return nil
}

// Test24_MetricsTiming 测试计时指标
func (ts *TestSuite) Test24_MetricsTiming() error {
	metrics := ginflux.NewMetrics(ts.client, "test_timings", map[string]string{
		"app": "test",
	})

	err := metrics.RecordTiming("request_duration", 250*time.Millisecond)
	if err != nil {
		return fmt.Errorf("record timing failed: %w", err)
	}

	return nil
}

// Test25_MetricsTimer 测试计时器
func (ts *TestSuite) Test25_MetricsTimer() error {
	metrics := ginflux.NewMetrics(ts.client, "test_timer", map[string]string{
		"app": "test",
	})

	timer := metrics.StartTimer("operation_duration")
	time.Sleep(50 * time.Millisecond) // 模拟操作
	err := timer.Stop()
	if err != nil {
		return fmt.Errorf("timer stop failed: %w", err)
	}

	return nil
}

// Test26_MetricsTimerWithError 测试带错误的计时器
func (ts *TestSuite) Test26_MetricsTimerWithError() error {
	metrics := ginflux.NewMetrics(ts.client, "test_timer_error", map[string]string{
		"app": "test",
	})

	timer := metrics.StartTimer("operation_with_error")
	time.Sleep(30 * time.Millisecond)
	err := timer.StopWithError(fmt.Errorf("test error"))
	if err != nil {
		return fmt.Errorf("timer stop with error failed: %w", err)
	}

	return nil
}

// Test27_Writer 测试批量写入器
func (ts *TestSuite) Test27_Writer() error {
	writer := ginflux.NewWriter(ts.client, ts.config.Bucket, 100, 1*time.Second)
	defer writer.Close()

	// 写入一些数据点
	for i := 0; i < 50; i++ {
		point := ginflux.NewPoint("test_writer").
			AddTag("batch", "writer_test").
			AddField("value", i).
			Build()

		err := writer.Write(point)
		if err != nil {
			return fmt.Errorf("writer write failed: %w", err)
		}
	}

	// 刷新
	err := writer.Flush()
	if err != nil {
		return fmt.Errorf("writer flush failed: %w", err)
	}

	return nil
}

// Test28_WriterBatch 测试批量写入器批量写入
func (ts *TestSuite) Test28_WriterBatch() error {
	writer := ginflux.NewWriter(ts.client, ts.config.Bucket, 200, 2*time.Second)
	defer writer.Close()

	points := make([]*write.Point, 100)
	for i := 0; i < 100; i++ {
		points[i] = ginflux.NewPoint("test_writer_batch").
			AddTag("type", "batch").
			AddField("value", i).
			Build()
	}

	err := writer.WriteBatch(points...)
	if err != nil {
		return fmt.Errorf("writer batch write failed: %w", err)
	}

	return nil
}

// Test29_DefaultClient 测试默认客户端
func (ts *TestSuite) Test29_DefaultClient() error {
	// 连接默认客户端
	config := ginflux.NewDefaultConfig(
		ts.config.ServerURL,
		ts.config.Token,
		ts.config.Organization,
		ts.config.Bucket,
	)

	err := ginflux.Connect(config)
	if err != nil {
		return fmt.Errorf("connect default client failed: %w", err)
	}

	// 测试使用默认客户端
	point := ginflux.NewPoint("test_default_client").
		AddTag("test", "default").
		AddField("value", 999).
		Build()

	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	err = ginflux.WriteBlocking(ctx, point)
	if err != nil {
		return fmt.Errorf("write with default client failed: %w", err)
	}

	return nil
}

// Test30_GlobalFunctions 测试全局函数
func (ts *TestSuite) Test30_GlobalFunctions() error {
	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()

	// Ping
	ok, err := ginflux.Ping(ctx)
	if err != nil {
		return fmt.Errorf("global ping failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("global ping returned false")
	}

	// Health
	err = ginflux.Health(ctx)
	if err != nil {
		return fmt.Errorf("global health failed: %w", err)
	}

	// Query
	query := fmt.Sprintf(`from(bucket: "%s") |> range(start: -1h) |> limit(n: 1)`, ts.config.Bucket)
	records, err := ginflux.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("global query failed: %w", err)
	}
	log.Printf("  Global query returned %d records", len(records))

	return nil
}

// Test31_StressTest 压力测试
func (ts *TestSuite) Test31_StressTest() error {
	log.Printf("  Starting stress test with 1000 points...")

	startTime := time.Now()
	batchSize := 100
	totalPoints := 1000

	for batch := 0; batch < totalPoints/batchSize; batch++ {
		points := make([]*write.Point, batchSize)
		for i := 0; i < batchSize; i++ {
			points[i] = ginflux.NewPoint("stress_test").
				AddTag("batch", fmt.Sprintf("%d", batch)).
				AddTag("index", fmt.Sprintf("%d", i)).
				AddField("value", batch*batchSize+i).
				AddField("timestamp", time.Now().Unix()).
				Build()
		}

		ctx, cancel := context.WithTimeout(ts.ctx, 10*time.Second)
		err := ts.client.WriteBatchBlocking(ctx, points...)
		cancel()

		if err != nil {
			return fmt.Errorf("stress test batch %d failed: %w", batch, err)
		}
	}

	duration := time.Since(startTime)
	rate := float64(totalPoints) / duration.Seconds()
	log.Printf("  Wrote %d points in %.2fs (%.2f points/sec)", totalPoints, duration.Seconds(), rate)

	return nil
}

// Test32_ConcurrentWrites 并发写入测试
func (ts *TestSuite) Test32_ConcurrentWrites() error {
	log.Printf("  Starting concurrent write test with 10 goroutines...")

	errChan := make(chan error, 10)
	doneChan := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(routineID int) {
			for j := 0; j < 10; j++ {
				point := ginflux.NewPoint("concurrent_test").
					AddTag("routine", fmt.Sprintf("%d", routineID)).
					AddTag("iteration", fmt.Sprintf("%d", j)).
					AddField("value", routineID*10+j).
					Build()

				ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
				err := ts.client.WriteBlocking(ctx, point)
				cancel()

				if err != nil {
					errChan <- fmt.Errorf("routine %d iteration %d failed: %w", routineID, j, err)
					return
				}
			}
			doneChan <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	completed := 0
	for completed < 10 {
		select {
		case err := <-errChan:
			return err
		case <-doneChan:
			completed++
		case <-time.After(30 * time.Second):
			return fmt.Errorf("concurrent write test timeout")
		}
	}

	log.Printf("  All 10 goroutines completed successfully")
	return nil
}

// Test33_DataRetrieval 数据检索验证
func (ts *TestSuite) Test33_DataRetrieval() error {
	// 先写入一些已知数据
	testValue := time.Now().Unix()
	point := ginflux.NewPoint("retrieval_test").
		AddTag("test_id", "unique_123").
		AddField("test_value", testValue).
		Build()

	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	err := ts.client.WriteBlocking(ctx, point)
	cancel()
	if err != nil {
		return fmt.Errorf("write test data failed: %w", err)
	}

	// 等待数据可用
	time.Sleep(1 * time.Second)

	// 查询并验证
	ctx, cancel = context.WithTimeout(ts.ctx, 10*time.Second)
	defer cancel()

	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -1m)
  |> filter(fn: (r) => r["_measurement"] == "retrieval_test")
  |> filter(fn: (r) => r["test_id"] == "unique_123")
  |> last()`, ts.config.Bucket)

	result, err := ts.client.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("query test data failed: %w", err)
	}

	found := false
	for result.Next() {
		found = true
		value := result.Record().Value()
		log.Printf("  Retrieved value: %v (expected: %d)", value, testValue)
	}

	if !found {
		return fmt.Errorf("test data not found in query results")
	}

	return nil
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 从环境变量读取配置
	config := &TestConfig{
		ServerURL:    getEnv("INFLUX_URL", "http://localhost:8086"),
		Token:        getEnv("INFLUX_TOKEN", "test-token-123456789"),
		Organization: getEnv("INFLUX_ORG", "test-org"),
		Bucket:       getEnv("INFLUX_BUCKET", "test-bucket"),
	}

	log.Println(strings.Repeat("=", 80))
	log.Println("GInflux 完整功能测试")
	log.Println(strings.Repeat("=", 80))
	log.Printf("Server: %s\n", config.ServerURL)
	log.Printf("Organization: %s\n", config.Organization)
	log.Printf("Bucket: %s\n", config.Bucket)
	log.Println(strings.Repeat("=", 80))

	// 创建测试套件
	ts := NewTestSuite(config)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n收到中断信号，正在清理...")
		if ts.client != nil {
			ts.client.Close()
		}
		ginflux.Close()
		os.Exit(0)
	}()

	// 运行所有测试
	ts.Run("01. 配置验证", ts.Test1_ConfigValidation)
	ts.Run("02. 配置链式调用", ts.Test2_ConfigChaining)
	ts.Run("03. 客户端创建", ts.Test3_ClientCreation)
	ts.Run("04. 客户端 Ping", ts.Test4_ClientPing)
	ts.Run("05. 健康检查", ts.Test5_ClientHealth)
	ts.Run("06. 就绪检查", ts.Test6_ClientReady)
	ts.Run("07. 数据点构建器", ts.Test7_PointBuilder)
	ts.Run("08. 批量数据点构建", ts.Test8_PointBuilderBatch)
	ts.Run("09. 阻塞式写入", ts.Test9_WriteBlocking)
	ts.Run("10. 批量阻塞写入", ts.Test10_WriteBatchBlocking)
	ts.Run("11. 记录阻塞写入", ts.Test11_WriteRecordBlocking)
	ts.Run("12. 非阻塞写入", ts.Test12_WriteNonBlocking)
	ts.Run("13. 查询构建器", ts.Test13_QueryBuilder)
	ts.Run("14. 查询执行", ts.Test14_QueryExecution)
	ts.Run("15. 原始查询", ts.Test15_QueryRaw)
	ts.Run("16. 聚合查询", ts.Test16_QueryBuilderAggregates)
	ts.Run("17. Bucket 列表", ts.Test17_BucketList)
	ts.Run("18. 获取 Bucket", ts.Test18_BucketGet)
	ts.Run("19. 创建/删除 Bucket", ts.Test19_BucketCreateAndDelete)
	ts.Run("20. 更新保留策略", ts.Test20_BucketUpdateRetention)
	ts.Run("21. 基础指标记录", ts.Test21_Metrics)
	ts.Run("22. 计数器指标", ts.Test22_MetricsCounter)
	ts.Run("23. 仪表盘指标", ts.Test23_MetricsGauge)
	ts.Run("24. 时间指标", ts.Test24_MetricsTiming)
	ts.Run("25. 计时器", ts.Test25_MetricsTimer)
	ts.Run("26. 带错误的计时器", ts.Test26_MetricsTimerWithError)
	ts.Run("27. 批量写入器", ts.Test27_Writer)
	ts.Run("28. 写入器批量写入", ts.Test28_WriterBatch)
	ts.Run("29. 默认客户端", ts.Test29_DefaultClient)
	ts.Run("30. 全局函数", ts.Test30_GlobalFunctions)
	ts.Run("31. 压力测试", ts.Test31_StressTest)
	ts.Run("32. 并发写入", ts.Test32_ConcurrentWrites)
	ts.Run("33. 数据检索验证", ts.Test33_DataRetrieval)

	// 清理
	if ts.client != nil {
		ts.client.Close()
	}
	ginflux.Close()

	// 打印总结
	ts.PrintSummary()
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
