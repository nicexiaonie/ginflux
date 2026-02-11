package ginflux

import (
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// PointBuilder 数据点构建器
type PointBuilder struct {
	measurement string
	tags        map[string]string
	fields      map[string]interface{}
	timestamp   time.Time
}

// NewPoint 创建新的数据点构建器
func NewPoint(measurement string) *PointBuilder {
	return &PointBuilder{
		measurement: measurement,
		tags:        make(map[string]string),
		fields:      make(map[string]interface{}),
		timestamp:   time.Now(),
	}
}

// AddTag 添加标签
func (pb *PointBuilder) AddTag(key, value string) *PointBuilder {
	pb.tags[key] = value
	return pb
}

// AddTags 批量添加标签
func (pb *PointBuilder) AddTags(tags map[string]string) *PointBuilder {
	for k, v := range tags {
		pb.tags[k] = v
	}
	return pb
}

// AddField 添加字段
func (pb *PointBuilder) AddField(key string, value interface{}) *PointBuilder {
	pb.fields[key] = value
	return pb
}

// AddFields 批量添加字段
func (pb *PointBuilder) AddFields(fields map[string]interface{}) *PointBuilder {
	for k, v := range fields {
		pb.fields[k] = v
	}
	return pb
}

// SetTimestamp 设置时间戳
func (pb *PointBuilder) SetTimestamp(t time.Time) *PointBuilder {
	pb.timestamp = t
	return pb
}

// Build 构建数据点
func (pb *PointBuilder) Build() *write.Point {
	point := write.NewPoint(pb.measurement, pb.tags, pb.fields, pb.timestamp)
	return point
}

// BuildWithTime 构建带指定时间的数据点
func BuildPoint(measurement string, tags map[string]string, fields map[string]interface{}, t time.Time) *write.Point {
	return write.NewPoint(measurement, tags, fields, t)
}

// BuildPointNow 构建当前时间的数据点
func BuildPointNow(measurement string, tags map[string]string, fields map[string]interface{}) *write.Point {
	return write.NewPoint(measurement, tags, fields, time.Now())
}
