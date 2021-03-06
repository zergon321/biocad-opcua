package api

import (
	"log"
	"net/http"
)

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		wr.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(wr, req)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, ok := r.Context().Value(0).(string)

			if !ok {
				requestID = "unknown"
			}

			logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

			next.ServeHTTP(w, r)
		})
	}
}
