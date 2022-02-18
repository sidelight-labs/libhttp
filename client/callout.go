package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout             = time.Minute
	defaultDialTimeout         = 5 * time.Second
	defaultTLSHandshakeTimeout = 5 * time.Second
)

type Caller interface {
	Get(url string, options ...RequestOption) ([]byte, error)
	Head(url string, options ...RequestOption) ([]byte, error)
	Post(url, body string, options ...RequestOption) ([]byte, error)
}

type Callout struct {
	defaultHeaders map[string]string
	defaultTimeout time.Duration
	defaultRetries int
	skipTLSVerify  bool
}

// Ensure Callout implements Caller interface
var _ Caller = &Callout{}

func New(options ...CalloutOption) *Callout {
	callout := &Callout{}

	for _, option := range options {
		option(callout)
	}

	return callout
}

func (c *Callout) Get(url string, options ...RequestOption) ([]byte, error) {
	return c.buildRequestWithOptions(http.MethodGet, url, "", options...)
}

func (c *Callout) Head(url string, options ...RequestOption) ([]byte, error) {
	return c.buildRequestWithOptions(http.MethodHead, url, "", options...)
}

func (c *Callout) Post(url, body string, options ...RequestOption) ([]byte, error) {
	return c.buildRequestWithOptions(http.MethodPost, url, body, options...)
}

func (c *Callout) buildRequestWithOptions(method string, url string, reqBody string, options ...RequestOption) ([]byte, error) {
	requestOpts := &requestOptions{}
	for _, option := range options {
		option(requestOpts)
	}

	if requestOpts.retries == 0 {
		requestOpts.retries = c.defaultRetries
	}
	if requestOpts.timeout == 0 {
		if c.defaultTimeout == 0 {
			requestOpts.timeout = defaultTimeout
		} else {
			requestOpts.timeout = c.defaultTimeout
		}
	}
	if requestOpts.skipTLSVerifySet == false {
		requestOpts.skipTLSVerify = c.skipTLSVerify
	}

	httpClient := http.Client{
		Timeout: requestOpts.timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: defaultDialTimeout,
			}).DialContext,
			TLSHandshakeTimeout: defaultTLSHandshakeTimeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: requestOpts.skipTLSVerify,
			},
		},
	}

	var reqBodyReader io.Reader
	if reqBody != "" {
		reqBodyReader = strings.NewReader(reqBody)
	}
	req, err := http.NewRequest(method, url, reqBodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}
	for key, value := range requestOpts.headers {
		req.Header.Set(key, value)
	}

	var statusCode int
	var body []byte
	for i := 0; i <= requestOpts.retries; i++ {
		body, statusCode, err = c.doRequest(req, httpClient, requestOpts.bodyWriter)
		if err != nil {
			return nil, err
		}

		if statusCode >= 200 && statusCode < 300 {
			if requestOpts.jsonValue != nil {
				err = json.Unmarshal(body, requestOpts.jsonValue)
				if err != nil {
					return body, fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}
			return body, nil
		}
		if statusCode >= 300 && statusCode < 500 {
			break
		}
	}

	return nil, ResponseError{
		URL:        url,
		StatusCode: statusCode,
		Body:       body,
	}
}

func (c *Callout) doRequest(req *http.Request, httpClient http.Client, writer io.Writer) ([]byte, int, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if writer != nil {
		_, err = io.Copy(writer, resp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to copy body: %w", err)
		}

		return nil, resp.StatusCode, nil
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read body: %w", err)
		}

		return body, resp.StatusCode, nil
	}
}
