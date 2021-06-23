package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(app.notFountResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowResponse)

	router.HandlerFunc(
		http.MethodGet,
		"/v1/healthcheck",
		app.healthCheckHandler)

	router.HandlerFunc(
		http.MethodGet,
		"/v1/movies",
		app.requirePermission("movies:read", app.listMovieHandler))

	router.HandlerFunc(
		http.MethodPost,
		"/v1/movies",
		app.requirePermission("movies:write", app.createMovieHandler))

	router.HandlerFunc(
		http.MethodGet,
		"/v1/movies/:id",
		app.requirePermission("movies:read", app.showMovieHandler))

	router.HandlerFunc(
		http.MethodPatch,
		"/v1/movies/:id",
		app.requirePermission("movies:write", app.updateMovieHandler))

	router.HandlerFunc(
		http.MethodDelete,
		"/v1/movies/:id",
		app.requirePermission("movies:write", app.deleteMovieHandler))

	router.HandlerFunc(
		http.MethodPost,
		"/v1/users",
		app.registerUserHandler)

	router.HandlerFunc(
		http.MethodPut,
		"/v1/users/activated",
		app.activateUserHandler)

	router.HandlerFunc(
		http.MethodPut,
		"/v1/users/password",
		app.updateUserPasswordHandler)

	router.HandlerFunc(
		http.MethodPost,
		"/v1/tokens/authentication",
		app.createAuthenticationTokenHandler)

	router.HandlerFunc(
		http.MethodPost,
		"/v1/tokens/password-reset",
		app.createPwdResetTokenHandler)

	if app.config.metrics {
		router.Handler(
			http.MethodGet,
			"/debug/vars",
			expvar.Handler())
	}
	auMiddle := app.authenticate(router)
	rlMiddle := app.rateLimit(auMiddle)
	corsMiddle := app.enabledCORS(rlMiddle)
	recoverMiddle := app.recoverPanic(corsMiddle)
	return app.metrics(recoverMiddle)
}
