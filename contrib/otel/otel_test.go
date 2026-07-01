package ginfluxotel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/nicexiaonie/ginflux"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWithTracingUsesWrappedTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var called int32
	config := ginflux.NewDefaultConfig(server.URL, "token", "org", "bucket").
		WithMaxRetries(0)

	client, err := ginflux.NewClient(config,
		ginflux.WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
			return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(&called, 1)
				return base.RoundTrip(req)
			})
		}),
		WithTracing(),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	point := ginflux.NewPoint("test").AddField("value", 1).Build()
	if err := client.WriteBlocking(context.Background(), point); err != nil {
		t.Fatalf("WriteBlocking failed: %v", err)
	}

	if atomic.LoadInt32(&called) == 0 {
		t.Error("wrapped transport was not used")
	}
}
