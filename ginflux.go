package ginflux

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Version 版本号
const Version = "1.0.0"

// 全局默认客户端
var defaultClient *Client

// Connect 连接 InfluxDB 并设置为默认客户端
func Connect(config *Config) error {
	client, err := NewClient(config)
	if err != nil {
		return err
	}
	defaultClient = client
	return nil
}

// MustConnect 连接 InfluxDB，失败则 panic
func MustConnect(config *Config) {
	if err := Connect(config); err != nil {
		panic(err)
	}
}

// SetDefaultClient 设置默认客户端
func SetDefaultClient(client *Client) {
	defaultClient = client
}

// GetDefaultClient 获取默认客户端
func GetDefaultClient() *Client {
	if defaultClient == nil {
		panic("default client not initialized, call Connect() first")
	}
	return defaultClient
}

// Close 关闭默认客户端
func Close() {
	if defaultClient != nil {
		defaultClient.Close()
	}
}

// Write 使用默认客户端写入数据点（非阻塞）
func Write(point *write.Point) {
	GetDefaultClient().Write(point)
}

// WriteBlocking 使用默认客户端阻塞写入数据点
func WriteBlocking(ctx context.Context, point *write.Point) error {
	return GetDefaultClient().WriteBlocking(ctx, point)
}

// WriteBatch 使用默认客户端批量写入数据点（非阻塞）
func WriteBatch(points ...*write.Point) {
	GetDefaultClient().WriteBatch(points...)
}

// WriteBatchBlocking 使用默认客户端阻塞批量写入数据点
func WriteBatchBlocking(ctx context.Context, points ...*write.Point) error {
	return GetDefaultClient().WriteBatchBlocking(ctx, points...)
}

// WriteRecord 使用默认客户端写入行协议格式的记录（非阻塞）
func WriteRecord(record string) {
	GetDefaultClient().WriteRecord(record)
}

// WriteRecordBlocking 使用默认客户端阻塞写入行协议格式的记录
func WriteRecordBlocking(ctx context.Context, record string) error {
	return GetDefaultClient().WriteRecordBlocking(ctx, record)
}

// Query 使用默认客户端执行 Flux 查询
func Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	result, err := GetDefaultClient().Query(ctx, query)
	if err != nil {
		return nil, err
	}

	records := make([]map[string]interface{}, 0)
	for result.Next() {
		record := make(map[string]interface{})
		for k, v := range result.Record().Values() {
			record[k] = v
		}
		records = append(records, record)
	}

	if result.Err() != nil {
		return nil, result.Err()
	}

	return records, nil
}

// QueryRaw 使用默认客户端执行原始 Flux 查询
func QueryRaw(ctx context.Context, query string) (string, error) {
	return GetDefaultClient().QueryRaw(ctx, query)
}

// NewQuery 使用默认客户端创建查询构建器
func NewQuery(bucket string) *QueryBuilder {
	return NewQueryBuilder(bucket)
}

// Flush 使用默认客户端强制刷新所有待写入的数据
func Flush() {
	GetDefaultClient().Flush()
}

// Ping 使用默认客户端测试连接
func Ping(ctx context.Context) (bool, error) {
	return GetDefaultClient().Ping(ctx)
}

// Health 使用默认客户端获取健康状态
func Health(ctx context.Context) error {
	_, err := GetDefaultClient().Health(ctx)
	return err
}
