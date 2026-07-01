package ginflux

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)

	if err := config.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Test invalid configs
	invalidConfigs := []*Config{
		{ServerURL: "", Token: "token", Organization: "org", Bucket: "bucket"},
		{ServerURL: "url", Token: "", Organization: "org", Bucket: "bucket"},
		{ServerURL: "url", Token: "token", Organization: "", Bucket: "bucket"},
		{ServerURL: "url", Token: "token", Organization: "org", Bucket: ""},
	}

	for i, cfg := range invalidConfigs {
		if err := cfg.Validate(); err == nil {
			t.Errorf("Invalid config %d should have failed validation", i)
		}
	}
}

func TestConfigChaining(t *testing.T) {
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	).WithBatchSize(1000).
		WithFlushInterval(5 * time.Second).
		WithUseGZip(true).
		WithLogLevel(2)

	if config.BatchSize != 1000 {
		t.Errorf("Expected BatchSize 1000, got %d", config.BatchSize)
	}
	if config.FlushInterval != 5*time.Second {
		t.Errorf("Expected FlushInterval 5s, got %v", config.FlushInterval)
	}
	if !config.UseGZip {
		t.Error("Expected UseGZip to be true")
	}
	if config.LogLevel != 2 {
		t.Errorf("Expected LogLevel 2, got %d", config.LogLevel)
	}
}

func TestPointBuilder(t *testing.T) {
	point := NewPoint("test_measurement").
		AddTag("tag1", "value1").
		AddTag("tag2", "value2").
		AddField("field1", 123).
		AddField("field2", 45.67).
		AddField("field3", "string_value").
		Build()

	if point == nil {
		t.Error("Point should not be nil")
	}
}

func TestPointBuilderBatch(t *testing.T) {
	tags := map[string]string{
		"host":       "server01",
		"region":     "us-west",
		"datacenter": "dc1",
	}

	fields := map[string]interface{}{
		"cpu":    75.5,
		"memory": 8192,
		"disk":   "healthy",
	}

	point := NewPoint("system_metrics").
		AddTags(tags).
		AddFields(fields).
		SetTimestamp(time.Now()).
		Build()

	if point == nil {
		t.Error("Point should not be nil")
	}
}

func TestQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder("test-bucket").
		Measurement("cpu").
		Start("-1h").
		Stop("now()").
		FilterTag("host", "server01").
		FilterField("usage_idle").
		Mean().
		GroupBy("host", "region").
		Limit(100)

	query := qb.Build()

	if query == "" {
		t.Error("Query should not be empty")
	}

	// Check if query contains expected parts
	expectedParts := []string{
		`from(bucket: "test-bucket")`,
		`range(start: -1h, stop: now())`,
		`r["_measurement"] == "cpu"`,
		`r["host"] == "server01"`,
		`r["_field"] == "usage_idle"`,
		"mean()",
		"limit(n: 100",
	}

	for _, part := range expectedParts {
		if !contains(query, part) {
			t.Errorf("Query should contain '%s'\nGot: %s", part, query)
		}
	}
}

func TestQueryBuilderMultipleFields(t *testing.T) {
	qb := NewQueryBuilder("test-bucket").
		Measurement("system").
		Start("-30m").
		FilterFields("cpu", "memory", "disk")

	query := qb.Build()

	if !contains(query, `r["_field"] == "cpu"`) {
		t.Error("Query should filter cpu field")
	}
	if !contains(query, `r["_field"] == "memory"`) {
		t.Error("Query should filter memory field")
	}
	if !contains(query, `r["_field"] == "disk"`) {
		t.Error("Query should filter disk field")
	}
}

