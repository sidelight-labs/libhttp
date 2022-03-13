package client_test

import (
	"bytes"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/sidelight-labs/libhttp/client"
	"github.com/sidelight-labs/libhttp/client/mocks"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUnitClient(t *testing.T) {
	spec.Run(t, "Client Test", testClient, spec.Report(report.Terminal{}))
}

func testClient(t *testing.T, when spec.G, it spec.S) {
	var (
		server       *httptest.Server
		tlsServer    *httptest.Server
		requestCount int
	)

	it.Before(func() {
		RegisterTestingT(t)

		requestCount = 0
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			requestCount++

			switch r.URL.Path {
			case "/print-request":
				_ = r.Write(w)
			case "/echo":
				if r.Method == http.MethodGet || r.Method == http.MethodHead {
					value := r.URL.Query().Get("body")

					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprint(w, value)
				} else if r.Method == http.MethodPost {
					body, err := ioutil.ReadAll(r.Body)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = fmt.Fprintf(w, "failed to read body: %s", err.Error())
						return
					}

					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprint(w, string(body))
				} else {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprintf(w, "Method must be GET, HEAD, or POST")
				}
			case "/sleep":
				duration, err := time.ParseDuration(r.URL.Query().Get("duration"))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprintf(w, "failed to parse duration: %s", err.Error())
					return
				}

				time.Sleep(duration)

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, "slept for %v", duration)
			case "/200":
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, "200")
			case "/400":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(w, "400")
			case "/500":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "500")
			case "/500forFirstThreeRequestsThen200":
				if requestCount > 3 {
					_, _ = fmt.Fprintf(w, "200 on request %d", requestCount)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "500 on request %d", requestCount)
			}
		}))
		tlsServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.Write(w)
		}))
	})

	it.After(func() {
		if server != nil {
			server.Close()
		}
		if tlsServer != nil {
			tlsServer.Close()
		}
	})

	when("Get", func() {
		it("returns the response body", func() {
			callout := client.New()

			url := server.URL + "/200"
			body, err := callout.Get(url)

			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("200"))

		})

		it("returns a ResponseError when the response is not 200", func() {
			callout := client.New()

			url := server.URL + "/500"
			body, err := callout.Get(url)

			Expect(err).To(MatchError(client.ResponseError{
				URL:        server.URL + "/500",
				StatusCode: 500,
				Body:       []byte("500"),
			}))
			Expect(body).To(BeEmpty())
		})

		it("returns an error when something else goes wrong", func() {
			callout := client.New()

			body, err := callout.Get("this isn't a URL")

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(BeAssignableToTypeOf(client.ResponseError{}))
			Expect(body).To(BeEmpty())
		})

		when("WithHeader(s)", func() {
			it("combines the default headers and request headers", func() {
				callout := client.New(client.WithDefaultHeaders(map[string]string{
					"header1": "value1",
					"header2": "value2",
				}), client.WithDefaultHeader(
					"header3", "value3",
				))

				url := server.URL + "/print-request"
				body, err := callout.Get(url, client.WithHeaders(map[string]string{
					"header1": "overridden by request",
					"header4": "value4",
				}), client.WithHeader(
					"header5", "value5",
				))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Header1: overridden by request"))
				Expect(string(body)).To(ContainSubstring("Header2: value2"))
				Expect(string(body)).To(ContainSubstring("Header3: value3"))
				Expect(string(body)).To(ContainSubstring("Header4: value4"))
				Expect(string(body)).To(ContainSubstring("Header5: value5"))
			})
		})

		when("WithTracing", func() {
			var (
				mockCtrl   *gomock.Controller
				mockTracer *mocks.MockTracer
				mockSpan   *mocks.MockSpan
				url        string
			)

			it.Before(func() {
				mockCtrl = gomock.NewController(t)
				mockTracer = mocks.NewMockTracer(mockCtrl)
				mockSpan = mocks.NewMockSpan(mockCtrl)
				url = fmt.Sprintf("%s/200s", server.URL)
			})

			it.After(func() {
				mockCtrl.Finish()
			})

			it("calls the trace method with the expected argument", func() {
				mockTracer.EXPECT().Trace("/200s").Return(mockSpan).Times(1)
				mockSpan.EXPECT().End().Times(1)

				callout := client.New(client.WithDefaultTracer(mockTracer))
				_, err := callout.Get(url)
				Expect(err).NotTo(HaveOccurred())
			})
			it("doesn't trace when WithTracer is not called", func() {
				mockTracer.EXPECT().Trace("/200s").Return(mockSpan).Times(0)
				mockSpan.EXPECT().End().Times(0)

				callout := client.New()
				_, err := callout.Get(url)
				Expect(err).NotTo(HaveOccurred())
			})
			it("overwrites the default setting", func() {
				mockTracer.EXPECT().Trace("/200s").Return(mockSpan).Times(1)
				mockSpan.EXPECT().End().Times(1)

				callout := client.New()
				_, err := callout.Get(url, client.WithTracer(mockTracer))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("WithTimeout", func() {
			it("returns an error if the request exceeds the client timeout", func() {
				timeout := time.Millisecond
				callout := client.New(client.WithDefaultTimeout(timeout))

				url := fmt.Sprintf("%s/sleep?duration=%s", server.URL, timeout*10)
				_, err := callout.Get(url)
				Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
			})
		})

		when("WithRetries", func() {
			it("retries on a 5XX response until a 2XX response", func() {
				callout := client.New()

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Get(url, client.WithRetries(5))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})

			it("returns the last error when the number of retries is exceeded", func() {
				callout := client.New()

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Get(url, client.WithRetries(2))

				Expect(err).To(MatchError(client.ResponseError{
					URL:        url,
					StatusCode: 500,
					Body:       []byte("500 on request 3"),
				}))
				Expect(body).To(BeEmpty())
			})

			it("does not retry on a 4XX response", func() {
				callout := client.New()

				url := server.URL + "/400"
				body, err := callout.Get(url, client.WithRetries(2))

				Expect(err).To(MatchError(client.ResponseError{
					URL:        url,
					StatusCode: 400,
					Body:       []byte("400"),
				}))
				Expect(body).To(BeEmpty())
				Expect(requestCount).To(Equal(1))
			})

			it("uses retries set on the callout", func() {
				callout := client.New(client.WithDefaultRetries(3))

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Get(url)

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})

			it("prefers the retries on the request", func() {
				callout := client.New(client.WithDefaultRetries(1))

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Get(url, client.WithRetries(3))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})
		})

		when("UnmarshalJSON", func() {
			it("unmarshals the response body to the given value", func() {
				callout := client.New()

				url := server.URL + `/echo?body={"key":"value"}`
				var value struct {
					Key string
				}
				body, err := callout.Get(url, client.UnmarshalJSONBody(&value))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(MatchJSON(`{"key":"value"}`))
				Expect(value.Key).To(Equal("value"))
			})
		})

		when("WriteBody", func() {
			it("writes the response body to the given writer but does not return it", func() {
				callout := client.New()

				url := server.URL + "/200"
				var buf bytes.Buffer
				body, err := callout.Get(url, client.WriteBody(&buf))

				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(BeEmpty())
				Expect(buf.String()).To(Equal("200"))
			})
		})

		when("SkipTLSVerify", func() {
			it("does not skip TLS verification by default", func() {
				callout := client.New()

				_, err := callout.Get(tlsServer.URL)
				Expect(err).To(HaveOccurred())
			})

			it("skips TLS verification when set to true on the client", func() {
				callout := client.New(client.DefaultSkipTLSVerify(true))

				_, err := callout.Get(tlsServer.URL)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	when("Post", func() {
		it("returns the response body", func() {
			callout := client.New()

			url := server.URL + "/echo"
			body, err := callout.Post(url, "foobar")

			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("foobar"))
		})

		it("returns a ResponseError when the response is not 200", func() {
			callout := client.New()

			url := server.URL + "/500"
			body, err := callout.Post(url, "body")

			Expect(err).To(MatchError(client.ResponseError{
				URL:        server.URL + "/500",
				StatusCode: 500,
				Body:       []byte("500"),
			}))
			Expect(body).To(BeEmpty())
		})

		it("returns an error when something else goes wrong", func() {
			callout := client.New()

			body, err := callout.Post("this isn't a URL", "body")

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(BeAssignableToTypeOf(client.ResponseError{}))
			Expect(body).To(BeEmpty())
		})

		when("WithHeader(s)", func() {
			it("combines the default headers and request headers", func() {
				callout := client.New(client.WithDefaultHeaders(map[string]string{
					"header1": "value1",
					"header2": "value2",
				}), client.WithDefaultHeader(
					"header3", "value3",
				))

				url := server.URL + "/print-request"
				body, err := callout.Post(url, "body", client.WithHeaders(map[string]string{
					"header1": "overridden by request",
					"header4": "value4",
				}), client.WithHeader(
					"header5", "value5",
				))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(ContainSubstring("Header1: overridden by request"))
				Expect(string(body)).To(ContainSubstring("Header2: value2"))
				Expect(string(body)).To(ContainSubstring("Header3: value3"))
				Expect(string(body)).To(ContainSubstring("Header4: value4"))
				Expect(string(body)).To(ContainSubstring("Header5: value5"))
			})
		})

		when("WithTimeout", func() {
			it("returns an error if the request exceeds the client timeout", func() {
				timeout := time.Millisecond
				callout := client.New(client.WithDefaultTimeout(timeout))

				url := fmt.Sprintf("%s/sleep?duration=%s", server.URL, timeout*10)
				_, err := callout.Post(url, "")
				Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
			})
		})

		when("WithRetries", func() {
			it("retries on a 5XX response until a 2XX response", func() {
				callout := client.New()

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Post(url, "body", client.WithRetries(5))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})

			it("returns the last error when the number of retries is exceeded", func() {
				callout := client.New()

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Post(url, "body", client.WithRetries(2))

				Expect(err).To(MatchError(client.ResponseError{
					URL:        url,
					StatusCode: 500,
					Body:       []byte("500 on request 3"),
				}))
				Expect(body).To(BeEmpty())
			})

			it("does not retry on a 4XX response", func() {
				callout := client.New()

				url := server.URL + "/400"
				body, err := callout.Post(url, "body", client.WithRetries(2))

				Expect(err).To(MatchError(client.ResponseError{
					URL:        url,
					StatusCode: 400,
					Body:       []byte("400"),
				}))
				Expect(body).To(BeEmpty())
				Expect(requestCount).To(Equal(1))
			})

			it("uses retries set on the callout", func() {
				callout := client.New(client.WithDefaultRetries(3))

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Post(url, "body")

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})

			it("prefers the retries on the request", func() {
				callout := client.New(client.WithDefaultRetries(1))

				url := server.URL + "/500forFirstThreeRequestsThen200"
				body, err := callout.Post(url, "body", client.WithRetries(3))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("200 on request 4"))
			})
		})

		when("UnmarshalJSON", func() {
			it("unmarshals the response body to the given value", func() {
				callout := client.New()

				url := server.URL + `/echo`
				var value struct {
					Key string
				}
				body, err := callout.Post(url, `{"key": "value"}`, client.UnmarshalJSONBody(&value))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(MatchJSON(`{"key":"value"}`))
				Expect(value.Key).To(Equal("value"))
			})
		})

		when("WriteBody", func() {
			it("writes the response body to the given writer but does not return it", func() {
				callout := client.New()

				url := server.URL + "/200"
				var buf bytes.Buffer
				body, err := callout.Post(url, "body", client.WriteBody(&buf))

				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(BeEmpty())
				Expect(buf.String()).To(Equal("200"))
			})
		})

		when("SkipTLSVerify", func() {
			it("does not skip TLS verification by default", func() {
				callout := client.New()

				_, err := callout.Post(tlsServer.URL, "")
				Expect(err).To(HaveOccurred())
			})

			it("skips TLS verification when set to true on the client", func() {
				callout := client.New(client.DefaultSkipTLSVerify(true))

				_, err := callout.Post(tlsServer.URL, "")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	when("Head", func() {
		it("returns the response body", func() {
			callout := client.New()

			url := server.URL + "/200"
			_, err := callout.Head(url)

			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a ResponseError when the response is not 200", func() {
			callout := client.New()

			url := server.URL + "/500"
			_, err := callout.Head(url)

			Expect(err).To(MatchError(client.ResponseError{
				URL:        server.URL + "/500",
				StatusCode: 500,
				Body:       []byte(""),
			}))
		})

		it("returns an error when something else goes wrong", func() {
			callout := client.New()

			body, err := callout.Head("this isn't a URL")

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(BeAssignableToTypeOf(client.ResponseError{}))
			Expect(body).To(BeEmpty())
		})

		when("WithTimeout", func() {
			it("returns an error if the request exceeds the client timeout", func() {
				timeout := time.Millisecond
				callout := client.New(client.WithDefaultTimeout(timeout))

				url := fmt.Sprintf("%s/sleep?duration=%s", server.URL, timeout*10)
				_, err := callout.Head(url)
				Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
			})
		})

		when("SkipTLSVerify", func() {
			it("does not skip TLS verification by default", func() {
				callout := client.New()

				_, err := callout.Head(tlsServer.URL)
				Expect(err).To(HaveOccurred())
			})

			it("skips TLS verification when set to true on the client", func() {
				callout := client.New(client.DefaultSkipTLSVerify(true))

				_, err := callout.Head(tlsServer.URL)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
}
