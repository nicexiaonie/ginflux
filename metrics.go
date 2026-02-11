package ginflux

import (
	"context"
	"fmt"
	"time"
)

// Metrics 指标收集器
type Metrics struct {
	client      *Client
	measurement string
	tags        map[string]string
}

// NewMetrics 创建指标收集器
func NewMetrics(client *Client, measurement string, tags map[string]string) *Metrics {
	if tags == nil {
		tags = make(map[string]string)
	}
	return &Metrics{
		client:      client,
		measurement: measurement,
		tags:        tags,
	}
}

// Record 记录指标
func (m *Metrics) Record(fields map[string]interface{}) error {
	point := NewPoint(m.measurement).
		AddTags(m.tags).
		AddFields(fields).
		Build()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.client.WriteBlocking(ctx, point)
}

// RecordWithTags 记录带额外标签的指标
func (m *Metrics) RecordWithTags(fields map[string]interface{}, extraTags map[string]string) error {
	allTags := make(map[string]string)
	for k, v := range m.tags {
		allTags[k] = v
	}
	for k, v := range extraTags {
		allTags[k] = v
	}

	point := NewPoint(m.measurement).
		AddTags(allTags).
		AddFields(fields).
		Build()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.client.WriteBlocking(ctx, point)
}

// RecordCounter 记录计数器
func (m *Metrics) RecordCounter(name string, value int64, extraTags ...map[string]string) error {
	fields := map[string]interface{}{
		name: value,
	}

	if len(extraTags) > 0 {
		return m.RecordWithTags(fields, extraTags[0])
	}
	return m.Record(fields)
}

// RecordGauge 记录仪表盘值
func (m *Metrics) RecordGauge(name string, value float64, extraTags ...map[string]string) error {
	fields := map[string]interface{}{
		name: value,
	}

	if len(extraTags) > 0 {
		return m.RecordWithTags(fields, extraTags[0])
	}
	return m.Record(fields)
}

// RecordTiming 记录时间（毫秒）
func (m *Metrics) RecordTiming(name string, duration time.Duration, extraTags ...map[string]string) error {
	fields := map[string]interface{}{
		name: duration.Milliseconds(),
	}

	if len(extraTags) > 0 {
		return m.RecordWithTags(fields, extraTags[0])
	}
	return m.Record(fields)
}

// Timer 计时器
type Timer struct {
	metrics   *Metrics
	name      string
	startTime time.Time
	tags      map[string]string
}

// StartTimer 开始计时
func (m *Metrics) StartTimer(name string, tags ...map[string]string) *Timer {
	var extraTags map[string]string
	if len(tags) > 0 {
		extraTags = tags[0]
	}

	return &Timer{
		metrics:   m,
		name:      name,
		startTime: time.Now(),
		tags:      extraTags,
	}
}

// Stop 停止计时并记录
func (t *Timer) Stop() error {
	duration := time.Since(t.startTime)
	return t.metrics.RecordTiming(t.name, duration, t.tags)
}

// StopWithError 停止计时并记录（带错误标签）
func (t *Timer) StopWithError(err error) error {
	duration := time.Since(t.startTime)

	tags := make(map[string]string)
	for k, v := range t.tags {
		tags[k] = v
	}

	if err != nil {
		tags["error"] = "true"
		tags["error_msg"] = err.Error()
	} else {
		tags["error"] = "false"
	}

	return t.metrics.RecordTiming(t.name, duration, tags)
}

// QueryMetrics 查询指标
func (m *Metrics) QueryMetrics(ctx context.Context, start, stop string, filters map[string]string) ([]map[string]interface{}, error) {
	qb := NewQueryBuilder(m.client.config.Bucket).
		Measurement(m.measurement).
		Start(start)

	if stop != "" {
		qb.Stop(stop)
	}

	// 添加基础标签过滤
	for k, v := range m.tags {
		qb.FilterTag(k, v)
	}

	// 添加额外过滤条件
	for k, v := range filters {
		qb.FilterTag(k, v)
	}

	return qb.Execute(ctx, m.client)
}

// GetLatestValue 获取最新值
func (m *Metrics) GetLatestValue(ctx context.Context, field string, filters map[string]string) (interface{}, error) {
	qb := NewQueryBuilder(m.client.config.Bucket).
		Measurement(m.measurement).
		Start("-1h").
		FilterField(field).
		Last().
		Limit(1)

	// 添加基础标签过滤
	for k, v := range m.tags {
		qb.FilterTag(k, v)
	}

	// 添加额外过滤条件
	for k, v := range filters {
		qb.FilterTag(k, v)
	}

	records, err := qb.Execute(ctx, m.client)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no data found")
	}

	return records[0]["_value"], nil
}

// GetAggregatedValue 获取聚合值
func (m *Metrics) GetAggregatedValue(ctx context.Context, field, aggregateFunc, start, stop string, filters map[string]string) (interface{}, error) {
	qb := NewQueryBuilder(m.client.config.Bucket).
		Measurement(m.measurement).
		Start(start).
		FilterField(field).
		Aggregate(aggregateFunc)

	if stop != "" {
		qb.Stop(stop)
	}

	// 添加基础标签过滤
	for k, v := range m.tags {
		qb.FilterTag(k, v)
	}

	// 添加额外过滤条件
	for k, v := range filters {
		qb.FilterTag(k, v)
	}

	records, err := qb.Execute(ctx, m.client)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no data found")
	}

	return records[0]["_value"], nil
}
