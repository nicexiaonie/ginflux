package ginflux

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Client InfluxDB 客户端
type Client struct {
	client influxdb2.Client
	config *Config
	mu     sync.RWMutex
	closed bool
}

// NewClient 创建新的 InfluxDB 客户端
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建客户端选项
	options := influxdb2.DefaultOptions().
		SetBatchSize(config.BatchSize).
		SetFlushInterval(uint(config.FlushInterval.Milliseconds())).
		SetRetryInterval(uint(config.RetryInterval.Milliseconds())).
		SetMaxRetries(config.MaxRetries).
		SetMaxRetryInterval(uint(config.MaxRetryInterval.Milliseconds())).
		SetExponentialBase(config.ExponentialBase).
		SetUseGZip(config.UseGZip).
		SetLogLevel(config.LogLevel).
		SetPrecision(time.Nanosecond)

	// 根据配置设置时间精度
	switch config.Precision {
	case "us":
		options.SetPrecision(time.Microsecond)
	case "ms":
		options.SetPrecision(time.Millisecond)
	case "s":
		options.SetPrecision(time.Second)
	default:
		options.SetPrecision(time.Nanosecond)
	}

	// 构造统一的 HTTP Transport（对齐官方默认值，见 influxdb-client-go
	// api/http/options.go 中未设置 HTTPClient 时的默认构造）。
	// 所有请求都走这份 Transport，保证用不用 tracing 时 HTTP 参数一致。
	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	// 可选: 用户包装 Transport（典型用途是接入 tracing）。
	if config.TransportWrapper != nil {
		transport = config.TransportWrapper(transport)
	}

	// 自行构造 http.Client 并传给底层库。
	// 注意: 一旦 SetHTTPClient，底层库的 SetHTTPRequestTimeout 会失效，
	// 因此超时在这里直接设置到 client 上。
	options.SetHTTPClient(&http.Client{
		Timeout:   config.HTTPRequestTimeout,
		Transport: transport,
	})

	// 创建 InfluxDB 客户端
	client := influxdb2.NewClientWithOptions(
		config.ServerURL,
		config.Token,
		options,
	)

	return &Client{
		client: client,
		config: config,
	}, nil
}

// WriteAPI 获取非阻塞写入 API
func (c *Client) WriteAPI() api.WriteAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.WriteAPI(c.config.Organization, c.config.Bucket)
}

// WriteAPIBlocking 获取阻塞写入 API
func (c *Client) WriteAPIBlocking() api.WriteAPIBlocking {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.WriteAPIBlocking(c.config.Organization, c.config.Bucket)
}

// QueryAPI 获取查询 API
func (c *Client) QueryAPI() api.QueryAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.QueryAPI(c.config.Organization)
}

// Write 写入单个数据点（非阻塞）
func (c *Client) Write(point *write.Point) {
	writeAPI := c.WriteAPI()
	writeAPI.WritePoint(point)
}

// WriteWithBucket 写入单个数据点到指定 bucket（非阻塞）
func (c *Client) WriteWithBucket(bucket string, point *write.Point) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	writeAPI := c.client.WriteAPI(c.config.Organization, bucket)
	writeAPI.WritePoint(point)
}

// WriteBlocking 阻塞写入单个数据点
func (c *Client) WriteBlocking(ctx context.Context, point *write.Point) error {
	writeAPI := c.WriteAPIBlocking()
	return writeAPI.WritePoint(ctx, point)
}

// WriteBlockingWithBucket 阻塞写入单个数据点到指定 bucket
func (c *Client) WriteBlockingWithBucket(ctx context.Context, bucket string, point *write.Point) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	writeAPI := c.client.WriteAPIBlocking(c.config.Organization, bucket)
	return writeAPI.WritePoint(ctx, point)
}

// WriteBatch 批量写入数据点（非阻塞）
func (c *Client) WriteBatch(points ...*write.Point) {
	writeAPI := c.WriteAPI()
	for _, point := range points {
		writeAPI.WritePoint(point)
	}
}

// WriteBatchBlocking 阻塞批量写入数据点
func (c *Client) WriteBatchBlocking(ctx context.Context, points ...*write.Point) error {
	writeAPI := c.WriteAPIBlocking()
	return writeAPI.WritePoint(ctx, points...)
}

// WriteRecord 写入行协议格式的记录（非阻塞）
func (c *Client) WriteRecord(record string) {
	writeAPI := c.WriteAPI()
	writeAPI.WriteRecord(record)
}

// WriteRecordBlocking 阻塞写入行协议格式的记录
func (c *Client) WriteRecordBlocking(ctx context.Context, record string) error {
	writeAPI := c.WriteAPIBlocking()
	return writeAPI.WriteRecord(ctx, record)
}

// Query 执行 Flux 查询
func (c *Client) Query(ctx context.Context, query string) (*api.QueryTableResult, error) {
	queryAPI := c.QueryAPI()
	return queryAPI.Query(ctx, query)
}

// QueryRaw 执行原始 Flux 查询，返回原始字符串
func (c *Client) QueryRaw(ctx context.Context, query string) (string, error) {
	queryAPI := c.QueryAPI()
	return queryAPI.QueryRaw(ctx, query, nil)
}

// Flush 强制刷新所有待写入的数据
func (c *Client) Flush() {
	writeAPI := c.WriteAPI()
	writeAPI.Flush()
}

// Errors 获取写入错误通道（用于非阻塞写入）
func (c *Client) Errors() <-chan error {
	writeAPI := c.WriteAPI()
	return writeAPI.Errors()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false, fmt.Errorf("client is closed")
	}

	return c.client.Ping(ctx)
}

// Health 获取健康状态
func (c *Client) Health(ctx context.Context) (*domain.HealthCheck, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.Health(ctx)
}

// Ready 检查服务是否就绪
func (c *Client) Ready(ctx context.Context) (*domain.Ready, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.Ready(ctx)
}

// Setup 初始化 InfluxDB（创建初始用户、组织、bucket）
func (c *Client) Setup(ctx context.Context, username, password, org, bucket string, retentionPeriodHours int) (*domain.OnboardingResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	return c.client.Setup(ctx, username, password, org, bucket, retentionPeriodHours)
}

// OrganizationsAPI 获取组织管理 API
func (c *Client) OrganizationsAPI() api.OrganizationsAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.OrganizationsAPI()
}

// BucketsAPI 获取 Bucket 管理 API
func (c *Client) BucketsAPI() api.BucketsAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.BucketsAPI()
}

// UsersAPI 获取用户管理 API
func (c *Client) UsersAPI() api.UsersAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.UsersAPI()
}

// AuthorizationsAPI 获取授权管理 API
func (c *Client) AuthorizationsAPI() api.AuthorizationsAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.AuthorizationsAPI()
}

// TasksAPI 获取任务管理 API
func (c *Client) TasksAPI() api.TasksAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.TasksAPI()
}

// LabelsAPI 获取标签管理 API
func (c *Client) LabelsAPI() api.LabelsAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.LabelsAPI()
}

// DeleteAPI 获取删除 API
func (c *Client) DeleteAPI() api.DeleteAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.DeleteAPI()
}

// Config 获取配置
func (c *Client) Config() *Config {
	return c.config
}

// Close 关闭客户端
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.client.Close()
	c.closed = true
}

// IsClosed 检查客户端是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// ServerURL 获取服务器 URL
func (c *Client) ServerURL() string {
	return c.client.ServerURL()
}
