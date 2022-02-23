package client

import (
	"io"
	"time"
)

type RequestOption func(*requestOptions)

type requestOptions struct {
	headers       map[string]string
	timeout       time.Duration
	retries       int
	skipTLSVerify bool
	jsonValue     interface{}
	bodyWriter    io.Writer
}

func WithHeader(name, value string) RequestOption {
	return func(r *requestOptions) {
		r.headers[name] = value
	}
}

func WithHeaders(headers map[string]string) RequestOption {
	return func(r *requestOptions) {
		if r.headers == nil {
			r.headers = map[string]string{}
		}
		for name, value := range headers {
			r.headers[name] = value
		}
	}
}

func WithTimeout(timeout time.Duration) RequestOption {
	return func(r *requestOptions) {
		r.timeout = timeout
	}
}

func WithRetries(retries int) RequestOption {
	return func(r *requestOptions) {
		r.retries = retries
	}
}

func SkipTLSVerify(skip bool) RequestOption {
	return func(r *requestOptions) {
		r.skipTLSVerify = skip
	}
}

func UnmarshalJSONBody(v interface{}) RequestOption {
	return func(r *requestOptions) {
		r.jsonValue = v
	}
}

func WriteBody(writer io.Writer) RequestOption {
	return func(r *requestOptions) {
		r.bodyWriter = writer
	}
}
