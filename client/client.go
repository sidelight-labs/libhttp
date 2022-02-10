package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/sidelight-labs/libc/logger"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	SkipVerifyEnv = "TLS_SKIP_VERIFY"
)

type ResponseError struct {
	Endpoint   string
	StatusCode int
}

func (r ResponseError) Error() string {
	return fmt.Sprintf("error calling %s, got status code %d", r.Endpoint, r.StatusCode)
}

type Caller interface {
	Get(string) (string, error)
	GetWithRetries(string, int) (string, error)
	Post(string, string) (string, error)
	SetHeaders(map[string]string)
}

type Callout struct {
	headers map[string]string
}

// Ensure Callout implements Caller interface
var _ Caller = &Callout{}

func (c *Callout) SetHeaders(headers map[string]string) {
	c.headers = headers
}

func (c *Callout) Get(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	skipVerify := false
	if os.Getenv(SkipVerifyEnv) == "true" {
		skipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", logger.Wrap(err, fmt.Sprintf("client GET error for endpoint %s", endpoint))
	}

	defer cleanUp(resp)

	if resp.StatusCode != http.StatusOK {
		return "", ResponseError{Endpoint: endpoint, StatusCode: resp.StatusCode}
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func (c *Callout) GetWithRetries(endpoint string, retries int) (string, error) {
	var err error

	for i := 0; i <= retries; i++ {
		var response string
		response, err = c.Get(endpoint)
		if err == nil {
			return response, nil
		}

		if !errors.As(err, &ResponseError{}) {
			return "", err
		}
	}

	return "", logger.Wrap(err, fmt.Sprintf("request failed %d times", retries))
}

func (c *Callout) Post(endpoint string, body string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", logger.Wrap(err, fmt.Sprintf("client POST error for endpoint %s with body %v", endpoint, body))
	}

	defer cleanUp(resp)

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func cleanUp(resp *http.Response) {
	if resp == nil {
		return
	}

	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Error closing the reponse body: %s\n", err.Error())
	}
}
