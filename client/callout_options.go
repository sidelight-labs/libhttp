package client

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type CalloutOption func(*Callout)

func DefaultSkipTLSVerify(skipTLSVerify bool) CalloutOption {
	return func(c *Callout) {
		c.skipTLSVerify = skipTLSVerify
	}
}

func WithDefaultHeader(name, value string) CalloutOption {
	return func(c *Callout) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = map[string]string{}
		}
		c.defaultHeaders[name] = value
	}
}

func WithDefaultHeaders(headers map[string]string) CalloutOption {
	return func(c *Callout) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = map[string]string{}
		}
		for name, value := range headers {
			c.defaultHeaders[name] = value
		}
	}
}

func WithDefaultRetries(retries int) CalloutOption {
	return func(c *Callout) {
		c.defaultRetries = retries
	}
}

func WithDefaultTimeout(timeout time.Duration) CalloutOption {
	return func(c *Callout) {
		c.defaultTimeout = timeout
	}
}

func WithDefaultTracer(tracer trace.Tracer, ctx context.Context) CalloutOption {
	return func(c *Callout) {
		c.defaultTracer = tracer
		c.defaultContext = ctx
	}
}