func TestQueryBuilderAggregates(t *testing.T) {
	tests := []struct {
		name     string
		builder  func(*QueryBuilder) *QueryBuilder
		expected string
	}{
		{"Mean", func(qb *QueryBuilder) *QueryBuilder { return qb.Mean() }, "mean()"},
		{"Sum", func(qb *QueryBuilder) *QueryBuilder { return qb.Sum() }, "sum()"},
		{"Count", func(qb *QueryBuilder) *QueryBuilder { return qb.Count() }, "count()"},
		{"Max", func(qb *QueryBuilder) *QueryBuilder { return qb.Max() }, "max()"},
		{"Min", func(qb *QueryBuilder) *QueryBuilder { return qb.Min() }, "min()"},
		{"First", func(qb *QueryBuilder) *QueryBuilder { return qb.First() }, "first()"},
		{"Last", func(qb *QueryBuilder) *QueryBuilder { return qb.Last() }, "last()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder("test-bucket").
				Measurement("test").
				Start("-1h")
			qb = tt.builder(qb)
			query := qb.Build()

			if !contains(query, tt.expected) {
				t.Errorf("Query should contain '%s'\nGot: %s", tt.expected, query)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	// Note: This test doesn't actually connect to InfluxDB
	// It just tests the Metrics structure creation
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)

	client, err := NewClient(config)
	if err != nil {
		t.Skipf("Skipping test, cannot create client: %v", err)
		return
	}
	defer client.Close()

	metrics := NewMetrics(client, "test_metrics", map[string]string{
		"service": "test-service",
		"env":     "test",
	})

	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if metrics.measurement != "test_metrics" {
		t.Errorf("Expected measurement 'test_metrics', got '%s'", metrics.measurement)
	}

	if metrics.tags["service"] != "test-service" {
		t.Error("Tags not set correctly")
	}
}

func TestTimer(t *testing.T) {
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)

	client, err := NewClient(config)
	if err != nil {
		t.Skipf("Skipping test, cannot create client: %v", err)
		return
	}
	defer client.Close()

	metrics := NewMetrics(client, "test_metrics", nil)
	timer := metrics.StartTimer("test_operation")

	if timer == nil {
		t.Error("Timer should not be nil")
	}

	time.Sleep(10 * time.Millisecond)

	// Note: This will fail if InfluxDB is not running, but tests the structure
	_ = timer.Stop()
}

func TestBuildPoint(t *testing.T) {
	tags := map[string]string{"host": "server01"}
	fields := map[string]interface{}{"value": 123}
	now := time.Now()

	point := BuildPoint("test", tags, fields, now)
	if point == nil {
		t.Error("Point should not be nil")
	}

	point2 := BuildPointNow("test", tags, fields)
	if point2 == nil {
		t.Error("Point should not be nil")
	}
}

func TestClientCreation(t *testing.T) {
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)

	client, err := NewClient(config)
	if err != nil {
		t.Skipf("Skipping test, cannot create client: %v", err)
		return
	}

	if client == nil {
		t.Error("Client should not be nil")
	}

	if client.IsClosed() {
		t.Error("Client should not be closed initially")
	}

	client.Close()

	if !client.IsClosed() {
		t.Error("Client should be closed after Close()")
	}
}

func TestClientPing(t *testing.T) {
	config := NewDefaultConfig(
		"http://localhost:8086",
		"test-token",
		"test-org",
		"test-bucket",
	)

	client, err := NewClient(config)
	if err != nil {
		t.Skipf("Skipping test, cannot create client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	_, err = client.Ping(ctx)
	if err != nil {
		t.Logf("Ping failed (InfluxDB may not be running): %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// roundTripperFunc 适配函数为 http.RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestTransportWrapperIsUsed 验证 TransportWrapper 包装后的 Transport
// 确实被实际请求路径使用。
func TestTransportWrapperIsUsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var called int32
	config := NewDefaultConfig(server.URL, "token", "org", "bucket").
		WithMaxRetries(0).
		WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(&called, 1)
				return base.RoundTrip(req)
			})
		})

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	point := NewPoint("test").AddField("value", 1).Build()
	if err := client.WriteBlocking(context.Background(), point); err != nil {
		t.Fatalf("WriteBlocking failed: %v", err)
	}

	if atomic.LoadInt32(&called) == 0 {
		t.Error("Wrapped transport was not used for the request")
	}
}

func TestClientOptionTransportWrappersAreComposedAfterConfigWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var order []string
	config := NewDefaultConfig(server.URL, "token", "org", "bucket").
		WithMaxRetries(0).
		WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, "config")
				return base.RoundTrip(req)
			})
		})

	client, err := NewClient(config,
		WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, "option1")
				return base.RoundTrip(req)
			})
		}),
		WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, "option2")
				return base.RoundTrip(req)
			})
		}),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	point := NewPoint("test").AddField("value", 1).Build()
	if err := client.WriteBlocking(context.Background(), point); err != nil {
		t.Fatalf("WriteBlocking failed: %v", err)
	}

	expected := []string{"option2", "option1", "config"}
	if len(order) != len(expected) {
		t.Fatalf("expected wrapper order %v, got %v", expected, order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("expected wrapper order %v, got %v", expected, order)
		}
	}
}

// TestHTTPRequestTimeoutApplied 验证 HTTPRequestTimeout 确实生效。
// 服务端故意延迟，超时应触发请求失败。
func TestHTTPRequestTimeoutApplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := NewDefaultConfig(server.URL, "token", "org", "bucket").
		WithMaxRetries(0).
		WithHTTPRequestTimeout(50 * time.Millisecond)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	point := NewPoint("test").AddField("value", 1).Build()
	err = client.WriteBlocking(context.Background(), point)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
