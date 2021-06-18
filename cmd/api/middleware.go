package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(middle)
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// initialize a gloabl rate limiter which allow an average of 20 request per second,
	// with a maximun of 25 requests in a single 'burst'
	globalLimiter := rate.NewLimiter(20, 25)
	middle := func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(middle)
}
