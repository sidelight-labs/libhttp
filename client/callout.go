package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
	getOptions := &requestOptions{}
	for _, option := range options {
		option(getOptions)
	}
	if getOptions.retries == 0 {
		getOptions.retries = c.defaultRetries
	}

	httpClient := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}
	for key, value := range getOptions.headers {
		req.Header.Set(key, value)
	}

	var statusCode int
	var body []byte
	for i := 0; i <= getOptions.retries; i++ {
		body, statusCode, err = c.doRequest(req, httpClient, getOptions.bodyWriter)
		if err != nil {
			return nil, err
		}

		if statusCode >= 200 && statusCode < 300 {
			if getOptions.jsonValue != nil {
				err = json.Unmarshal(body, getOptions.jsonValue)
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

func (c *Callout) Head(url string, options ...RequestOption) ([]byte, error) {
	return nil, nil
}

func (c *Callout) Post(url, body string, options ...RequestOption) ([]byte, error) {
	return nil, nil
}
