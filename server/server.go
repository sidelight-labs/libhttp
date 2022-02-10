package server

import (
	"fmt"
	"github.com/sidelight-labs/libc/logger"
	"net/http"
)

func ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, logRequest(http.DefaultServeMux))
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Log(fmt.Sprintf("%s %s %s", r.RemoteAddr, r.Method, r.URL))
		handler.ServeHTTP(w, r)
	})
}
