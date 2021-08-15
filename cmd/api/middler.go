package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/datewu/toushi"
	"github.com/datewu/xyz/internal/data"
	"github.com/datewu/xyz/internal/validator"
)

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			r = contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			toushi.InvalidAuthenticationTokenResponse.ServeHTTP(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			toushi.InvalidAuthenticationTokenResponse.ServeHTTP(w, r)
			return
		}

		user, err := models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				toushi.InvalidAuthenticationTokenResponse.ServeHTTP(w, r)
			default:
				toushi.ServerErrResponse(err).ServeHTTP(w, r)
			}
			return
		}

		r = contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

func requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		if user.IsAnonymous() {
			toushi.AuthenticationRequireResponse.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		if !user.Activated {
			toushi.InactiveAccountResponse.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return requireAuthenticatedUser(fn)
}

func requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		permissions, err := models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			toushi.ServerErrResponse(err).ServeHTTP(w, r)
			return
		}

		if !permissions.Include(code) {
			toushi.NotPermittedResponse.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return requireActivatedUser(fn)
}
