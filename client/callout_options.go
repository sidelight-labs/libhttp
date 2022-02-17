package client

import (
	"time"
)

type CalloutOption func(*Callout)

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

func WithDefaultTimeout(timeout time.Duration) CalloutOption {
	return func(c *Callout) {
		c.defaultTimeout = timeout
	}
}

func WithDefaultRetries(retries int) CalloutOption {
	return func(c *Callout) {
		c.defaultRetries = retries
	}
}

func WithSkipTLSVerify(skipTLSVerify bool) CalloutOption {
	return func(c *Callout) {
		c.skipTLSVerify = skipTLSVerify
	}
}
