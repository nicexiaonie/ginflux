package ginfluxotel

import (
	"net/http"

	"github.com/nicexiaonie/ginflux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WithTracing 为 ginflux HTTP 请求启用 OpenTelemetry trace。
func WithTracing(opts ...otelhttp.Option) ginflux.ClientOption {
	transportOpts := append([]otelhttp.Option{
		otelhttp.WithSpanNameFormatter(defaultSpanNameFormatter),
	}, opts...)

	return ginflux.WithTransportWrapper(func(base http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(base, transportOpts...)
	})
}

func defaultSpanNameFormatter(_ string, req *http.Request) string {
	return "InfluxDB " + req.Method + " " + req.URL.Path
}
