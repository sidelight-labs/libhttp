package client_test

import (
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/sidelight-labs/libhttp/client"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnitClient(t *testing.T) {
	spec.Run(t, "Client Test", testClient, spec.Report(report.Terminal{}))
}

func testClient(t *testing.T, when spec.G, it spec.S) {
	var (
		callout      client.Callout
		server       *httptest.Server
		requestCount int
	)

	it.Before(func() {
		RegisterTestingT(t)

		callout = client.Callout{}

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			requestCount++

			switch r.Method {
			case http.MethodGet:
				switch r.URL.Path {
				case "/200":
					_, _ = fmt.Fprint(w, "200")
				case "/500":
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprint(w, "500")
				case "/200forFirstThreeRequestsThen500":
					if requestCount <= 3 {
						_, _ = fmt.Fprintf(w, "200 on request %d", requestCount)
						return
					}

					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintf(w, "500 on request %d", requestCount)
				case "/500forFirstThreeRequestsThen200":
					if requestCount > 3 {
						_, _ = fmt.Fprintf(w, "200 on request %d", requestCount)
						return
					}

					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprintf(w, "500 on request %d", requestCount)
				case "/headers":
					for key, value := range r.Header {
						_, _ = fmt.Fprintf(w, "%s: %s\n", key, value)
					}
				}
			case http.MethodPost:
				switch r.URL.Path {
				case "/200":
					body, err := ioutil.ReadAll(r.Body)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = fmt.Fprintf(w, "Error reading body: %s", err.Error())
						return
					}

					_, _ = fmt.Fprintf(w, "Posted %s to 200", string(body))
				case "/500":
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = fmt.Fprint(w, "Posted to 500")
				case "/headers":
					for key, value := range r.Header {
						_, _ = fmt.Fprintf(w, "%s: %s\n", key, value)
					}
					_, _ = fmt.Fprint(w, "Posted to headers")
				}
			}
		}))

		requestCount = 0
	})

	it.After(func() {
		if server != nil {
			server.Close()
		}
	})

	when("Get", func() {
		it("returns the response body as a string", func() {
			body, err := callout.Get(server.URL + "/200")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(Equal("200"))
		})

		it("uses headers when they have been set", func() {
			callout.SetHeaders(map[string]string{
				"some-header":       "some-value",
				"some-other-header": "some-other-value",
			})

			body, err := callout.Get(server.URL + "/headers")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Some-Header: [some-value]"))
			Expect(body).To(ContainSubstring("Some-Other-Header: [some-other-value]"))
		})

		it("returns a ResponseError when the response is not 200", func() {
			url := server.URL + "/500"
			body, err := callout.Get(url)
			Expect(err).To(MatchError(client.ResponseError{Endpoint: url, StatusCode: 500}))
			Expect(body).To(BeEmpty())
		})

		it("returns an error when something else goes wrong", func() {
			body, err := callout.Get("this isn't a URL")
			Expect(err).To(HaveOccurred())
			Expect(body).To(BeEmpty())
		})
	})

	when("GetWithRetries", func() {
		it("returns the response body as a string if the request succeeds on the first try", func() {
			body, err := callout.GetWithRetries(server.URL+"/200", 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(Equal("200"))
		})

		it("returns the response body as a string if the request succeeds within the specified number of retries", func() {
			body, err := callout.GetWithRetries(server.URL+"/500forFirstThreeRequestsThen200", 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(Equal("200 on request 4"))
		})

		it("returns the last error when the number of retries is exceeded", func() {
			url := server.URL + "/500forFirstThreeRequestsThen200"
			body, err := callout.GetWithRetries(url, 2)
			Expect(err).To(MatchError(client.ResponseError{Endpoint: url, StatusCode: 500}))
			Expect(body).To(BeEmpty())
		})

		it("returns an error immediately if the error is not a ResponseError", func() {
			body, err := callout.GetWithRetries("this isn't a URL", math.MaxInt)
			Expect(err).To(HaveOccurred())
			Expect(errors.As(err, &client.ResponseError{})).To(BeFalse())
			Expect(body).To(BeEmpty())
		})
	})

	when("Post", func() {
		it("returns the response body as a string", func() {
			body, err := callout.Post(server.URL+"/200", "some-body")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(Equal("Posted some-body to 200"))
		})

		it("uses headers when they have been set", func() {
			callout.SetHeaders(map[string]string{
				"some-header":       "some-value",
				"some-other-header": "some-other-value",
			})

			body, err := callout.Post(server.URL+"/headers", "some-body")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Some-Header: [some-value]"))
			Expect(body).To(ContainSubstring("Some-Other-Header: [some-other-value]"))
			Expect(body).To(ContainSubstring("Posted to headers"))
		})
	})
}
