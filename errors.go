package ginflux

import (
	"errors"
)

var (
	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("client is closed")

	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid config")

	// ErrNoServerURL 缺少服务器地址
	ErrNoServerURL = errors.New("serverURL is required")

	// ErrNoToken 缺少认证令牌
	ErrNoToken = errors.New("token is required")

	// ErrNoOrganization 缺少组织名称
	ErrNoOrganization = errors.New("organization is required")

	// ErrNoBucket 缺少存储桶名称
	ErrNoBucket = errors.New("bucket is required")

	// ErrInvalidPrecision 无效的时间精度
	ErrInvalidPrecision = errors.New("invalid precision")

	// ErrNoMeasurement 缺少 measurement
	ErrNoMeasurement = errors.New("measurement is required")

	// ErrNoFields 缺少字段
	ErrNoFields = errors.New("at least one field is required")

	// ErrWriteFailed 写入失败
	ErrWriteFailed = errors.New("write failed")

	// ErrQueryFailed 查询失败
	ErrQueryFailed = errors.New("query failed")

	// ErrConnectionFailed 连接失败
	ErrConnectionFailed = errors.New("connection failed")

	// ErrDefaultClientNotInitialized 默认客户端未初始化
	ErrDefaultClientNotInitialized = errors.New("default client not initialized")
)
