package client

import (
	"crypto/tls"
	"fmt"
	"github.com/sidelight-labs/libc/logger"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	ResponseError = "error calling %s, got status code %d"
	SkipVerifyEnv = "TLS_SKIP_VERIFY"
)

type Caller interface {
	Get(string) (string, error)
	Post(string, string) (string, error)
	SetHeaders(map[string]string)
}

type Callout struct {
	headers map[string]string
}

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
		return "", fmt.Errorf(ResponseError, endpoint, resp.StatusCode)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(buf), nil
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
