package ginflux

import "net/http"

// ClientOption 配置 Client 创建选项。
type ClientOption func(*clientOptions)

type clientOptions struct {
	transportWrappers []func(http.RoundTripper) http.RoundTripper
}

// WithTransportWrapper 设置 HTTP Transport 包装函数。
func WithTransportWrapper(wrapper func(http.RoundTripper) http.RoundTripper) ClientOption {
	return func(opts *clientOptions) {
		if wrapper != nil {
			opts.transportWrappers = append(opts.transportWrappers, wrapper)
		}
	}
}

func buildClientOptions(opts ...ClientOption) clientOptions {
	var clientOpts clientOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&clientOpts)
		}
	}
	return clientOpts
}
