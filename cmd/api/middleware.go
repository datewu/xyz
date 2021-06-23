package main

import (
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
	"golang.org/x/time/rate"
)

func (app *application) enabledCORS(next http.Handler) http.Handler {
	if len(app.config.cors.trustedOrigins) == 0 {
		return next
	}
	app.logger.PrintInfo("enable cors middler", nil)
	middle := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")

		// Add the "Vary: Access-Control-Request-Method" header.
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")

		if origin != "" && len(app.config.cors.trustedOrigins) != 0 {
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// Check if the request has the HTTP method OPTIONS and contains the
					// "Access-Control-Request-Method" header. If it does, then we treat
					// it as a preflight request.
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// Set the necessary preflight response headers, as discussed
						// previously.
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						// Write the headers along with a 200 OK status and return from
						// the middleware with no further action.
						w.WriteHeader(http.StatusOK)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(middle)
}

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
	if !app.config.limiter.enabled {
		return next
	}
	app.logger.PrintInfo("enable ratelimit middler", nil)
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		clients = make(map[string]*client)
		mu      sync.Mutex
	)
	delOld := func(interval time.Duration) {
		for {
			time.Sleep(interval)
			mu.Lock()
			for k, v := range clients {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(clients, k)
				}
			}
			mu.Unlock()
		}
	}
	go delOld(time.Minute)

	middle := func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrResponse(w, r, err)
			return
		}
		mu.Lock()
		if _, existed := clients[ip]; !existed {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps),
					app.config.limiter.burst),
			}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(middle)
}

func (app *application) authenticate(next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		ah := r.Header.Get("Authorization")
		if ah == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		headerParts := strings.Split(ah, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrResponse(w, r, err)
			}
			return
		}
		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(middle)
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	middle := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequireResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
	return middle
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	middle := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
	return app.requireAuthenticatedUser(middle)
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	middle := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		ps, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrResponse(w, r, err)
			return
		}
		if !ps.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
	return app.requireActivatedUser(middle)
}

func (app *application) metrics(next http.Handler) http.Handler {
	if !app.config.metrics {
		return next
	}
	app.logger.PrintInfo("enable expvar metrics middler", nil)
	totalRequestReceived := expvar.NewInt("total_requests_received")
	totalResponsesSend := expvar.NewInt("total_responses_send")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_us")
	middle := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		totalRequestReceived.Add(1)
		next.ServeHTTP(w, r)
		totalResponsesSend.Add(1)
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	}
	return http.HandlerFunc(middle)
}
