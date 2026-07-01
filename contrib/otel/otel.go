package ginfluxotel

import (
	"net/http"

	"github.com/nicexiaonie/ginflux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WithTracing 为 ginflux HTTP 请求启用 OpenTelemetry trace。
func WithTracing(opts ...otelhttp.Option) ginflux.ClientOption {
	return ginflux.WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(base, opts...)
	})
}
