package ginflux

import (
	"fmt"
	"time"
)

// Config InfluxDB 客户端配置
type Config struct {
	// ServerURL InfluxDB 服务器地址
	// 格式: http://localhost:8086 或 https://cloud.influxdata.com
	ServerURL string

	// Token 认证令牌
	Token string

	// Organization 组织名称
	Organization string

	// Bucket 默认存储桶名称
	Bucket string

	// Precision 时间精度，默认为纳秒 (ns)
	// 可选值: ns, us, ms, s
	Precision string

	// BatchSize 批量写入大小，默认 5000
	BatchSize uint

	// FlushInterval 批量写入刷新间隔，默认 1 秒
	FlushInterval time.Duration

	// RetryInterval 重试间隔，默认 5 秒
	RetryInterval time.Duration

	// MaxRetries 最大重试次数，默认 3
	MaxRetries uint

	// MaxRetryInterval 最大重试间隔，默认 125 秒
	MaxRetryInterval time.Duration

	// ExponentialBase 指数退避基数，默认 2
	ExponentialBase uint

	// HTTPRequestTimeout HTTP 请求超时时间，默认 20 秒
	HTTPRequestTimeout time.Duration

	// UseGZip 是否启用 GZip 压缩，默认 false
	UseGZip bool

	// LogLevel 日志级别 0-4，0=无日志，1=错误，2=警告，3=信息，4=调试
	LogLevel uint

	// TLSConfig TLS 配置（可选）
	// TLSConfig *tls.Config

	// HTTPClient 自定义 HTTP 客户端（可选）
	// HTTPClient *http.Client
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig(serverURL, token, org, bucket string) *Config {
	return &Config{
		ServerURL:          serverURL,
		Token:              token,
		Organization:       org,
		Bucket:             bucket,
		Precision:          "ns",
		BatchSize:          5000,
		FlushInterval:      1 * time.Second,
		RetryInterval:      5 * time.Second,
		MaxRetries:         3,
		MaxRetryInterval:   125 * time.Second,
		ExponentialBase:    2,
		HTTPRequestTimeout: 20 * time.Second,
		UseGZip:            false,
		LogLevel:           1, // 默认只记录错误
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("serverURL is required")
	}
	if c.Token == "" {
		return fmt.Errorf("token is required")
	}
	if c.Organization == "" {
		return fmt.Errorf("organization is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	// 验证时间精度
	validPrecisions := map[string]bool{
		"ns": true, "us": true, "ms": true, "s": true,
	}
	if !validPrecisions[c.Precision] {
		return fmt.Errorf("invalid precision: %s, must be one of: ns, us, ms, s", c.Precision)
	}

	return nil
}

// WithBatchSize 设置批量写入大小
func (c *Config) WithBatchSize(size uint) *Config {
	c.BatchSize = size
	return c
}

// WithFlushInterval 设置刷新间隔
func (c *Config) WithFlushInterval(interval time.Duration) *Config {
	c.FlushInterval = interval
	return c
}

// WithRetryInterval 设置重试间隔
func (c *Config) WithRetryInterval(interval time.Duration) *Config {
	c.RetryInterval = interval
	return c
}

// WithMaxRetries 设置最大重试次数
func (c *Config) WithMaxRetries(retries uint) *Config {
	c.MaxRetries = retries
	return c
}

// WithHTTPRequestTimeout 设置 HTTP 请求超时
func (c *Config) WithHTTPRequestTimeout(timeout time.Duration) *Config {
	c.HTTPRequestTimeout = timeout
	return c
}

// WithUseGZip 设置是否启用 GZip 压缩
func (c *Config) WithUseGZip(useGZip bool) *Config {
	c.UseGZip = useGZip
	return c
}

// WithLogLevel 设置日志级别
func (c *Config) WithLogLevel(level uint) *Config {
	c.LogLevel = level
	return c
}

// WithPrecision 设置时间精度
func (c *Config) WithPrecision(precision string) *Config {
	c.Precision = precision
	return c
}
