package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/nicexiaonie/ginflux"
)

// BenchmarkConfig 基准测试配置
type BenchmarkConfig struct {
	ServerURL    string
	Token        string
	Organization string
	Bucket       string
}

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	TestName       string
	TotalPoints    int
	Duration       time.Duration
	PointsPerSec   float64
	AvgLatency     time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	ErrorCount     int
}

func (br *BenchmarkResult) Print() {
	fmt.Printf("\n测试: %s\n", br.TestName)
	fmt.Printf("  总数据点: %d\n", br.TotalPoints)
	fmt.Printf("  总耗时: %.2fs\n", br.Duration.Seconds())
	fmt.Printf("  吞吐量: %.2f points/sec\n", br.PointsPerSec)
	fmt.Printf("  平均延迟: %v\n", br.AvgLatency)
	fmt.Printf("  最小延迟: %v\n", br.MinLatency)
	fmt.Printf("  最大延迟: %v\n", br.MaxLatency)
	if br.ErrorCount > 0 {
		fmt.Printf("  错误数: %d\n", br.ErrorCount)
	}
}

// 1. 单点写入基准测试
func BenchmarkSingleWrite(client *ginflux.Client, count int) *BenchmarkResult {
	ctx := context.Background()
	start := time.Now()

	var totalLatency time.Duration
	var minLatency = time.Hour
	var maxLatency time.Duration
	errorCount := 0

	for i := 0; i < count; i++ {
		point := ginflux.NewPoint("bench_single").
			AddTag("test", "single").
			AddField("value", i).
			Build()

		writeStart := time.Now()
		err := client.WriteBlocking(ctx, point)
		latency := time.Since(writeStart)

		if err != nil {
			errorCount++
		} else {
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	duration := time.Since(start)
	avgLatency := totalLatency / time.Duration(count-errorCount)

	return &BenchmarkResult{
		TestName:     "单点写入",
		TotalPoints:  count,
		Duration:     duration,
		PointsPerSec: float64(count) / duration.Seconds(),
		AvgLatency:   avgLatency,
		MinLatency:   minLatency,
		MaxLatency:   maxLatency,
		ErrorCount:   errorCount,
	}
}

// 2. 批量写入基准测试
func BenchmarkBatchWrite(client *ginflux.Client, totalPoints, batchSize int) *BenchmarkResult {
	ctx := context.Background()
	start := time.Now()

	batches := totalPoints / batchSize
	var totalLatency time.Duration
	var minLatency = time.Hour
	var maxLatency time.Duration
	errorCount := 0

	for batch := 0; batch < batches; batch++ {
		points := make([]*write.Point, batchSize)
		for i := 0; i < batchSize; i++ {
			points[i] = ginflux.NewPoint("bench_batch").
				AddTag("test", "batch").
				AddTag("batch", fmt.Sprintf("%d", batch)).
				AddField("value", batch*batchSize+i).
				Build()
		}

		writeStart := time.Now()
		err := client.WriteBatchBlocking(ctx, points...)
		latency := time.Since(writeStart)

		if err != nil {
			errorCount++
		} else {
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	duration := time.Since(start)
	avgLatency := totalLatency / time.Duration(batches-errorCount)

	return &BenchmarkResult{
		TestName:     fmt.Sprintf("批量写入 (batch=%d)", batchSize),
		TotalPoints:  totalPoints,
		Duration:     duration,
		PointsPerSec: float64(totalPoints) / duration.Seconds(),
		AvgLatency:   avgLatency,
		MinLatency:   minLatency,
		MaxLatency:   maxLatency,
		ErrorCount:   errorCount,
	}
}

// 3. 非阻塞写入基准测试
func BenchmarkAsyncWrite(client *ginflux.Client, count int) *BenchmarkResult {
	start := time.Now()

	for i := 0; i < count; i++ {
		point := ginflux.NewPoint("bench_async").
			AddTag("test", "async").
			AddField("value", i).
			Build()
		client.Write(point)
	}

	client.Flush()
	duration := time.Since(start)

	return &BenchmarkResult{
		TestName:     "非阻塞写入",
		TotalPoints:  count,
		Duration:     duration,
		PointsPerSec: float64(count) / duration.Seconds(),
		ErrorCount:   0,
	}
}

// 4. Writer 批量写入基准测试
func BenchmarkWriterWrite(client *ginflux.Client, bucket string, totalPoints, batchSize int, interval time.Duration) *BenchmarkResult {
	start := time.Now()

	writer := ginflux.NewWriter(client, bucket, batchSize, interval)

	for i := 0; i < totalPoints; i++ {
		point := ginflux.NewPoint("bench_writer").
			AddTag("test", "writer").
			AddField("value", i).
			Build()
		writer.Write(point)
	}

	writer.Close()
	duration := time.Since(start)

	return &BenchmarkResult{
		TestName:     fmt.Sprintf("Writer写入 (batch=%d, interval=%v)", batchSize, interval),
		TotalPoints:  totalPoints,
		Duration:     duration,
		PointsPerSec: float64(totalPoints) / duration.Seconds(),
		ErrorCount:   0,
	}
}

// 5. 并发写入基准测试
func BenchmarkConcurrentWrite(client *ginflux.Client, totalPoints, goroutines int) *BenchmarkResult {
	ctx := context.Background()
	start := time.Now()

	pointsPerGoroutine := totalPoints / goroutines
	errChan := make(chan error, goroutines)
	doneChan := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < pointsPerGoroutine; i++ {
				point := ginflux.NewPoint("bench_concurrent").
					AddTag("test", "concurrent").
					AddTag("goroutine", fmt.Sprintf("%d", id)).
					AddField("value", id*pointsPerGoroutine+i).
					Build()

				err := client.WriteBlocking(ctx, point)
				if err != nil {
					errChan <- err
					return
				}
			}
			doneChan <- true
		}(g)
	}

	errorCount := 0
	completed := 0
	for completed < goroutines {
		select {
		case <-errChan:
			errorCount++
			completed++
		case <-doneChan:
			completed++
		}
	}

	duration := time.Since(start)

	return &BenchmarkResult{
		TestName:     fmt.Sprintf("并发写入 (%d goroutines)", goroutines),
		TotalPoints:  totalPoints,
		Duration:     duration,
		PointsPerSec: float64(totalPoints) / duration.Seconds(),
		ErrorCount:   errorCount,
	}
}

// 6. 查询性能基准测试
func BenchmarkQuery(client *ginflux.Client, bucket string, iterations int) *BenchmarkResult {
	ctx := context.Background()
	start := time.Now()

	var totalLatency time.Duration
	var minLatency = time.Hour
	var maxLatency time.Duration
	errorCount := 0

	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "bench_batch")
  |> limit(n: 100)`, bucket)

	for i := 0; i < iterations; i++ {
		queryStart := time.Now()
		_, err := client.Query(ctx, query)
		latency := time.Since(queryStart)

		if err != nil {
			errorCount++
		} else {
			totalLatency += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}
	}

	duration := time.Since(start)
	avgLatency := totalLatency / time.Duration(iterations-errorCount)

	return &BenchmarkResult{
		TestName:     "查询性能",
		TotalPoints:  iterations,
		Duration:     duration,
		PointsPerSec: float64(iterations) / duration.Seconds(),
		AvgLatency:   avgLatency,
		MinLatency:   minLatency,
		MaxLatency:   maxLatency,
		ErrorCount:   errorCount,
	}
}

func main() {
	log.SetFlags(log.LstdFlags)

	// 从环境变量读取配置
	config := &BenchmarkConfig{
		ServerURL:    getEnv("INFLUX_URL", "http://localhost:8086"),
		Token:        getEnv("INFLUX_TOKEN", "your-token-here"),
		Organization: getEnv("INFLUX_ORG", "your-org"),
		Bucket:       getEnv("INFLUX_BUCKET", "your-bucket"),
	}

	fmt.Println("================================================================================")
	fmt.Println("GInflux 性能基准测试")
	fmt.Println("================================================================================")
	fmt.Printf("Server: %s\n", config.ServerURL)
	fmt.Printf("Organization: %s\n", config.Organization)
	fmt.Printf("Bucket: %s\n", config.Bucket)
	fmt.Println("================================================================================")

	// 创建客户端
	clientConfig := ginflux.NewDefaultConfig(
		config.ServerURL,
		config.Token,
		config.Organization,
		config.Bucket,
	)

	client, err := ginflux.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, err := client.Ping(ctx)
	if err != nil || !ok {
		log.Fatalf("Failed to connect to InfluxDB: %v", err)
	}

	fmt.Println("\n开始性能测试...\n")

	results := make([]*BenchmarkResult, 0)

	// 1. 单点写入 (100 个点)
	fmt.Println("[ 1/7 ] 运行单点写入基准测试...")
	results = append(results, BenchmarkSingleWrite(client, 100))

	// 2. 批量写入 - 小批量 (1000 个点, batch=50)
	fmt.Println("[ 2/7 ] 运行小批量写入基准测试...")
	results = append(results, BenchmarkBatchWrite(client, 1000, 50))

	// 3. 批量写入 - 中批量 (1000 个点, batch=100)
	fmt.Println("[ 3/7 ] 运行中批量写入基准测试...")
	results = append(results, BenchmarkBatchWrite(client, 1000, 100))

	// 4. 批量写入 - 大批量 (1000 个点, batch=500)
	fmt.Println("[ 4/7 ] 运行大批量写入基准测试...")
	results = append(results, BenchmarkBatchWrite(client, 1000, 500))

	// 5. 非阻塞写入 (5000 个点)
	fmt.Println("[ 5/7 ] 运行非阻塞写入基准测试...")
	results = append(results, BenchmarkAsyncWrite(client, 5000))

	// 6. 并发写入 (1000 个点, 10 goroutines)
	fmt.Println("[ 6/7 ] 运行并发写入基准测试...")
	results = append(results, BenchmarkConcurrentWrite(client, 1000, 10))

	// 7. 查询性能 (50 次查询)
	fmt.Println("[ 7/7 ] 运行查询性能基准测试...")
	results = append(results, BenchmarkQuery(client, config.Bucket, 50))

	// 打印结果
	fmt.Println("\n================================================================================")
	fmt.Println("基准测试结果")
	fmt.Println("================================================================================")

	for _, result := range results {
		result.Print()
	}

	// 生成总结
	fmt.Println("\n================================================================================")
	fmt.Println("性能总结")
	fmt.Println("================================================================================")

	// 找出最快的写入方式
	var fastestWrite *BenchmarkResult
	for _, r := range results {
		if r.TestName != "查询性能" {
			if fastestWrite == nil || r.PointsPerSec > fastestWrite.PointsPerSec {
				fastestWrite = r
			}
		}
	}

	if fastestWrite != nil {
		fmt.Printf("最快的写入方式: %s (%.2f points/sec)\n",
			fastestWrite.TestName, fastestWrite.PointsPerSec)
	}

	// 计算总数据点
	totalPoints := 0
	for _, r := range results {
		if r.TestName != "查询性能" {
			totalPoints += r.TotalPoints
		}
	}
	fmt.Printf("总写入数据点: %d\n", totalPoints)

	fmt.Println("================================================================================")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
