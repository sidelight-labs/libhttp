package client

import "fmt"

type ResponseError struct {
	URL        string
	StatusCode int
	Body       []byte
}

func (r ResponseError) Error() string {
	return fmt.Sprintf("error calling %s, got status code %d with body:\n%s", r.URL, r.StatusCode, string(r.Body))
}
