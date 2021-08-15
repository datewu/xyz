package main

import (
	"net/http"

	"github.com/datewu/toushi"
)

func setRoutes() http.Handler {
	r := toushi.New(nil)
	customRoutes(r)
	return r.Routes(authenticate)
}

func customRoutes(r *toushi.Router) {

	r.Get("/v1/movies",
		requirePermission("movies:read", listMovieHandler))

	r.Post("/v1/movies",
		requirePermission("movies:write", createMovieHandler))

	r.Get("/v1/movies/:id",
		requirePermission("movies:read", showMovieHandler))

	r.Patch("/v1/movies/:id",
		requirePermission("movies:write", updateMovieHandler))

	r.Delete("/v1/movies/:id",
		requirePermission("movies:write", deleteMovieHandler))

	r.Post("/v1/users",
		registerUserHandler)

	r.Put("/v1/users/activated",
		activateUserHandler)

	r.Put("/v1/users/password",
		updateUserPasswordHandler)

	r.Post("/v1/tokens/authentication",
		createAuthenticationTokenHandler)

	r.Post("/v1/tokens/activation",
		createActivationTokenHandler)

	r.Post("/v1/tokens/password-reset",
		createPwdResetTokenHandler)
}
