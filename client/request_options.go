package client

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"io"
)

type RequestOption func(*requestOptions)

type requestOptions struct {
	headers    map[string]string
	retries    int
	jsonValue  interface{}
	bodyWriter io.Writer
	tracer     trace.Tracer
	context    context.Context
}

func UnmarshalJSONBody(v interface{}) RequestOption {
	return func(r *requestOptions) {
		r.jsonValue = v
	}
}

func WithHeader(name, value string) RequestOption {
	return func(r *requestOptions) {
		if r.headers == nil {
			r.headers = map[string]string{}
		}
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

func WithRetries(retries int) RequestOption {
	return func(r *requestOptions) {
		r.retries = retries
	}
}

func WithTracer(tracer trace.Tracer, ctx context.Context) RequestOption {
	return func(r *requestOptions) {
		r.tracer = tracer
		r.context = ctx
	}
}

func WriteBody(writer io.Writer) RequestOption {
	return func(r *requestOptions) {
		r.bodyWriter = writer
	}
}
