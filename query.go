package ginflux

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryBuilder Flux 查询构建器
type QueryBuilder struct {
	bucket      string
	measurement string
	start       string
	stop        string
	filters     []string
	fields      []string
	groupBy     []string
	aggregates  []string
	limit       int
	offset      int
	sort        string
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(bucket string) *QueryBuilder {
	return &QueryBuilder{
		bucket:  bucket,
		start:   "-1h", // 默认查询最近1小时
		filters: make([]string, 0),
		fields:  make([]string, 0),
		groupBy: make([]string, 0),
	}
}

// Measurement 设置 measurement
func (qb *QueryBuilder) Measurement(measurement string) *QueryBuilder {
	qb.measurement = measurement
	return qb
}

// Start 设置开始时间
// 支持相对时间（如 "-1h", "-30m", "-7d"）或绝对时间
func (qb *QueryBuilder) Start(start string) *QueryBuilder {
	qb.start = start
	return qb
}

// StartTime 设置开始时间（使用 time.Time）
func (qb *QueryBuilder) StartTime(t time.Time) *QueryBuilder {
	qb.start = t.Format(time.RFC3339)
	return qb
}

// Stop 设置结束时间
func (qb *QueryBuilder) Stop(stop string) *QueryBuilder {
	qb.stop = stop
	return qb
}

// StopTime 设置结束时间（使用 time.Time）
func (qb *QueryBuilder) StopTime(t time.Time) *QueryBuilder {
	qb.stop = t.Format(time.RFC3339)
	return qb
}

// Filter 添加过滤条件
func (qb *QueryBuilder) Filter(filter string) *QueryBuilder {
	qb.filters = append(qb.filters, filter)
	return qb
}

// FilterTag 按标签过滤
func (qb *QueryBuilder) FilterTag(key, value string) *QueryBuilder {
	qb.filters = append(qb.filters, fmt.Sprintf(`r["%s"] == "%s"`, key, value))
	return qb
}

// FilterField 按字段过滤
func (qb *QueryBuilder) FilterField(field string) *QueryBuilder {
	qb.fields = append(qb.fields, field)
	return qb
}

// FilterFields 按多个字段过滤
func (qb *QueryBuilder) FilterFields(fields ...string) *QueryBuilder {
	qb.fields = append(qb.fields, fields...)
	return qb
}

// GroupBy 设置分组字段
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

// Aggregate 添加聚合函数
func (qb *QueryBuilder) Aggregate(fn string) *QueryBuilder {
	qb.aggregates = append(qb.aggregates, fn)
	return qb
}

// Mean 计算平均值
func (qb *QueryBuilder) Mean() *QueryBuilder {
	return qb.Aggregate("mean()")
}

// Sum 计算总和
func (qb *QueryBuilder) Sum() *QueryBuilder {
	return qb.Aggregate("sum()")
}

// Count 计数
func (qb *QueryBuilder) Count() *QueryBuilder {
	return qb.Aggregate("count()")
}

// Max 最大值
func (qb *QueryBuilder) Max() *QueryBuilder {
	return qb.Aggregate("max()")
}

// Min 最小值
func (qb *QueryBuilder) Min() *QueryBuilder {
	return qb.Aggregate("min()")
}

// First 第一个值
func (qb *QueryBuilder) First() *QueryBuilder {
	return qb.Aggregate("first()")
}

// Last 最后一个值
func (qb *QueryBuilder) Last() *QueryBuilder {
	return qb.Aggregate("last()")
}

// Limit 限制返回数量
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

// Offset 设置偏移量
func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offset = n
	return qb
}

// Sort 设置排序
func (qb *QueryBuilder) Sort(columns ...string) *QueryBuilder {
	qb.sort = strings.Join(columns, ", ")
	return qb
}

// Build 构建 Flux 查询语句
func (qb *QueryBuilder) Build() string {
	var query strings.Builder

	// from bucket
	query.WriteString(fmt.Sprintf(`from(bucket: "%s")`, qb.bucket))

	// range
	query.WriteString(fmt.Sprintf(`\n  |> range(start: %s`, qb.start))
	if qb.stop != "" {
		query.WriteString(fmt.Sprintf(`, stop: %s`, qb.stop))
	}
	query.WriteString(")")

	// filter measurement
	if qb.measurement != "" {
		query.WriteString(fmt.Sprintf(`\n  |> filter(fn: (r) => r["_measurement"] == "%s")`, qb.measurement))
	}

	// filter fields
	if len(qb.fields) > 0 {
		fieldFilters := make([]string, len(qb.fields))
		for i, field := range qb.fields {
			fieldFilters[i] = fmt.Sprintf(`r["_field"] == "%s"`, field)
		}
		query.WriteString(fmt.Sprintf(`\n  |> filter(fn: (r) => %s)`, strings.Join(fieldFilters, " or ")))
	}

	// custom filters
	for _, filter := range qb.filters {
		query.WriteString(fmt.Sprintf(`\n  |> filter(fn: (r) => %s)`, filter))
	}

	// aggregates
	for _, agg := range qb.aggregates {
		query.WriteString(fmt.Sprintf("\n  |> %s", agg))
	}

	// group by
	if len(qb.groupBy) > 0 {
		columns := make([]string, len(qb.groupBy))
		for i, col := range qb.groupBy {
			columns[i] = fmt.Sprintf(`"%s"`, col)
		}
		query.WriteString(fmt.Sprintf("\n  |> group(columns: [%s])", strings.Join(columns, ", ")))
	}

	// sort
	if qb.sort != "" {
		query.WriteString(fmt.Sprintf("\n  |> sort(columns: [%s])", qb.sort))
	}

	// limit
	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf("\n  |> limit(n: %d", qb.limit))
		if qb.offset > 0 {
			query.WriteString(fmt.Sprintf(", offset: %d", qb.offset))
		}
		query.WriteString(")")
	}

	return query.String()
}

// Execute 执行查询
func (qb *QueryBuilder) Execute(ctx context.Context, client *Client) ([]map[string]interface{}, error) {
	query := qb.Build()
	result, err := client.Query(ctx, query)
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

// ExecuteRaw 执行原始查询
func (qb *QueryBuilder) ExecuteRaw(ctx context.Context, client *Client) (string, error) {
	query := qb.Build()
	return client.QueryRaw(ctx, query)
}
